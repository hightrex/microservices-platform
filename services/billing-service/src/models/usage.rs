//! Usage metering domain models.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use validator::Validate;

/// Database model for usage records.
#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct UsageRecord {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub subscription_id: Uuid,
    pub metric_name: String,
    pub quantity: i64,
    pub recorded_at: DateTime<Utc>,
    pub aggregated: bool,
    pub created_at: DateTime<Utc>,
}

/// Request to record usage.
#[derive(Debug, Clone, Deserialize, Validate)]
pub struct RecordUsageRequest {
    #[validate(length(min = 1, max = 100, message = "metric_name is required"))]
    pub metric_name: String,
    #[validate(range(min = 1, message = "quantity must be positive"))]
    pub quantity: i64,
}

/// Usage aggregation response.
#[derive(Debug, Clone, Serialize)]
pub struct UsageResponse {
    pub metric_name: String,
    pub total_quantity: i64,
    pub period_start: DateTime<Utc>,
    pub period_end: DateTime<Utc>,
}

/// Quota status response.
#[derive(Debug, Clone, Serialize)]
pub struct QuotaStatusResponse {
    pub metric_name: String,
    pub current_usage: i64,
    pub limit: i64,
    pub remaining: i64,
    pub exceeded: bool,
}
