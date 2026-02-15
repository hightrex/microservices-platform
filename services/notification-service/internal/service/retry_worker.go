package service

import (
	"context"
	"time"

	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
)

// RetryWorker periodically retries failed notification deliveries.
type RetryWorker struct {
	deliveryRepo DeliveryRepository
	notifSvc     *NotificationService
	maxRetries   int
	interval     time.Duration
}

// NewRetryWorker creates a new RetryWorker.
func NewRetryWorker(deliveryRepo DeliveryRepository, notifSvc *NotificationService, maxRetries int) *RetryWorker {
	return &RetryWorker{
		deliveryRepo: deliveryRepo,
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

		sender, ok := w.notifSvc.channels[delivery.Channel]
		if !ok {
			logger.Warn().Str("channel", string(delivery.Channel)).Msg("No sender for channel, skipping retry")
			continue
		}

		if err := sender.Send(ctx, delivery.Recipient, "", "", nil); err != nil {
			logger.Warn().Err(err).
				Str("delivery_id", delivery.ID.String()).
				Int("retry_count", delivery.RetryCount).
				Msg("Retry attempt failed")
		} else {
			logger.Info().
				Str("delivery_id", delivery.ID.String()).
				Msg("Retry delivery succeeded")
		}
	}
}
