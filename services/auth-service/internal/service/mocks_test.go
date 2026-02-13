package service

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/services/auth-service/internal/models"
)

// mockUserRepo is a test double for UserRepository.
type mockUserRepo struct {
	mu    sync.RWMutex
	users map[uuid.UUID]*models.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[uuid.UUID]*models.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Check for duplicate email within tenant
	for _, u := range m.users {
		if u.Email == user.Email && u.TenantID == user.TenantID {
			return &duplicateEmailError{}
		}
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, &notFoundError{}
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, &notFoundError{}
}

func (m *mockUserRepo) List(_ context.Context, _ models.UserFilter, page models.Pagination) ([]models.User, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []models.User
	for _, u := range m.users {
		result = append(result, *u)
	}
	total := len(result)
	start := (page.Page - 1) * page.PageSize
	if start >= total {
		return nil, total, nil
	}
	end := start + page.PageSize
	if end > total {
		end = total
	}
	return result[start:end], total, nil
}

func (m *mockUserRepo) Update(_ context.Context, id uuid.UUID, fields map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return &notFoundError{}
	}
	if v, ok := fields["first_name"]; ok {
		u.FirstName = v.(string)
	}
	if v, ok := fields["last_name"]; ok {
		u.LastName = v.(string)
	}
	if v, ok := fields["status"]; ok {
		u.Status = v.(models.UserStatus)
	}
	if v, ok := fields["password_hash"]; ok {
		u.PasswordHash = v.(string)
	}
	if v, ok := fields["mfa_enabled"]; ok {
		u.MFAEnabled = v.(bool)
	}
	if v, ok := fields["mfa_secret"]; ok {
		u.MFASecret = v.(string)
	}
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[id]; ok {
		u.Status = models.UserStatusDisabled
		return nil
	}
	return &notFoundError{}
}

func (m *mockUserRepo) IncrementFailedLogins(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[id]; ok {
		u.FailedLoginCount++
		return nil
	}
	return nil
}

func (m *mockUserRepo) ResetFailedLogins(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[id]; ok {
		u.FailedLoginCount = 0
		u.LockedUntil = nil
		return nil
	}
	return nil
}

func (m *mockUserRepo) UpdateLastLogin(_ context.Context, _ uuid.UUID) error {
	return nil
}

// mockSessionRepo is a test double for SessionRepository.
type mockSessionRepo struct {
	mu       sync.RWMutex
	sessions map[uuid.UUID]*models.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{sessions: make(map[uuid.UUID]*models.Session)}
}

func (m *mockSessionRepo) Create(_ context.Context, s *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionRepo) GetByRefreshToken(_ context.Context, tokenHash string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.sessions {
		if s.RefreshTokenHash == tokenHash {
			return s, nil
		}
	}
	return nil, &notFoundError{}
}

