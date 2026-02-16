package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/audit-service/internal/models"
)

// AuditRepo implements service.AuditRepository using PostgreSQL.
type AuditRepo struct {
	db *pgxpool.Pool
}

// NewAuditRepo creates a new AuditRepo.
func NewAuditRepo(db *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{db: db}
}

// Create inserts a new audit log entry (append-only).
func (r *AuditRepo) Create(ctx context.Context, log *models.AuditLog) error {
	log.ID = uuid.New()
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}
	if log.Metadata == nil {
		log.Metadata = []byte("{}")
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO audit_logs (id, tenant_id, event_type, event_category, actor_id, actor_type, resource_type, resource_id, action, outcome, ip_address, user_agent, metadata, timestamp, previous_hash, current_hash)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		log.ID, log.TenantID, log.EventType, log.EventCategory, log.ActorID,
		log.ActorType, log.ResourceType, log.ResourceID, log.Action, log.Outcome,
		log.IPAddress, log.UserAgent, log.Metadata, log.Timestamp,
		log.PreviousHash, log.CurrentHash,
	)
	if err != nil {
		return errors.InternalServerError("Failed to create audit log", err)
	}
	return nil
}

// GetByID retrieves an audit log by ID, scoped to tenant.
func (r *AuditRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var log models.AuditLog
	err = r.db.QueryRow(ctx,
		`SELECT id, tenant_id, event_type, event_category, actor_id, actor_type, resource_type, resource_id, action, outcome, ip_address, user_agent, metadata, timestamp, previous_hash, current_hash
		 FROM audit_logs WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(
		&log.ID, &log.TenantID, &log.EventType, &log.EventCategory, &log.ActorID,
		&log.ActorType, &log.ResourceType, &log.ResourceID, &log.Action, &log.Outcome,
		&log.IPAddress, &log.UserAgent, &log.Metadata, &log.Timestamp,
		&log.PreviousHash, &log.CurrentHash,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Audit log not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get audit log", err)
	}
	return &log, nil
}

// Search retrieves audit logs with advanced filtering and pagination, scoped to tenant.
func (r *AuditRepo) Search(ctx context.Context, filter models.SearchRequest, page models.Pagination) ([]models.AuditLog, int, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, 0, err
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.EventType != nil {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIdx))
		args = append(args, *filter.EventType)
		argIdx++
	}

	if filter.EventCategory != nil {
		conditions = append(conditions, fmt.Sprintf("event_category = $%d", argIdx))
		args = append(args, *filter.EventCategory)
		argIdx++
	}

	if filter.ActorID != nil {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argIdx))
		args = append(args, *filter.ActorID)
		argIdx++
	}

	if filter.ResourceType != nil {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", argIdx))
		args = append(args, *filter.ResourceType)
		argIdx++
	}

	if filter.ResourceID != nil {
		conditions = append(conditions, fmt.Sprintf("resource_id = $%d", argIdx))
		args = append(args, *filter.ResourceID)
		argIdx++
	}

	if filter.Outcome != nil {
		conditions = append(conditions, fmt.Sprintf("outcome = $%d", argIdx))
		args = append(args, *filter.Outcome)
		argIdx++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs WHERE %s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.InternalServerError("Failed to count audit logs", err)
	}

	// Fetch page
	offset := (page.Page - 1) * page.PageSize
	listQuery := fmt.Sprintf(
		`SELECT id, tenant_id, event_type, event_category, actor_id, actor_type, resource_type, resource_id, action, outcome, ip_address, user_agent, metadata, timestamp, previous_hash, current_hash
		 FROM audit_logs WHERE %s ORDER BY timestamp DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, page.PageSize, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, errors.InternalServerError("Failed to search audit logs", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.EventType, &l.EventCategory, &l.ActorID,
			&l.ActorType, &l.ResourceType, &l.ResourceID, &l.Action, &l.Outcome,
			&l.IPAddress, &l.UserAgent, &l.Metadata, &l.Timestamp,
			&l.PreviousHash, &l.CurrentHash,
		); err != nil {
			return nil, 0, errors.InternalServerError("Failed to scan audit log", err)
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}

// GetLastHash retrieves the last hash for a tenant's audit chain.
func (r *AuditRepo) GetLastHash(ctx context.Context, tenantID uuid.UUID) (*string, error) {
	var hash string
	err := r.db.QueryRow(ctx,
		`SELECT current_hash FROM audit_logs WHERE tenant_id = $1 ORDER BY timestamp DESC LIMIT 1`,
		tenantID,
	).Scan(&hash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No logs yet — first entry
		}
		return nil, errors.InternalServerError("Failed to get last hash", err)
	}
	return &hash, nil
}

// GetLogsForVerification retrieves logs in order for hash chain verification.
func (r *AuditRepo) GetLogsForVerification(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]models.AuditLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, tenant_id, event_type, event_category, actor_id, actor_type, resource_type, resource_id, action, outcome, ip_address, user_agent, metadata, timestamp, previous_hash, current_hash
		 FROM audit_logs WHERE tenant_id = $1 AND timestamp >= $2 AND timestamp <= $3
		 ORDER BY timestamp ASC`,
		tenantID, start, end,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to get logs for verification", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.EventType, &l.EventCategory, &l.ActorID,
			&l.ActorType, &l.ResourceType, &l.ResourceID, &l.Action, &l.Outcome,
			&l.IPAddress, &l.UserAgent, &l.Metadata, &l.Timestamp,
			&l.PreviousHash, &l.CurrentHash,
		); err != nil {
			return nil, errors.InternalServerError("Failed to scan audit log for verification", err)
		}
		logs = append(logs, l)
	}

	return logs, nil
}

// GetStatistics retrieves event count statistics for a tenant.
func (r *AuditRepo) GetStatistics(ctx context.Context, tenantID uuid.UUID) (*models.Statistics, error) {
	stats := &models.Statistics{
		ByEventType: make(map[string]int),
		ByCategory:  make(map[models.EventCategory]int),
		ByOutcome:   make(map[models.Outcome]int),
	}

	// Total count
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1", tenantID,
	).Scan(&stats.TotalLogs); err != nil {
		return nil, errors.InternalServerError("Failed to get total logs count", err)
	}

	// By event type
	rows, err := r.db.Query(ctx,
		"SELECT event_type, COUNT(*) FROM audit_logs WHERE tenant_id = $1 GROUP BY event_type ORDER BY COUNT(*) DESC LIMIT 20", tenantID)
	if err != nil {
		return nil, errors.InternalServerError("Failed to get event type stats", err)
	}
	defer rows.Close()
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, errors.InternalServerError("Failed to scan event type stats", err)
		}
		stats.ByEventType[eventType] = count
	}

	// By category
	rows2, err := r.db.Query(ctx,
		"SELECT event_category, COUNT(*) FROM audit_logs WHERE tenant_id = $1 GROUP BY event_category", tenantID)
	if err != nil {
		return nil, errors.InternalServerError("Failed to get category stats", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var cat models.EventCategory
		var count int
		if err := rows2.Scan(&cat, &count); err != nil {
			return nil, errors.InternalServerError("Failed to scan category stats", err)
		}
		stats.ByCategory[cat] = count
	}

	// By outcome
	rows3, err := r.db.Query(ctx,
		"SELECT outcome, COUNT(*) FROM audit_logs WHERE tenant_id = $1 GROUP BY outcome", tenantID)
	if err != nil {
		return nil, errors.InternalServerError("Failed to get outcome stats", err)
	}
	defer rows3.Close()
	for rows3.Next() {
		var outcome models.Outcome
		var count int
		if err := rows3.Scan(&outcome, &count); err != nil {
			return nil, errors.InternalServerError("Failed to scan outcome stats", err)
		}
		stats.ByOutcome[outcome] = count
	}

	return stats, nil
}

// PurgeExpiredByEventType deletes audit logs older than the given cutoff date
// for a specific event type and tenant. Used exclusively by the retention enforcement worker.
// Runs inside a transaction with a session variable to bypass the immutability trigger.
func (r *AuditRepo) PurgeExpiredByEventType(ctx context.Context, tenantID uuid.UUID, eventType string, olderThan time.Time) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, errors.InternalServerError("Failed to begin purge transaction", err)
	}
	defer tx.Rollback(ctx)

	// Enable the retention bypass flag for this transaction only
	if _, err := tx.Exec(ctx, "SET LOCAL app.retention_purge = 'true'"); err != nil {
		return 0, errors.InternalServerError("Failed to set retention purge flag", err)
	}

	tag, err := tx.Exec(ctx,
		`DELETE FROM audit_logs WHERE tenant_id = $1 AND event_type = $2 AND timestamp < $3`,
		tenantID, eventType, olderThan,
	)
	if err != nil {
		return 0, errors.InternalServerError("Failed to purge expired audit logs", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, errors.InternalServerError("Failed to commit purge transaction", err)
	}

	return tag.RowsAffected(), nil
}
