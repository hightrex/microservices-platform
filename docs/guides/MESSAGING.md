# Messaging Guide

We use **Redis Streams** for asynchronous, event-driven communication.

## Event Schema
```json
{
  "id": "uuid",
  "type": "user.created",
  "version": "1.0",
  "source": "auth-service",
  "tenant_id": "uuid",
  "data": { ... },
  "metadata": {
    "correlation_id": "uuid",
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## Best Practices
- **Idempotency**: Consumers must handle duplicate events.
- **Consumer Groups**: Use groups for load balancing consumers.
- **DLQ**: Failed events go to a Dead Letter Queue for inspection.
