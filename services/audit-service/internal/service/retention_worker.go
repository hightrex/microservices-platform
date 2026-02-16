package service

import (
	"context"
	"time"

	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
)

// RetentionWorker periodically enforces retention policies by purging expired audit logs.
type RetentionWorker struct {
	auditRepo     AuditRepository
	retentionRepo RetentionRepository
	interval      time.Duration
}

// NewRetentionWorker creates a new RetentionWorker.
func NewRetentionWorker(auditRepo AuditRepository, retentionRepo RetentionRepository) *RetentionWorker {
	return &RetentionWorker{
		auditRepo:     auditRepo,
		retentionRepo: retentionRepo,
		interval:      24 * time.Hour,
	}
}

// Start begins the retention enforcement loop. It blocks until the context is cancelled.
func (w *RetentionWorker) Start(ctx context.Context) {
	logger.Info().Msg("Retention enforcement worker started")

	// Run once immediately on startup
	w.enforce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Retention enforcement worker stopping")
			return
		case <-ticker.C:
			w.enforce(ctx)
		}
	}
}

func (w *RetentionWorker) enforce(ctx context.Context) {
	// Use ListAllActive which does not require a tenant context — the worker
	// runs as a background process and must enforce policies across all tenants.
	policies, err := w.retentionRepo.ListAllActive(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch retention policies for enforcement")
		return
	}

	if len(policies) == 0 {
		logger.Debug().Msg("No retention policies to enforce")
		return
	}

	now := time.Now()
	totalPurged := int64(0)

	for _, policy := range policies {
		if ctx.Err() != nil {
			return
		}

		cutoff := now.AddDate(0, 0, -policy.RetentionDays)
		purged, err := w.auditRepo.PurgeExpiredByEventType(ctx, policy.TenantID, policy.EventType, cutoff)
		if err != nil {
			logger.Error().Err(err).
				Str("tenant_id", policy.TenantID.String()).
				Str("event_type", policy.EventType).
				Int("retention_days", policy.RetentionDays).
				Msg("Failed to purge expired audit logs")
			continue
		}

		if purged > 0 {
			logger.Info().
				Str("tenant_id", policy.TenantID.String()).
				Str("event_type", policy.EventType).
				Int64("purged_count", purged).
				Time("cutoff", cutoff).
				Msg("Purged expired audit logs per retention policy")
			totalPurged += purged
		}
	}

	if totalPurged > 0 {
		logger.Info().Int64("total_purged", totalPurged).Msg("Retention enforcement cycle completed")
	}
}
