//! Storage quota domain models.

use chrono::{DateTime, Utc};
use serde::Serialize;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, sqlx::FromRow)]
pub struct StorageQuota {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub quota_bytes: i64,
    pub used_bytes: i64,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize)]
pub struct QuotaResponse {
    pub quota_bytes: i64,
    pub used_bytes: i64,
    pub remaining_bytes: i64,
    pub usage_percentage: f64,
}

impl From<StorageQuota> for QuotaResponse {
    fn from(q: StorageQuota) -> Self {
        let remaining = (q.quota_bytes - q.used_bytes).max(0);
        let pct = if q.quota_bytes > 0 {
            (q.used_bytes as f64 / q.quota_bytes as f64) * 100.0
        } else {
            0.0
        };
        Self {
            quota_bytes: q.quota_bytes,
            used_bytes: q.used_bytes,
            remaining_bytes: remaining,
            usage_percentage: pct,
        }
    }
}
