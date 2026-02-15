package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
)

// OrgRepository defines the interface for organization data access.
type OrgRepository interface {
	Create(ctx context.Context, org *models.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	GetByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*models.Organization, error)
	Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter models.OrgFilter, page models.Pagination) ([]models.Organization, int, error)
	CountByOwnerID(ctx context.Context, ownerID uuid.UUID) (int, error)
}

// ModuleRepository defines the interface for module data access.
type ModuleRepository interface {
	GetByOrg(ctx context.Context, orgID uuid.UUID) ([]models.Module, error)
	GetByOrgUnscoped(ctx context.Context, orgID uuid.UUID) ([]models.Module, error)
	Toggle(ctx context.Context, orgID uuid.UUID, moduleName string, enabled bool) error
	GetConfig(ctx context.Context, orgID uuid.UUID, moduleName string) (*models.Module, error)
	UpdateConfig(ctx context.Context, orgID uuid.UUID, moduleName string, config []byte) error
}

// MemberRepository defines the interface for member data access.
type MemberRepository interface {
	Add(ctx context.Context, orgID uuid.UUID, member *models.Member) error
	List(ctx context.Context, orgID uuid.UUID, filter models.MemberFilter, page models.Pagination) ([]models.Member, int, error)
	Remove(ctx context.Context, orgID, userID uuid.UUID) error
	UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role string) error
	CountActive(ctx context.Context, orgID uuid.UUID) (int, error)
}

// PlanRepository defines the interface for plan data access.
type PlanRepository interface {
	GetByName(ctx context.Context, name string) (*models.Plan, error)
	List(ctx context.Context) ([]models.Plan, error)
}

// DeptRepository defines the interface for department data access.
type DeptRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, dept *models.Department) error
	List(ctx context.Context, orgID uuid.UUID) ([]models.Department, error)
	Update(ctx context.Context, orgID, deptID uuid.UUID, fields map[string]interface{}) error
	Delete(ctx context.Context, orgID, deptID uuid.UUID) error
	GetByID(ctx context.Context, orgID, deptID uuid.UUID) (*models.Department, error)
}

// ModuleCache defines the interface for Redis-based module caching.
type ModuleCache interface {
	GetModules(ctx context.Context, orgID uuid.UUID) ([]models.ModuleResponse, error)
	SetModules(ctx context.Context, orgID uuid.UUID, modules []models.ModuleResponse) error
	Invalidate(ctx context.Context, orgID uuid.UUID) error
}

// EventPublisher defines the interface for publishing domain events.
type EventPublisher interface {
	Publish(ctx context.Context, stream string, eventType string, data interface{}) error
}
