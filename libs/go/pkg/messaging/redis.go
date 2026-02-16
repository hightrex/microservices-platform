package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hightrex/microservices-platform/libs/go/pkg/logger"
	"github.com/hightrex/microservices-platform/libs/go/pkg/tenant"
	"github.com/redis/go-redis/v9"
)

// Event represents a standardized event structure.
// Per foundation rules: must include id, type, version, source, tenant_id, and correlation_id.
type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Version   string          `json:"version"`
	Source    string          `json:"source"`
	TenantID  string          `json:"tenant_id"`
	Data      json.RawMessage `json:"data"`
	Metadata  Metadata        `json:"metadata"`
	Timestamp time.Time       `json:"timestamp"`
}

// Metadata holds event metadata
type Metadata struct {
	CorrelationID string `json:"correlation_id"`
}

// Producer publishes events to Redis Stream
type Producer struct {
	client *redis.Client
	source string
}

// NewProducer creates a new event producer
func NewProducer(client *redis.Client, source string) *Producer {
	return &Producer{client: client, source: source}
}

// Publish sends an event to the specified stream
func (p *Producer) Publish(ctx context.Context, stream string, eventType string, data interface{}) error {
	tenantID := tenant.FromContext(ctx)

	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Propagate correlation ID from request context for distributed tracing;
	// fall back to a new UUID if none is set.
	correlationID := tenant.CorrelationIDFromContext(ctx)
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	event := Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Version:   "1.0",
		Source:    p.source,
		TenantID:  tenantID.String(),
		Data:      bytes,
		Timestamp: time.Now(),
		Metadata: Metadata{
			CorrelationID: correlationID,
		},
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"event": eventBytes,
		},
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Info().Str("stream", stream).Str("type", eventType).Msg("Event published")
	return nil
}

// Consumer consumes events from a Redis Stream
type Consumer struct {
	client *redis.Client
}

// NewConsumer creates a new event consumer
func NewConsumer(client *redis.Client) *Consumer {
	return &Consumer{client: client}
}

// HandlerFunc is the function signature for processing events
type HandlerFunc func(ctx context.Context, event Event) error

// SubscribeConfig holds configuration for event subscription
type SubscribeConfig struct {
	BatchSize     int64         // Number of messages to read per poll (default: 10)
	BlockDuration time.Duration // How long to block waiting for new messages (default: 2s)
	MaxRetries    int           // Max delivery attempts before DLQ (default: 5)
	DLQStream     string        // DLQ stream name (default: "{stream}.dlq")
}

// DefaultSubscribeConfig returns sensible defaults for subscription configuration
func DefaultSubscribeConfig(stream string) SubscribeConfig {
	return SubscribeConfig{
		BatchSize:     10,
		BlockDuration: 2 * time.Second,
		MaxRetries:    5,
		DLQStream:     stream + ".dlq",
	}
}

// Subscribe starts listening for events on a stream. This method BLOCKS until
// the context is cancelled. It recovers pending messages on startup, then
// continuously reads new messages. Failed messages are retried via the PEL
// (Pending Entries List) and moved to a DLQ after MaxRetries attempts.
func (c *Consumer) Subscribe(ctx context.Context, stream, group, consumer string, handler HandlerFunc) error {
	return c.SubscribeWithConfig(ctx, stream, group, consumer, handler, DefaultSubscribeConfig(stream))
}

// SubscribeWithConfig starts listening with custom configuration (blocking).
func (c *Consumer) SubscribeWithConfig(ctx context.Context, stream, group, consumer string, handler HandlerFunc, cfg SubscribeConfig) error {
	// Create consumer group if it doesn't exist
	err := c.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	logger.Info().
		Str("stream", stream).
		Str("group", group).
		Str("consumer", consumer).
		Msg("Consumer subscribed to stream")

	// Phase 1: Recover pending messages from previous runs (crash recovery)
	c.recoverPending(ctx, stream, group, consumer, handler, cfg)

	// Phase 2: Continuously read new messages (blocking loop)
	for {
		select {
		case <-ctx.Done():
			logger.Info().Str("stream", stream).Msg("Consumer shutting down")
			return ctx.Err()
		default:
		}

		entries, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{stream, ">"},
			Count:    cfg.BatchSize,
			Block:    cfg.BlockDuration,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logger.Error().Err(err).Str("stream", stream).Msg("Failed to read from stream")
			time.Sleep(time.Second) // Backoff on transient errors
			continue
		}

		for _, streamEntry := range entries {
			for _, message := range streamEntry.Messages {
				c.processMessage(ctx, stream, group, handler, message, cfg)
			}
		}
	}
}

