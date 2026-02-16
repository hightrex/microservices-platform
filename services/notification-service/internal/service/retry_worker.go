package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/hightrex/microservices-platform/services/notification-service/internal/models"
)

// RetryWorker periodically retries failed notification deliveries.
type RetryWorker struct {
	deliveryRepo DeliveryRepository
	notifRepo    NotificationRepository
	dlqRepo      DLQRepository
	notifSvc     *NotificationService
	maxRetries   int
	interval     time.Duration
}

// NewRetryWorker creates a new RetryWorker.
func NewRetryWorker(deliveryRepo DeliveryRepository, notifRepo NotificationRepository, dlqRepo DLQRepository, notifSvc *NotificationService, maxRetries int) *RetryWorker {
	return &RetryWorker{
		deliveryRepo: deliveryRepo,
		notifRepo:    notifRepo,
		dlqRepo:      dlqRepo,
		notifSvc:     notifSvc,
		maxRetries:   maxRetries,
		interval:     5 * time.Minute,
	}
}

// Start begins the retry worker loop. It blocks until the context is cancelled.
func (w *RetryWorker) Start(ctx context.Context) {
	logger.Info().Int("max_retries", w.maxRetries).Msg("Retry worker started")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Retry worker stopping")
			return
		case <-ticker.C:
			w.processRetries(ctx)
		}
	}
}

func (w *RetryWorker) processRetries(ctx context.Context) {
	failed, err := w.deliveryRepo.GetFailedDeliveries(ctx, w.maxRetries)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch failed deliveries for retry")
		return
	}

	if len(failed) == 0 {
		return
	}

	logger.Info().Int("count", len(failed)).Msg("Processing failed deliveries for retry")

	for _, delivery := range failed {
		if ctx.Err() != nil {
			return
		}

		// Set tenant context from the delivery record for proper tenant scoping
		tenantCtx := tenant.NewContext(ctx, delivery.TenantID)

		// Check if max retries exceeded — move to DLQ
		if delivery.RetryCount >= w.maxRetries {
			payload, _ := json.Marshal(map[string]interface{}{
				"delivery_id":     delivery.ID,
				"notification_id": delivery.NotificationID,
				"channel":         delivery.Channel,
				"recipient":       delivery.Recipient,
			})
			errMsg := "Max retries exceeded"
			if delivery.ErrorMessage != nil {
				errMsg = *delivery.ErrorMessage
			}
			if dlqErr := w.dlqRepo.Insert(tenantCtx, delivery.TenantID, "notification.delivery_failed", payload, errMsg); dlqErr != nil {
				logger.Error().Err(dlqErr).
					Str("delivery_id", delivery.ID.String()).
					Msg("Failed to write to DLQ after max retries")
			} else {
				logger.Warn().
					Str("delivery_id", delivery.ID.String()).
					Int("retry_count", delivery.RetryCount).
					Msg("Delivery moved to DLQ after max retries exceeded")
			}
			continue
		}

		sender, ok := w.notifSvc.channels[delivery.Channel]
		if !ok {
			logger.Warn().Str("channel", string(delivery.Channel)).Msg("No sender for channel, skipping retry")
			continue
		}

		// Look up the original notification to get subject and body
		origNotif, err := w.notifRepo.GetByID(tenantCtx, delivery.NotificationID)
		if err != nil {
			logger.Warn().Err(err).
				Str("delivery_id", delivery.ID.String()).
				Str("notification_id", delivery.NotificationID.String()).
				Msg("Failed to look up original notification for retry, skipping")
			continue
		}

		subject := ""
		if origNotif.Subject != nil {
			subject = *origNotif.Subject
		}

		// Increment retry count before attempting
		if incErr := w.deliveryRepo.IncrementRetryCount(tenantCtx, delivery.ID); incErr != nil {
			logger.Warn().Err(incErr).Str("delivery_id", delivery.ID.String()).Msg("Failed to increment retry count")
		}

		if err := sender.Send(tenantCtx, delivery.Recipient, subject, origNotif.Body, nil); err != nil {
			logger.Warn().Err(err).
				Str("delivery_id", delivery.ID.String()).
				Int("retry_count", delivery.RetryCount+1).
				Msg("Retry attempt failed")
		} else {
			// Mark delivery as delivered so it won't be retried again
			if statusErr := w.deliveryRepo.UpdateStatus(tenantCtx, delivery.ID, models.DeliveryStatusDelivered); statusErr != nil {
				logger.Error().Err(statusErr).
					Str("delivery_id", delivery.ID.String()).
					Msg("Failed to update delivery status to delivered after successful retry")
			}
			logger.Info().
				Str("delivery_id", delivery.ID.String()).
				Msg("Retry delivery succeeded")
		}
	}
}
