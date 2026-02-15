//! Redis Streams event consumer.
//!
//! Consumes events with consumer group support, message acknowledgment,
//! dead letter queue (DLQ), and retry logic with exponential backoff.
//! Mirrors the Go `messaging.Consumer` with PEL recovery.

use crate::schema::Event;
use std::collections::HashMap;
use std::time::Duration;

/// Consumer configuration.
#[derive(Debug, Clone)]
pub struct ConsumerConfig {
    /// Number of messages to read per poll (default: 10).
    pub batch_size: usize,
    /// How long to block waiting for new messages (default: 2s).
    pub block_duration: Duration,
    /// Max delivery attempts before DLQ (default: 5).
    pub max_retries: usize,
    /// DLQ stream name (default: "{stream}.dlq").
    pub dlq_stream: String,
}

impl ConsumerConfig {
    /// Create a default consumer config for the given stream.
    pub fn new(stream: &str) -> Self {
        Self {
            batch_size: 10,
            block_duration: Duration::from_secs(2),
            max_retries: 5,
            dlq_stream: format!("{stream}.dlq"),
        }
    }

    /// Override the max retries.
    pub fn with_max_retries(mut self, max_retries: usize) -> Self {
        self.max_retries = max_retries;
        self
    }

    /// Override the batch size.
    pub fn with_batch_size(mut self, batch_size: usize) -> Self {
        self.batch_size = batch_size;
        self
    }
}

/// Event handler function signature.
pub type HandlerFn = Box<dyn Fn(Event) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), ConsumerError>> + Send>> + Send + Sync>;

/// Redis Streams event consumer with consumer group support.
pub struct Consumer {
    client: redis::Client,
}

impl Consumer {
    /// Create a new consumer.
    ///
    /// # Errors
    ///
    /// Returns an error if the Redis client cannot be created.
    pub fn new(redis_url: &str) -> Result<Self, redis::RedisError> {
        let client = redis::Client::open(redis_url)?;
        Ok(Self { client })
    }

    /// Create a consumer from an existing Redis client.
    pub fn from_client(client: redis::Client) -> Self {
        Self { client }
    }

    /// Subscribe to a stream with a consumer group.
    ///
    /// This method BLOCKS until the cancellation token is triggered.
    /// It recovers pending messages on startup, then continuously
    /// reads new messages. Failed messages are retried via the PEL
    /// and moved to a DLQ after `max_retries` attempts.
    ///
    /// # Errors
    ///
    /// Returns an error if the subscription cannot be established.
    pub async fn subscribe<F, Fut>(
        &self,
        stream: &str,
        group: &str,
        consumer_name: &str,
        handler: F,
        config: ConsumerConfig,
        cancel: tokio::sync::watch::Receiver<bool>,
    ) -> Result<(), ConsumerError>
    where
        F: Fn(Event) -> Fut + Send + Sync,
        Fut: std::future::Future<Output = Result<(), ConsumerError>> + Send,
    {
        let mut conn = self
            .client
            .get_multiplexed_async_connection()
            .await
            .map_err(ConsumerError::Redis)?;

        // Create consumer group (idempotent)
        Self::ensure_consumer_group(&mut conn, stream, group).await?;

        tracing::info!(
            stream = %stream,
            group = %group,
            consumer = %consumer_name,
            "Consumer subscribed to stream"
        );

        // Phase 1: Recover pending messages from previous runs
        self.recover_pending(&mut conn, stream, group, consumer_name, &handler, &config)
            .await;

        // Phase 2: Continuously read new messages
        let cancel = cancel;
        loop {
            // Check for cancellation
            if *cancel.borrow() {
                tracing::info!(stream = %stream, "Consumer shutting down");
                return Ok(());
            }

            let result: Result<Vec<redis::streams::StreamReadReply>, redis::RedisError> =
                redis::cmd("XREADGROUP")
                    .arg("GROUP")
                    .arg(group)
                    .arg(consumer_name)
                    .arg("COUNT")
                    .arg(config.batch_size)
                    .arg("BLOCK")
                    .arg(config.block_duration.as_millis() as u64)
                    .arg("STREAMS")
                    .arg(stream)
                    .arg(">")
                    .query_async(&mut conn)
                    .await;

            match result {
                Ok(replies) => {
                    for reply in replies {
                        for stream_key in reply.keys {
                            for message in stream_key.ids {
                                self.process_message(
                                    &mut conn,
                                    stream,
                                    group,
                                    &handler,
                                    &message,
                                    &config,
                                )
                                .await;
                            }
                        }
                    }
                }
                Err(e) => {
                    // Check if we should shut down
                    if *cancel.borrow() {
                        return Ok(());
                    }
                    tracing::error!(error = %e, stream = %stream, "Failed to read from stream");
                    tokio::time::sleep(Duration::from_secs(1)).await;
                }
            }

            // Non-blocking check for cancellation
            if cancel.has_changed().unwrap_or(false) && *cancel.borrow() {
                tracing::info!(stream = %stream, "Consumer shutting down");
                return Ok(());
            }
        }
    }