func (m *mockSessionRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []models.Session
	for _, s := range m.sessions {
		if s.UserID == userID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockSessionRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

func (m *mockSessionRepo) DeleteAllByUser(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockSessionRepo) CleanupExpired(_ context.Context) (int64, error) {
	return 0, nil
}

// mockRoleRepo is a test double for RoleRepository.
type mockRoleRepo struct {
	roles     map[models.RoleName]*models.Role
	userRoles map[uuid.UUID][]models.UserRole
}

func newMockRoleRepo() *mockRoleRepo {
	memberRoleID := uuid.New()
	return &mockRoleRepo{
		roles: map[models.RoleName]*models.Role{
			models.RoleMember: {
				ID:       memberRoleID,
				Name:     models.RoleMember,
				IsSystem: true,
			},
			models.RoleOrgOwner: {
				ID:       uuid.New(),
				Name:     models.RoleOrgOwner,
				IsSystem: true,
			},
		},
		userRoles: make(map[uuid.UUID][]models.UserRole),
	}
}

func (m *mockRoleRepo) GetByName(_ context.Context, name models.RoleName) (*models.Role, error) {
	if r, ok := m.roles[name]; ok {
		return r, nil
	}
	return nil, &notFoundError{}
}

func (m *mockRoleRepo) ListRoles(_ context.Context) ([]models.Role, error) {
	var result []models.Role
	for _, r := range m.roles {
		result = append(result, *r)
	}
	return result, nil
}

func (m *mockRoleRepo) AssignRole(_ context.Context, userID, roleID uuid.UUID, grantedBy *uuid.UUID) error {
	for name, role := range m.roles {
		if role.ID == roleID {
			m.userRoles[userID] = append(m.userRoles[userID], models.UserRole{
				UserID:   userID,
				RoleID:   roleID,
				RoleName: name,
			})
			break
		}
	}
	return nil
}

func (m *mockRoleRepo) RevokeRole(_ context.Context, userID, roleID uuid.UUID) error {
	roles := m.userRoles[userID]
	for i, ur := range roles {
		if ur.RoleID == roleID {
			m.userRoles[userID] = append(roles[:i], roles[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockRoleRepo) GetUserRoles(_ context.Context, userID uuid.UUID) ([]models.UserRole, error) {
	return m.userRoles[userID], nil
}

func (m *mockRoleRepo) GetUserPermissions(_ context.Context, _ uuid.UUID) ([]models.Permission, error) {
	return nil, nil
}

// mockPasswordHistoryRepo is a test double for PasswordHistoryRepository.
type mockPasswordHistoryRepo struct {
	history map[uuid.UUID][]string
}

func newMockPasswordHistoryRepo() *mockPasswordHistoryRepo {
	return &mockPasswordHistoryRepo{history: make(map[uuid.UUID][]string)}
}

func (m *mockPasswordHistoryRepo) Add(_ context.Context, userID uuid.UUID, hash string) error {
	m.history[userID] = append(m.history[userID], hash)
	return nil
}

func (m *mockPasswordHistoryRepo) GetRecent(_ context.Context, userID uuid.UUID, count int) ([]string, error) {
	hashes := m.history[userID]
	if len(hashes) > count {
		return hashes[len(hashes)-count:], nil
	}
	return hashes, nil
}

// mockTokenCache is a test double for TokenCache.
type mockTokenCache struct {
	mu          sync.RWMutex
	blacklisted map[string]bool
	sessions    map[string]*models.Session
	mfaTokens   map[string]string
}

func newMockTokenCache() *mockTokenCache {
	return &mockTokenCache{
		blacklisted: make(map[string]bool),
		sessions:    make(map[string]*models.Session),
		mfaTokens:   make(map[string]string),
	}
}

func (m *mockTokenCache) BlacklistToken(_ context.Context, tokenID string, _ int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blacklisted[tokenID] = true
	return nil
}

func (m *mockTokenCache) IsBlacklisted(_ context.Context, tokenID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.blacklisted[tokenID], nil
}

func (m *mockTokenCache) CacheSession(_ context.Context, id string, s *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[id] = s
	return nil
}

func (m *mockTokenCache) GetCachedSession(_ context.Context, id string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.sessions[id]; ok {
		return s, nil
	}
	return nil, &notFoundError{}
}

func (m *mockTokenCache) InvalidateSession(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

func (m *mockTokenCache) StoreMFAToken(_ context.Context, userID, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mfaTokens[userID] = secret
	return nil
}

func (m *mockTokenCache) GetMFAToken(_ context.Context, userID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.mfaTokens[userID]; ok {
		return s, nil
	}
	return "", &notFoundError{}
}

func (m *mockTokenCache) DeleteMFAToken(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.mfaTokens, userID)
	return nil
}

// mockEventPublisher is a test double for EventPublisher.
type mockEventPublisher struct {
	mu     sync.Mutex
	events []publishedEvent
}

type publishedEvent struct {
	Stream    string
	EventType string
	Data      interface{}
}

func newMockEventPublisher() *mockEventPublisher {
	return &mockEventPublisher{}
}

func (m *mockEventPublisher) Publish(_ context.Context, stream, eventType string, data interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, publishedEvent{Stream: stream, EventType: eventType, Data: data})
	return nil
}

func (m *mockEventPublisher) lastEvent() *publishedEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.events) == 0 {
		return nil
	}
	return &m.events[len(m.events)-1]
}

// Error types for mocks
type notFoundError struct{}

func (e *notFoundError) Error() string { return "not found" }

type duplicateEmailError struct{}

func (e *duplicateEmailError) Error() string { return "duplicate email" }
