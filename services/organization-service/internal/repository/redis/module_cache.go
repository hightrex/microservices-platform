package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/cache"
	"github.com/hightrex/microservices-platform/services/organization-service/internal/models"
)

const (
	modulePrefix     = "org:modules:"
	moduleCacheTTL   = 5 * time.Minute
)

// ModuleCache provides Redis-based caching for organization module configurations.
// This is the hot path used by the API Gateway to check if a module is enabled.
type ModuleCache struct {
	client *cache.Client
}

// NewModuleCache creates a new ModuleCache.
func NewModuleCache(client *cache.Client) *ModuleCache {
	return &ModuleCache{client: client}
}

// CachedModules represents the cached module state for an org.
type CachedModules struct {
	Modules []models.ModuleResponse `json:"modules"`
}

// GetModules retrieves cached modules for an organization.
// Returns nil if not cached.
func (c *ModuleCache) GetModules(ctx context.Context, orgID uuid.UUID) ([]models.ModuleResponse, error) {
	key := fmt.Sprintf("%s%s", modulePrefix, orgID.String())
	var cached CachedModules
	err := c.client.Get(ctx, key, &cached)
	if err != nil {
		return nil, err
	}
	return cached.Modules, nil
}

// SetModules caches modules for an organization with a 5-minute TTL.
func (c *ModuleCache) SetModules(ctx context.Context, orgID uuid.UUID, modules []models.ModuleResponse) error {
	key := fmt.Sprintf("%s%s", modulePrefix, orgID.String())
	return c.client.Set(ctx, key, CachedModules{Modules: modules}, moduleCacheTTL)
}

// Invalidate removes cached modules for an organization.
// Called when a module is toggled or config is changed.
func (c *ModuleCache) Invalidate(ctx context.Context, orgID uuid.UUID) error {
	key := fmt.Sprintf("%s%s", modulePrefix, orgID.String())
	return c.client.Delete(ctx, key)
}