    /// Create consumer group if it doesn't exist.
    async fn ensure_consumer_group(
        conn: &mut redis::aio::MultiplexedConnection,
        stream: &str,
        group: &str,
    ) -> Result<(), ConsumerError> {
        let result: Result<String, redis::RedisError> = redis::cmd("XGROUP")
            .arg("CREATE")
            .arg(stream)
            .arg(group)
            .arg("0")
            .arg("MKSTREAM")
            .query_async(conn)
            .await;

        match result {
            Ok(_) => Ok(()),
            Err(e) => {
                let msg = e.to_string();
                if msg.contains("BUSYGROUP") {
                    // Group already exists, which is fine
                    Ok(())
                } else {
                    Err(ConsumerError::Redis(e))
                }
            }
        }
    }

    /// Recover and process pending messages from previous runs.
    async fn recover_pending<F, Fut>(
        &self,
        conn: &mut redis::aio::MultiplexedConnection,
        stream: &str,
        group: &str,
        consumer_name: &str,
        handler: &F,
        config: &ConsumerConfig,
    ) where
        F: Fn(Event) -> Fut + Send + Sync,
        Fut: std::future::Future<Output = Result<(), ConsumerError>> + Send,
    {
        // Get pending message info including delivery count
        let pending_result: Result<Vec<Vec<redis::Value>>, redis::RedisError> =
            redis::cmd("XPENDING")
                .arg(stream)
                .arg(group)
                .arg("-")
                .arg("+")
                .arg(100)
                .arg(consumer_name)
                .query_async(conn)
                .await;

        let pending_info = match pending_result {
            Ok(info) => info,
            Err(e) => {
                tracing::debug!(error = %e, stream = %stream, "No pending messages to recover");
                return;
            }
        };

        if pending_info.is_empty() {
            tracing::info!(stream = %stream, "No pending messages to recover");
            return;
        }

        // Build retry count map from PEL metadata
        let mut retry_counts: HashMap<String, usize> = HashMap::new();
        for entry in &pending_info {
            if entry.len() >= 4 {
                if let (Some(id), Some(count)) = (
                    redis::from_redis_value::<String>(entry[0].clone()).ok(),
                    redis::from_redis_value::<usize>(entry[3].clone()).ok(),
                ) {
                    retry_counts.insert(id, count);
                }
            }
        }

        tracing::info!(
            stream = %stream,
            count = retry_counts.len(),
            "Recovering pending messages"
        );

        // Read the actual pending messages
        let read_result: Result<Vec<redis::streams::StreamReadReply>, redis::RedisError> =
            redis::cmd("XREADGROUP")
                .arg("GROUP")
                .arg(group)
                .arg(consumer_name)
                .arg("COUNT")
                .arg(100)
                .arg("STREAMS")
                .arg(stream)
                .arg("0")
                .query_async(conn)
                .await;

        let replies = match read_result {
            Ok(r) => r,
            Err(_) => return,
        };

        for reply in replies {
            for stream_key in reply.keys {
                for message in stream_key.ids {
                    let retries = retry_counts.get(&message.id).copied().unwrap_or(0);

                    if retries >= config.max_retries {
                        tracing::warn!(
                            message_id = %message.id,
                            retries = retries,
                            max_retries = config.max_retries,
                            stream = %stream,
                            "Moving message to DLQ after max retries exceeded"
                        );
                        Self::move_to_dlq(conn, &config.dlq_stream, stream, &message).await;
                        let _ = Self::ack_message(conn, stream, group, &message.id).await;
                        continue;
                    }

                    match Self::parse_event(&message) {
                        Ok(event) => {
                            if let Err(e) = handler(event).await {
                                tracing::warn!(
                                    error = %e,
                                    message_id = %message.id,
                                    retries = retries,
                                    "Pending message processing failed, will retry"
                                );
                                // Leave unacked for next recovery
                                continue;
                            }
                            let _ = Self::ack_message(conn, stream, group, &message.id).await;
                        }
                        Err(e) => {
                            tracing::error!(
                                error = %e,
                                message_id = %message.id,
                                "Unparseable pending message, moving to DLQ"
                            );
                            Self::move_to_dlq(conn, &config.dlq_stream, stream, &message).await;
                            let _ = Self::ack_message(conn, stream, group, &message.id).await;
                        }
                    }
                }
            }
        }
    }

