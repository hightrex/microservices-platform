package consumer

import (
	"context"
	"sync"

	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/messaging"
	"github.com/redis/go-redis/v9"
)

// handlerRegistration maps an event type to its handler function
type handlerRegistration struct {
	EventType string
	Handler   messaging.HandlerFunc
}

// Consumer manages event subscriptions and processing with event type routing.
// It subscribes to Redis Streams and routes events to the correct handler
// based on the event's Type field.
type Consumer struct {
	client  *redis.Client
	config  Config
	streams map[string][]handlerRegistration // stream name -> event type handlers
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// Config holds consumer configuration
type Config struct {
	GroupName    string
	ConsumerName string
	Concurrency  int
}

// NewConsumer creates a new consumer manager
func NewConsumer(client *redis.Client, cfg Config) *Consumer {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	return &Consumer{
		client:  client,
		config:  cfg,
		streams: make(map[string][]handlerRegistration),
	}
}

// RegisterHandler registers a handler for a specific stream and event type.
// Multiple handlers can be registered for different event types on the same stream.
// The consumer will route each event to the handler matching its Type field.
func (c *Consumer) RegisterHandler(stream, eventType string, handler messaging.HandlerFunc) {
	c.streams[stream] = append(c.streams[stream], handlerRegistration{
		EventType: eventType,
		Handler:   handler,
	})
	logger.Info().
		Str("stream", stream).
		Str("event_type", eventType).
		Msg("Registered event handler")
}

// Start begins consuming events in background goroutines.
// Each stream gets its own goroutine with a routing handler that dispatches
// events to the correct handler based on event type.
func (c *Consumer) Start(ctx context.Context) error {
	logger.Info().
		Int("streams", len(c.streams)).
		Str("group", c.config.GroupName).
		Str("consumer", c.config.ConsumerName).
		Msg("Starting event consumers")

	ctx, c.cancel = context.WithCancel(ctx)

	for stream, handlers := range c.streams {
		c.wg.Add(1)
		go c.consumeLoop(ctx, stream, handlers)
	}

	return nil
}

func (c *Consumer) consumeLoop(ctx context.Context, stream string, handlers []handlerRegistration) {
	defer c.wg.Done()

	libConsumer := messaging.NewConsumer(c.client)

	// Build a routing handler that dispatches events by type
	router := func(ctx context.Context, event messaging.Event) error {
		for _, h := range handlers {
			if h.EventType == event.Type {
				return h.Handler(ctx, event)
			}
		}
		// No handler registered for this event type — acknowledge and skip
		logger.Debug().
			Str("event_type", event.Type).
			Str("stream", stream).
			Msg("No handler registered for event type, skipping")
		return nil
	}

	// Subscribe blocks until context is cancelled
	err := libConsumer.Subscribe(ctx, stream, c.config.GroupName, c.config.ConsumerName, router)
	if err != nil && ctx.Err() == nil {
		logger.Error().Err(err).Str("stream", stream).Msg("Consumer subscription failed")
	}
}

// Stop gracefully shuts down all consumers by cancelling the context
// and waiting for all goroutines to finish.
func (c *Consumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	logger.Info().Msg("Event consumers stopped")
}

// Check implements the health.Check interface for health monitoring
func (c *Consumer) Check(ctx context.Context) error {
	for stream := range c.streams {
		_, err := c.client.XInfoGroups(ctx, stream).Result()
		if err != nil {
			return err
		}
	}
	return nil
}

// Name returns the name of the health check
func (c *Consumer) Name() string {
	return "event_consumers"
}
