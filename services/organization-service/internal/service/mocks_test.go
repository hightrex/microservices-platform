package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
)

// ensure fmt is used
var _ = fmt.Sprintf

// --- Mock OrgRepository ---

type mockOrgRepo struct {
	createFn          func(ctx context.Context, org *models.Organization) error
	getByIDFn         func(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	getByIDUnscopedFn func(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	getBySlugFn       func(ctx context.Context, slug string) (*models.Organization, error)
	updateFn          func(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	deleteFn          func(ctx context.Context, id uuid.UUID) error
	listFn            func(ctx context.Context, filter models.OrgFilter, page models.Pagination) ([]models.Organization, int, error)
	countByOwnerIDFn  func(ctx context.Context, ownerID uuid.UUID) (int, error)
}

func (m *mockOrgRepo) Create(ctx context.Context, org *models.Organization) error {
	if m.createFn != nil {
		return m.createFn(ctx, org)
	}
	org.ID = uuid.New()
	return nil
}

func (m *mockOrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockOrgRepo) GetByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	if m.getByIDUnscopedFn != nil {
		return m.getByIDUnscopedFn(ctx, id)
	}
	return nil, nil
}

func (m *mockOrgRepo) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	if m.getBySlugFn != nil {
		return m.getBySlugFn(ctx, slug)
	}
	return nil, nil
}

func (m *mockOrgRepo) Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, fields)
	}
	return nil
}

func (m *mockOrgRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockOrgRepo) List(ctx context.Context, filter models.OrgFilter, page models.Pagination) ([]models.Organization, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter, page)
	}
	return nil, 0, nil
}

func (m *mockOrgRepo) CountByOwnerID(ctx context.Context, ownerID uuid.UUID) (int, error) {
	if m.countByOwnerIDFn != nil {
		return m.countByOwnerIDFn(ctx, ownerID)
	}
	return 0, nil
}

// --- Mock ModuleRepository ---

type mockModuleRepo struct {
	getByOrgFn         func(ctx context.Context, orgID uuid.UUID) ([]models.Module, error)
	getByOrgUnscopedFn func(ctx context.Context, orgID uuid.UUID) ([]models.Module, error)
	toggleFn           func(ctx context.Context, orgID uuid.UUID, moduleName string, enabled bool) error
	getConfigFn        func(ctx context.Context, orgID uuid.UUID, moduleName string) (*models.Module, error)
	updateConfigFn     func(ctx context.Context, orgID uuid.UUID, moduleName string, config []byte) error
}

func (m *mockModuleRepo) GetByOrg(ctx context.Context, orgID uuid.UUID) ([]models.Module, error) {
	if m.getByOrgFn != nil {
		return m.getByOrgFn(ctx, orgID)
	}
	return nil, nil
}

func (m *mockModuleRepo) GetByOrgUnscoped(ctx context.Context, orgID uuid.UUID) ([]models.Module, error) {
	if m.getByOrgUnscopedFn != nil {
		return m.getByOrgUnscopedFn(ctx, orgID)
	}
	return nil, nil
}

func (m *mockModuleRepo) Toggle(ctx context.Context, orgID uuid.UUID, moduleName string, enabled bool) error {
	if m.toggleFn != nil {
		return m.toggleFn(ctx, orgID, moduleName, enabled)
	}
	return nil
}

func (m *mockModuleRepo) GetConfig(ctx context.Context, orgID uuid.UUID, moduleName string) (*models.Module, error) {
	if m.getConfigFn != nil {
		return m.getConfigFn(ctx, orgID, moduleName)
	}
	return nil, nil
}

func (m *mockModuleRepo) UpdateConfig(ctx context.Context, orgID uuid.UUID, moduleName string, config []byte) error {
	if m.updateConfigFn != nil {
		return m.updateConfigFn(ctx, orgID, moduleName, config)
	}
	return nil
}

// --- Mock MemberRepository ---

type mockMemberRepo struct {
	addFn         func(ctx context.Context, orgID uuid.UUID, member *models.Member) error
	listFn        func(ctx context.Context, orgID uuid.UUID, filter models.MemberFilter, page models.Pagination) ([]models.Member, int, error)
	removeFn      func(ctx context.Context, orgID, userID uuid.UUID) error
	updateRoleFn  func(ctx context.Context, orgID, userID uuid.UUID, role string) error
	countActiveFn func(ctx context.Context, orgID uuid.UUID) (int, error)
}

func (m *mockMemberRepo) Add(ctx context.Context, orgID uuid.UUID, member *models.Member) error {
	if m.addFn != nil {
		return m.addFn(ctx, orgID, member)
	}
	member.ID = uuid.New()
	return nil
}