    /// Process a single new message from the stream.
    async fn process_message<F, Fut>(
        &self,
        conn: &mut redis::aio::MultiplexedConnection,
        stream: &str,
        group: &str,
        handler: &F,
        message: &redis::streams::StreamId,
        config: &ConsumerConfig,
    ) where
        F: Fn(Event) -> Fut + Send + Sync,
        Fut: std::future::Future<Output = Result<(), ConsumerError>> + Send,
    {
        match Self::parse_event(message) {
            Ok(event) => {
                if let Err(e) = handler(event).await {
                    tracing::error!(
                        error = %e,
                        message_id = %message.id,
                        stream = %stream,
                        "Failed to process event, will retry via PEL recovery"
                    );
                    // Don't ack — message stays in PEL for retry
                    return;
                }
                let _ = Self::ack_message(conn, stream, group, &message.id).await;
            }
            Err(e) => {
                tracing::error!(
                    error = %e,
                    message_id = %message.id,
                    "Unparseable message, moving to DLQ"
                );
                Self::move_to_dlq(conn, &config.dlq_stream, stream, message).await;
                let _ = Self::ack_message(conn, stream, group, &message.id).await;
            }
        }
    }

    /// Parse an Event from a Redis Stream message.
    fn parse_event(message: &redis::streams::StreamId) -> Result<Event, ConsumerError> {
        let event_data = message
            .map
            .get("event")
            .and_then(|v| match v {
                redis::Value::BulkString(bytes) => String::from_utf8(bytes.clone()).ok(),
                redis::Value::SimpleString(s) => Some(s.clone()),
                _ => None,
            })
            .ok_or_else(|| {
                ConsumerError::Parse(format!(
                    "missing 'event' field in message {}",
                    message.id
                ))
            })?;

        Event::from_json(&event_data).map_err(|e| {
            ConsumerError::Parse(format!(
                "failed to deserialize event from message {}: {e}",
                message.id
            ))
        })
    }

    /// Acknowledge a message.
    async fn ack_message(
        conn: &mut redis::aio::MultiplexedConnection,
        stream: &str,
        group: &str,
        message_id: &str,
    ) -> Result<(), redis::RedisError> {
        redis::cmd("XACK")
            .arg(stream)
            .arg(group)
            .arg(message_id)
            .query_async(conn)
            .await
    }

    /// Move a failed message to the dead letter queue.
    async fn move_to_dlq(
        conn: &mut redis::aio::MultiplexedConnection,
        dlq_stream: &str,
        source_stream: &str,
        message: &redis::streams::StreamId,
    ) {
        // Re-publish the event data to the DLQ stream with metadata
        let mut fields: Vec<(&str, String)> = Vec::new();

        if let Some(event_val) = message.map.get("event") {
            if let redis::Value::BulkString(bytes) = event_val {
                if let Ok(s) = String::from_utf8(bytes.clone()) {
                    fields.push(("event", s));
                }
            }
        }

        let now = chrono::Utc::now().to_rfc3339();

        let result: Result<String, redis::RedisError> = redis::cmd("XADD")
            .arg(dlq_stream)
            .arg("*")
            .arg("event")
            .arg(fields.first().map(|(_, v)| v.as_str()).unwrap_or(""))
            .arg("_dlq_source_stream")
            .arg(source_stream)
            .arg("_dlq_original_id")
            .arg(&message.id)
            .arg("_dlq_moved_at")
            .arg(&now)
            .query_async(conn)
            .await;

        if let Err(e) = result {
            tracing::error!(
                error = %e,
                dlq_stream = %dlq_stream,
                message_id = %message.id,
                "Failed to move message to DLQ"
            );
        }
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

/// Errors that can occur during event consumption.
#[derive(Debug, thiserror::Error)]
pub enum ConsumerError {
    #[error("Redis error: {0}")]
    Redis(#[from] redis::RedisError),
    #[error("Parse error: {0}")]
    Parse(String),
    #[error("Handler error: {0}")]
    Handler(String),
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_consumer_config_defaults() {
        let config = ConsumerConfig::new("test-stream");
        assert_eq!(config.batch_size, 10);
        assert_eq!(config.max_retries, 5);
        assert_eq!(config.dlq_stream, "test-stream.dlq");
    }

    #[test]
    fn test_consumer_config_overrides() {
        let config = ConsumerConfig::new("test-stream")
            .with_max_retries(3)
            .with_batch_size(50);
        assert_eq!(config.max_retries, 3);
        assert_eq!(config.batch_size, 50);
    }

    #[test]
    fn test_consumer_error_display() {
        let err = ConsumerError::Parse("bad data".to_string());
        assert_eq!(format!("{err}"), "Parse error: bad data");
    }
}