// recoverPending processes messages left unacknowledged from previous runs.
// This handles crash recovery: if the consumer died mid-processing, those
// messages are still in the PEL and need to be re-processed or DLQ'd.
func (c *Consumer) recoverPending(ctx context.Context, stream, group, consumer string, handler HandlerFunc, cfg SubscribeConfig) {
	// Get pending message info including delivery count
	pendingInfo, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream:   stream,
		Group:    group,
		Start:    "-",
		End:      "+",
		Count:    100,
		Consumer: consumer,
	}).Result()

	if err != nil || len(pendingInfo) == 0 {
		logger.Info().Str("stream", stream).Msg("No pending messages to recover")
		return
	}

	logger.Info().Str("stream", stream).Int("count", len(pendingInfo)).Msg("Recovering pending messages")

	// Build retry count map from PEL metadata
	retryCounts := make(map[string]int64)
	for _, p := range pendingInfo {
		retryCounts[p.ID] = p.RetryCount
	}

	// Read the actual pending message data
	entries, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, "0"},
		Count:    100,
	}).Result()

	if err != nil || len(entries) == 0 || len(entries[0].Messages) == 0 {
		return
	}

	for _, msg := range entries[0].Messages {
		select {
		case <-ctx.Done():
			return
		default:
		}

		retries := retryCounts[msg.ID]

		// DLQ: move poison messages after max retries
		if int(retries) >= cfg.MaxRetries {
			logger.Warn().
				Str("message_id", msg.ID).
				Int64("retries", retries).
				Int("max_retries", cfg.MaxRetries).
				Str("stream", stream).
				Msg("Moving message to DLQ after max retries exceeded")
			c.moveToDLQ(ctx, cfg.DLQStream, stream, msg)
			c.client.XAck(ctx, stream, group, msg.ID)
			continue
		}

		event, err := parseMessage(msg)
		if err != nil {
			logger.Error().Err(err).Str("message_id", msg.ID).Msg("Unparseable pending message, moving to DLQ")
			c.moveToDLQ(ctx, cfg.DLQStream, stream, msg)
			c.client.XAck(ctx, stream, group, msg.ID)
			continue
		}

		if err := handler(ctx, event); err != nil {
			logger.Warn().Err(err).
				Str("event_id", event.ID).
				Str("event_type", event.Type).
				Int64("delivery_count", retries).
				Msg("Pending message processing failed, will retry on next restart")
			continue // Leave unacked for next recovery
		}

		if ackErr := c.client.XAck(ctx, stream, group, msg.ID).Err(); ackErr != nil {
			logger.Error().Err(ackErr).Str("message_id", msg.ID).Msg("Failed to acknowledge recovered message")
		}
	}
}

// processMessage handles a single new message from the stream.
func (c *Consumer) processMessage(ctx context.Context, stream, group string, handler HandlerFunc, message redis.XMessage, cfg SubscribeConfig) {
	event, err := parseMessage(message)
	if err != nil {
		logger.Error().Err(err).Str("message_id", message.ID).Msg("Unparseable message, acking to prevent poison retry loop")
		c.client.XAck(ctx, stream, group, message.ID)
		return
	}

	if err := handler(ctx, event); err != nil {
		logger.Error().Err(err).
			Str("event_id", event.ID).
			Str("event_type", event.Type).
			Str("stream", stream).
			Msg("Failed to process event, will retry via PEL recovery")
		// Don't ack — message stays in PEL for retry on next recovery cycle
		return
	}

	if ackErr := c.client.XAck(ctx, stream, group, message.ID).Err(); ackErr != nil {
		logger.Error().Err(ackErr).Str("message_id", message.ID).Msg("Failed to acknowledge message")
	}
}

// parseMessage extracts an Event from a Redis Stream message.
func parseMessage(msg redis.XMessage) (Event, error) {
	eventData, ok := msg.Values["event"].(string)
	if !ok {
		return Event{}, fmt.Errorf("invalid event format: missing 'event' field in message %s", msg.ID)
	}

	var event Event
	if err := json.Unmarshal([]byte(eventData), &event); err != nil {
		return Event{}, fmt.Errorf("failed to unmarshal event from message %s: %w", msg.ID, err)
	}

	return event, nil
}

// moveToDLQ moves a failed message to the dead letter queue stream.
func (c *Consumer) moveToDLQ(ctx context.Context, dlqStream, sourceStream string, message redis.XMessage) {
	values := make(map[string]interface{})
	for k, v := range message.Values {
		values[k] = v
	}
	values["_dlq_source_stream"] = sourceStream
	values["_dlq_original_id"] = message.ID
	values["_dlq_moved_at"] = time.Now().Format(time.RFC3339)

	if err := c.client.XAdd(ctx, &redis.XAddArgs{
		Stream: dlqStream,
		Values: values,
	}).Err(); err != nil {
		logger.Error().Err(err).
			Str("dlq_stream", dlqStream).
			Str("message_id", message.ID).
			Msg("Failed to move message to DLQ")
	}
}