func (m *mockMemberRepo) List(ctx context.Context, orgID uuid.UUID, filter models.MemberFilter, page models.Pagination) ([]models.Member, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID, filter, page)
	}
	return nil, 0, nil
}

func (m *mockMemberRepo) Remove(ctx context.Context, orgID, userID uuid.UUID) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, orgID, userID)
	}
	return nil
}

func (m *mockMemberRepo) UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	if m.updateRoleFn != nil {
		return m.updateRoleFn(ctx, orgID, userID, role)
	}
	return nil
}

func (m *mockMemberRepo) CountActive(ctx context.Context, orgID uuid.UUID) (int, error) {
	if m.countActiveFn != nil {
		return m.countActiveFn(ctx, orgID)
	}
	return 0, nil
}

// --- Mock PlanRepository ---

type mockPlanRepo struct {
	getByNameFn func(ctx context.Context, name string) (*models.Plan, error)
	listFn      func(ctx context.Context) ([]models.Plan, error)
}

func (m *mockPlanRepo) GetByName(ctx context.Context, name string) (*models.Plan, error) {
	if m.getByNameFn != nil {
		return m.getByNameFn(ctx, name)
	}
	return &models.Plan{
		ID:               uuid.New(),
		Name:             name,
		MaxUsers:         5,
		MaxStorageBytes:  1073741824,
		AvailableModules: []string{"audit_logging"},
	}, nil
}

func (m *mockPlanRepo) List(ctx context.Context) ([]models.Plan, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

// --- Mock DeptRepository ---

type mockDeptRepo struct {
	createFn  func(ctx context.Context, orgID uuid.UUID, dept *models.Department) error
	listFn    func(ctx context.Context, orgID uuid.UUID) ([]models.Department, error)
	updateFn  func(ctx context.Context, orgID, deptID uuid.UUID, fields map[string]interface{}) error
	deleteFn  func(ctx context.Context, orgID, deptID uuid.UUID) error
	getByIDFn func(ctx context.Context, orgID, deptID uuid.UUID) (*models.Department, error)
}

func (m *mockDeptRepo) Create(ctx context.Context, orgID uuid.UUID, dept *models.Department) error {
	if m.createFn != nil {
		return m.createFn(ctx, orgID, dept)
	}
	dept.ID = uuid.New()
	return nil
}

func (m *mockDeptRepo) List(ctx context.Context, orgID uuid.UUID) ([]models.Department, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID)
	}
	return nil, nil
}

func (m *mockDeptRepo) Update(ctx context.Context, orgID, deptID uuid.UUID, fields map[string]interface{}) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, orgID, deptID, fields)
	}
	return nil
}

func (m *mockDeptRepo) Delete(ctx context.Context, orgID, deptID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, orgID, deptID)
	}
	return nil
}

func (m *mockDeptRepo) GetByID(ctx context.Context, orgID, deptID uuid.UUID) (*models.Department, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, deptID)
	}
	return nil, nil
}

// --- Mock ModuleCache ---

type mockModuleCache struct {
	getModulesFn func(ctx context.Context, orgID uuid.UUID) ([]models.ModuleResponse, error)
	setModulesFn func(ctx context.Context, orgID uuid.UUID, modules []models.ModuleResponse) error
	invalidateFn func(ctx context.Context, orgID uuid.UUID) error
}

func (m *mockModuleCache) GetModules(ctx context.Context, orgID uuid.UUID) ([]models.ModuleResponse, error) {
	if m.getModulesFn != nil {
		return m.getModulesFn(ctx, orgID)
	}
	return nil, fmt.Errorf("cache miss")
}

func (m *mockModuleCache) SetModules(ctx context.Context, orgID uuid.UUID, modules []models.ModuleResponse) error {
	if m.setModulesFn != nil {
		return m.setModulesFn(ctx, orgID, modules)
	}
	return nil
}

func (m *mockModuleCache) Invalidate(ctx context.Context, orgID uuid.UUID) error {
	if m.invalidateFn != nil {
		return m.invalidateFn(ctx, orgID)
	}
	return nil
}

// --- Mock EventPublisher ---

type mockEventPublisher struct {
	publishFn func(ctx context.Context, stream string, eventType string, data interface{}) error
	events    []publishedEvent
}

type publishedEvent struct {
	Stream    string
	EventType string
	Data      interface{}
}

func (m *mockEventPublisher) Publish(ctx context.Context, stream string, eventType string, data interface{}) error {
	m.events = append(m.events, publishedEvent{Stream: stream, EventType: eventType, Data: data})
	if m.publishFn != nil {
		return m.publishFn(ctx, stream, eventType, data)
	}
	return nil
}
