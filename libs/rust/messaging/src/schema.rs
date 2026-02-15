//! Event schema definitions for Redis Streams.
//!
//! Per foundation rules, every event MUST include: `id`, `type`, `version`,
//! `source`, `tenant_id`, and `correlation_id`.
//!
//! This schema is wire-compatible with the Go `messaging.Event` struct.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// Standardized event structure for inter-service communication.
///
/// Wire-compatible with the Go Event struct in `libs/go/pkg/messaging`.
/// All events published to Redis Streams MUST use this structure.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Event {
    /// Unique event identifier (UUID v4).
    pub id: String,
    /// Event type (e.g., "user.created", "subscription.updated").
    #[serde(rename = "type")]
    pub event_type: String,
    /// Schema version for forward compatibility (e.g., "1.0").
    pub version: String,
    /// Source service name (e.g., "billing-service").
    pub source: String,
    /// Tenant ID scoping this event.
    pub tenant_id: String,
    /// Event payload (JSON value).
    pub data: serde_json::Value,
    /// Event metadata.
    pub metadata: EventMetadata,
    /// Timestamp when the event was created.
    pub timestamp: DateTime<Utc>,
}

/// Event metadata.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EventMetadata {
    /// Correlation ID for distributed tracing.
    pub correlation_id: String,
}

impl Event {
    /// Create a new event with auto-generated ID and timestamp.
    pub fn new(
        event_type: impl Into<String>,
        source: impl Into<String>,
        tenant_id: Uuid,
        data: serde_json::Value,
    ) -> Self {
        Self {
            id: Uuid::new_v4().to_string(),
            event_type: event_type.into(),
            version: "1.0".to_string(),
            source: source.into(),
            tenant_id: tenant_id.to_string(),
            data,
            metadata: EventMetadata {
                correlation_id: Uuid::new_v4().to_string(),
            },
            timestamp: Utc::now(),
        }
    }

    /// Create an event with a specific correlation ID (for chaining events).
    pub fn with_correlation_id(mut self, correlation_id: impl Into<String>) -> Self {
        self.metadata.correlation_id = correlation_id.into();
        self
    }

    /// Serialize this event to a JSON string for Redis.
    ///
    /// # Errors
    ///
    /// Returns an error if serialization fails.
    pub fn to_json(&self) -> Result<String, serde_json::Error> {
        serde_json::to_string(self)
    }

    /// Deserialize an event from a JSON string.
    ///
    /// # Errors
    ///
    /// Returns an error if deserialization fails.
    pub fn from_json(json: &str) -> Result<Self, serde_json::Error> {
        serde_json::from_str(json)
    }
}

/// Well-known stream names matching the Go service conventions.
pub mod streams {
    pub const AUTH_EVENTS: &str = "auth-events";
    pub const ORG_EVENTS: &str = "org-events";
    pub const NOTIFICATION_EVENTS: &str = "notification-events";
    pub const BILLING_EVENTS: &str = "billing-events";
    pub const FILE_EVENTS: &str = "file-events";
    pub const AUDIT_EVENTS: &str = "audit-events";
}

/// Well-known event types.
pub mod event_types {
    // Auth events
    pub const USER_CREATED: &str = "user.created";
    pub const USER_UPDATED: &str = "user.updated";
    pub const AUTH_LOGIN: &str = "auth.login";
    pub const AUTH_LOGOUT: &str = "auth.logout";

    // Org events
    pub const ORG_CREATED: &str = "org.created";
    pub const ORG_UPDATED: &str = "org.updated";
    pub const ORG_DELETED: &str = "org.deleted";

    // Billing events
    pub const SUBSCRIPTION_CREATED: &str = "subscription.created";
    pub const SUBSCRIPTION_UPDATED: &str = "subscription.updated";
    pub const SUBSCRIPTION_CANCELED: &str = "subscription.canceled";
    pub const INVOICE_PAID: &str = "invoice.paid";
    pub const INVOICE_FAILED: &str = "invoice.failed";
    pub const USAGE_RECORDED: &str = "usage.recorded";
    pub const USAGE_QUOTA_EXCEEDED: &str = "usage.quota_exceeded";

    // File events
    pub const FILE_UPLOADED: &str = "file.uploaded";
    pub const FILE_DELETED: &str = "file.deleted";
    pub const FILE_SCAN_COMPLETED: &str = "file.scan_completed";
    pub const FILE_QUARANTINED: &str = "file.quarantined";
    pub const QUOTA_EXCEEDED: &str = "quota.exceeded";

    // Notification events
    pub const NOTIFICATION_SENT: &str = "notification.sent";
    pub const NOTIFICATION_FAILED: &str = "notification.failed";
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_event_creation() {
        let tenant_id = Uuid::new_v4();
        let event = Event::new(
            "user.created",
            "auth-service",
            tenant_id,
            json!({"user_id": "abc123"}),
        );

        assert_eq!(event.event_type, "user.created");
        assert_eq!(event.source, "auth-service");
        assert_eq!(event.tenant_id, tenant_id.to_string());
        assert_eq!(event.version, "1.0");
        assert!(!event.id.is_empty());
        assert!(!event.metadata.correlation_id.is_empty());
    }

    #[test]
    fn test_event_serialization_roundtrip() {
        let event = Event::new(
            "subscription.created",
            "billing-service",
            Uuid::new_v4(),
            json!({"plan_id": "pro", "amount": 9900}),
        );

        let json = event.to_json().unwrap();
        let deserialized = Event::from_json(&json).unwrap();

        assert_eq!(deserialized.id, event.id);
        assert_eq!(deserialized.event_type, event.event_type);
        assert_eq!(deserialized.source, event.source);
        assert_eq!(deserialized.tenant_id, event.tenant_id);
    }

    #[test]
    fn test_event_with_correlation_id() {
        let event = Event::new(
            "test.event",
            "test",
            Uuid::new_v4(),
            json!({}),
        )
        .with_correlation_id("custom-correlation-123");

        assert_eq!(event.metadata.correlation_id, "custom-correlation-123");
    }

    #[test]
    fn test_event_go_compatibility() {
        // Verify the JSON structure matches Go's Event format
        let event = Event::new(
            "user.created",
            "auth-service",
            Uuid::new_v4(),
            json!({"email": "test@example.com"}),
        );

        let json_value: serde_json::Value = serde_json::from_str(&event.to_json().unwrap()).unwrap();

        // Check all required fields exist
        assert!(json_value.get("id").is_some());
        assert!(json_value.get("type").is_some());
        assert!(json_value.get("version").is_some());
        assert!(json_value.get("source").is_some());
        assert!(json_value.get("tenant_id").is_some());
        assert!(json_value.get("data").is_some());
        assert!(json_value.get("metadata").is_some());
        assert!(json_value.get("timestamp").is_some());

        // Check metadata has correlation_id
        let metadata = json_value.get("metadata").unwrap();
        assert!(metadata.get("correlation_id").is_some());
    }
}
