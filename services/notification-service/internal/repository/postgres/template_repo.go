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
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

// TemplateRepo implements service.TemplateRepository using PostgreSQL.
type TemplateRepo struct {
	db *pgxpool.Pool
}

// NewTemplateRepo creates a new TemplateRepo.
func NewTemplateRepo(db *pgxpool.Pool) *TemplateRepo {
	return &TemplateRepo{db: db}
}

// Create inserts a new notification template.
func (r *TemplateRepo) Create(ctx context.Context, tmpl *models.NotificationTemplate) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	tmpl.ID = uuid.New()
	tmpl.TenantID = tenantID
	tmpl.CreatedAt = time.Now()
	tmpl.UpdatedAt = time.Now()
	if tmpl.Language == "" {
		tmpl.Language = "en"
	}
	if tmpl.TemplateDataSchema == nil {
		tmpl.TemplateDataSchema = []byte("{}")
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO notification_templates (id, tenant_id, name, channel, subject_template, body_template, template_data_schema, language, is_active, created_at, updated_at, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		tmpl.ID, tmpl.TenantID, tmpl.Name, tmpl.Channel, tmpl.SubjectTemplate,
		tmpl.BodyTemplate, tmpl.TemplateDataSchema, tmpl.Language, tmpl.IsActive,
		tmpl.CreatedAt, tmpl.UpdatedAt, tmpl.CreatedBy,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_templates_tenant_name_channel") {
			return errors.BadRequest("Template with this name and channel already exists", err)
		}
		return errors.InternalServerError("Failed to create template", err)
	}
	return nil
}

// GetByID retrieves a template by ID, scoped to tenant.
func (r *TemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.NotificationTemplate, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var tmpl models.NotificationTemplate
	err = r.db.QueryRow(ctx,
		`SELECT id, tenant_id, name, channel, subject_template, body_template, template_data_schema, language, is_active, created_at, updated_at, created_by
		 FROM notification_templates WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(
		&tmpl.ID, &tmpl.TenantID, &tmpl.Name, &tmpl.Channel, &tmpl.SubjectTemplate,
		&tmpl.BodyTemplate, &tmpl.TemplateDataSchema, &tmpl.Language, &tmpl.IsActive,
		&tmpl.CreatedAt, &tmpl.UpdatedAt, &tmpl.CreatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Template not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get template", err)
	}
	return &tmpl, nil
}

// GetByName retrieves a template by name and channel, scoped to tenant.
func (r *TemplateRepo) GetByName(ctx context.Context, name string, channel models.Channel) (*models.NotificationTemplate, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}

	var tmpl models.NotificationTemplate
	err = r.db.QueryRow(ctx,
		`SELECT id, tenant_id, name, channel, subject_template, body_template, template_data_schema, language, is_active, created_at, updated_at, created_by
		 FROM notification_templates WHERE tenant_id = $1 AND name = $2 AND channel = $3 AND is_active = true`,
		tenantID, name, channel,
	).Scan(
		&tmpl.ID, &tmpl.TenantID, &tmpl.Name, &tmpl.Channel, &tmpl.SubjectTemplate,
		&tmpl.BodyTemplate, &tmpl.TemplateDataSchema, &tmpl.Language, &tmpl.IsActive,
		&tmpl.CreatedAt, &tmpl.UpdatedAt, &tmpl.CreatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("Template not found", nil)
		}
		return nil, errors.InternalServerError("Failed to get template by name", err)
	}
	return &tmpl, nil
}

// List retrieves templates with filtering and pagination, scoped to tenant.
func (r *TemplateRepo) List(ctx context.Context, filter models.TemplateFilter, page models.Pagination) ([]models.NotificationTemplate, int, error) {
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

	if filter.Channel != nil {
		conditions = append(conditions, fmt.Sprintf("channel = $%d", argIdx))
		args = append(args, *filter.Channel)
		argIdx++
	}

	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}

	if filter.Language != nil {
		conditions = append(conditions, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, *filter.Language)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notification_templates WHERE %s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.InternalServerError("Failed to count templates", err)
	}

	// Fetch page
	offset := (page.Page - 1) * page.PageSize
	listQuery := fmt.Sprintf(
		`SELECT id, tenant_id, name, channel, subject_template, body_template, template_data_schema, language, is_active, created_at, updated_at, created_by
		 FROM notification_templates WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, page.PageSize, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, errors.InternalServerError("Failed to list templates", err)
	}
	defer rows.Close()

	var templates []models.NotificationTemplate
	for rows.Next() {
		var tmpl models.NotificationTemplate
		if err := rows.Scan(
			&tmpl.ID, &tmpl.TenantID, &tmpl.Name, &tmpl.Channel, &tmpl.SubjectTemplate,
			&tmpl.BodyTemplate, &tmpl.TemplateDataSchema, &tmpl.Language, &tmpl.IsActive,
			&tmpl.CreatedAt, &tmpl.UpdatedAt, &tmpl.CreatedBy,
		); err != nil {
			return nil, 0, errors.InternalServerError("Failed to scan template", err)
		}
		templates = append(templates, tmpl)
	}

	return templates, total, nil
}

// Update performs a partial update on a template, scoped to tenant.
func (r *TemplateRepo) Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}

	if len(fields) == 0 {
		return nil
	}

	var setClauses []string
	var args []interface{}
	argIdx := 1

	for key, value := range fields {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
		args = append(args, value)
		argIdx++
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, id, tenantID)

	query := fmt.Sprintf(
		"UPDATE notification_templates SET %s WHERE id = $%d AND tenant_id = $%d",
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return errors.InternalServerError("Failed to update template", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Template not found", nil)
	}
	return nil
}

// Delete performs a soft delete on a template (set is_active=false), scoped to tenant.
func (r *TemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.Update(ctx, id, map[string]interface{}{"is_active": false})
}
