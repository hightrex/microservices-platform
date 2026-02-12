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

// Event represents a standardized event structure
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
}

// NewProducer creates a new event producer
func NewProducer(client *redis.Client) *Producer {
	return &Producer{client: client}
}

// Publish sends an event to the specified stream
func (p *Producer) Publish(ctx context.Context, stream string, eventType string, data interface{}) error {
	tenantID := tenant.FromContext(ctx)

	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	event := Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Version:   "1.0",
		Source:    "service", // Should be configurable
		TenantID:  tenantID.String(),
		Data:      bytes,
		Timestamp: time.Now(),
		Metadata: Metadata{
			CorrelationID: uuid.New().String(), // Should pull from context
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

// Subscribe starts listening for events on a stream
func (c *Consumer) Subscribe(ctx context.Context, stream, group, consumer string, handler HandlerFunc) error {
	// Create consumer group if it doesn't exist
	err := c.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				entries, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    group,
					Consumer: consumer,
					Streams:  []string{stream, ">"},
					Count:    10,
					Block:    2 * time.Second,
				}).Result()

				if err != nil {
					if err != redis.Nil {
						logger.Error().Err(err).Msg("Failed to read from stream")
					}
					continue
				}

				for _, streamEntry := range entries {
					for _, message := range streamEntry.Messages {
						var event Event
						eventData, ok := message.Values["event"].(string)
						if !ok {
							logger.Error().Msg("Invalid event format")
							continue
						}

						if err := json.Unmarshal([]byte(eventData), &event); err != nil {
							logger.Error().Err(err).Msg("Failed to unmarshal event")
							continue
						}

						// Process event
						if err := handler(ctx, event); err != nil {
							logger.Error().Err(err).Msg("Failed to process event")
							// Maybe implement retry or DLQ logic here
						} else {
							// Acknowledge message
							c.client.XAck(ctx, stream, group, message.ID)
						}
					}
				}
			}
		}
	}()

	return nil
}
