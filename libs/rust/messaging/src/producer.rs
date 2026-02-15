//! Redis Streams event producer.
//!
//! Publishes events to Redis Streams with automatic timestamp
//! and correlation ID injection. Mirrors the Go `messaging.Producer`.

use crate::schema::Event;
use uuid::Uuid;

/// Event producer that publishes to Redis Streams.
#[derive(Clone)]
pub struct Producer {
    client: redis::Client,
    source: String,
}

impl Producer {
    /// Create a new producer for the given source service.
    ///
    /// # Errors
    ///
    /// Returns an error if the Redis client cannot be created.
    pub fn new(redis_url: &str, source: impl Into<String>) -> Result<Self, redis::RedisError> {
        let client = redis::Client::open(redis_url)?;
        Ok(Self {
            client,
            source: source.into(),
        })
    }

    /// Create a producer from an existing Redis client.
    pub fn from_client(client: redis::Client, source: impl Into<String>) -> Self {
        Self {
            client,
            source: source.into(),
        }
    }

    /// Publish an event to a Redis Stream.
    ///
    /// The event is serialized to JSON and published as a single field
    /// `event` in the stream entry, matching the Go producer's format.
    ///
    /// # Errors
    ///
    /// Returns an error if publishing fails.
    pub async fn publish(
        &self,
        stream: &str,
        event_type: &str,
        tenant_id: Uuid,
        data: serde_json::Value,
    ) -> Result<String, ProducerError> {
        let event = Event::new(event_type, &self.source, tenant_id, data);
        self.publish_event(stream, &event).await
    }

    /// Publish an event with a specific correlation ID.
    ///
    /// Use this when chaining events from an existing request context.
    ///
    /// # Errors
    ///
    /// Returns an error if publishing fails.
    pub async fn publish_with_correlation(
        &self,
        stream: &str,
        event_type: &str,
        tenant_id: Uuid,
        data: serde_json::Value,
        correlation_id: &str,
    ) -> Result<String, ProducerError> {
        let event = Event::new(event_type, &self.source, tenant_id, data)
            .with_correlation_id(correlation_id);
        self.publish_event(stream, &event).await
    }

    /// Publish a pre-built event to a stream.
    ///
    /// # Errors
    ///
    /// Returns an error if publishing fails.
    pub async fn publish_event(
        &self,
        stream: &str,
        event: &Event,
    ) -> Result<String, ProducerError> {
        let event_json = event
            .to_json()
            .map_err(|e| ProducerError::Serialization(e.to_string()))?;

        let mut conn = self
            .client
            .get_multiplexed_async_connection()
            .await
            .map_err(ProducerError::Redis)?;

        let message_id: String = redis::cmd("XADD")
            .arg(stream)
            .arg("*")
            .arg("event")
            .arg(&event_json)
            .query_async(&mut conn)
            .await
            .map_err(ProducerError::Redis)?;

        tracing::info!(
            stream = %stream,
            event_type = %event.event_type,
            event_id = %event.id,
            tenant_id = %event.tenant_id,
            message_id = %message_id,
            "Event published"
        );

        Ok(message_id)
    }

    /// Check if the Redis connection is healthy.
    ///
    /// # Errors
    ///
    /// Returns an error string if the health check fails.
    pub async fn health_check(&self) -> Result<(), String> {
        let mut conn = self
            .client
            .get_multiplexed_async_connection()
            .await
            .map_err(|e| format!("redis connection failed: {e}"))?;

        redis::cmd("PING")
            .query_async::<String>(&mut conn)
            .await
            .map(|_| ())
            .map_err(|e| format!("redis ping failed: {e}"))
    }
}

/// Errors that can occur during event publishing.
#[derive(Debug, thiserror::Error)]
pub enum ProducerError {
    #[error("Redis error: {0}")]
    Redis(#[from] redis::RedisError),
    #[error("Serialization error: {0}")]
    Serialization(String),
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_producer_error_display() {
        let err = ProducerError::Serialization("bad json".to_string());
        assert_eq!(format!("{err}"), "Serialization error: bad json");
    }
}
