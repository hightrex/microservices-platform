package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
)

// MemberRepo implements service.MemberRepository using PostgreSQL.
type MemberRepo struct {
	db *pgxpool.Pool
}

// NewMemberRepo creates a new MemberRepo.
func NewMemberRepo(db *pgxpool.Pool) *MemberRepo {
	return &MemberRepo{db: db}
}

// Add invites a member to an organization.
func (r *MemberRepo) Add(ctx context.Context, orgID uuid.UUID, member *models.Member) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}
	if tenantID != orgID {
		return errors.Forbidden("Access denied to this organization", nil)
	}

	member.ID = uuid.New()
	member.OrgID = orgID
	member.CreatedAt = time.Now()
	member.UpdatedAt = time.Now()
	if member.Status == "" {
		member.Status = models.MemberStatusInvited
	}
	now := time.Now()
	member.InvitedAt = &now

	_, err = r.db.Exec(ctx,
		`INSERT INTO org_members (id, org_id, user_id, role, invited_by, invited_at, joined_at, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		member.ID, member.OrgID, member.UserID, member.Role,
		member.InvitedBy, member.InvitedAt, member.JoinedAt,
		member.Status, member.CreatedAt, member.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_org_members_org_user") {
			return errors.BadRequest("User is already a member of this organization", err)
		}
		return errors.InternalServerError("Failed to add member", err)
	}
	return nil
}

// List retrieves members of an organization with filtering and pagination.
func (r *MemberRepo) List(ctx context.Context, orgID uuid.UUID, filter models.MemberFilter, page models.Pagination) ([]models.Member, int, error) {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return nil, 0, err
	}
	if tenantID != orgID {
		return nil, 0, errors.Forbidden("Access denied to this organization", nil)
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("org_id = $%d", argIdx))
	args = append(args, orgID)
	argIdx++

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.Role != nil {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, *filter.Role)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM org_members WHERE %s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.InternalServerError("Failed to count members", err)
	}

	// Fetch page
	offset := (page.Page - 1) * page.PageSize
	listQuery := fmt.Sprintf(
		`SELECT id, org_id, user_id, role, invited_by, invited_at, joined_at, status, created_at, updated_at
		 FROM org_members WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, page.PageSize, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, errors.InternalServerError("Failed to list members", err)
	}
	defer rows.Close()

	var members []models.Member
	for rows.Next() {
		var m models.Member
		if err := rows.Scan(
			&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.InvitedBy,
			&m.InvitedAt, &m.JoinedAt, &m.Status, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, 0, errors.InternalServerError("Failed to scan member", err)
		}
		members = append(members, m)
	}

	return members, total, nil
}

// Remove sets a member's status to removed.
func (r *MemberRepo) Remove(ctx context.Context, orgID, userID uuid.UUID) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}
	if tenantID != orgID {
		return errors.Forbidden("Access denied to this organization", nil)
	}

	tag, err := r.db.Exec(ctx,
		`UPDATE org_members SET status = $1, updated_at = $2 WHERE org_id = $3 AND user_id = $4 AND status != 'removed'`,
		models.MemberStatusRemoved, time.Now(), orgID, userID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to remove member", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Member not found", nil)
	}
	return nil
}

// UpdateRole updates a member's role within the organization.
func (r *MemberRepo) UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	tenantID, err := tenant.RequireTenant(ctx)
	if err != nil {
		return err
	}
	if tenantID != orgID {
		return errors.Forbidden("Access denied to this organization", nil)
	}

	tag, err := r.db.Exec(ctx,
		`UPDATE org_members SET role = $1, updated_at = $2 WHERE org_id = $3 AND user_id = $4 AND status = 'active'`,
		role, time.Now(), orgID, userID,
	)
	if err != nil {
		return errors.InternalServerError("Failed to update member role", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("Active member not found", nil)
	}
	return nil
}

// CountActive returns the count of active members in an organization.
func (r *MemberRepo) CountActive(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM org_members WHERE org_id = $1 AND status IN ('active', 'invited')`,
		orgID,
	).Scan(&count)
	if err != nil {
		return 0, errors.InternalServerError("Failed to count members", err)
	}
	return count, nil
}
