package health

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// Check is an interface for health checks
type Check interface {
	Name() string
	Check(ctx context.Context) error
}

// Manager manages health checks
type Manager struct {
	checks []Check
	mu     sync.RWMutex
}

// NewManager creates a new health check manager
func NewManager() *Manager {
	return &Manager{
		checks: make([]Check, 0),
	}
}

// AddCheck adds a health check
func (m *Manager) AddCheck(check Check) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks = append(m.checks, check)
}

// Handler returns a Gin handler for health checks
func (m *Manager) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		status := http.StatusOK
		results := make(map[string]string)

		m.mu.RLock()
		checks := m.checks
		m.mu.RUnlock()

		for _, check := range checks {
			if err := check.Check(c.Request.Context()); err != nil {
				status = http.StatusServiceUnavailable
				results[check.Name()] = err.Error()
			} else {
				results[check.Name()] = "UP"
			}
		}

		c.JSON(status, gin.H{
			"status": func() string {
				if status == http.StatusOK {
					return "UP"
				}
				return "DOWN"
			}(),
			"checks": results,
		})
	}
}
