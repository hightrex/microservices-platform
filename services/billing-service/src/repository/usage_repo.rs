//! Usage record repository — all queries tenant-scoped.

use crate::models::usage::UsageRecord;
use platform_common::error::AppError;
use sqlx::PgPool;
use uuid::Uuid;

/// Record a new usage entry (tenant-scoped).
pub async fn record(pool: &PgPool, usage: &UsageRecord) -> Result<UsageRecord, AppError> {
    sqlx::query_as::<_, UsageRecord>(
        r#"INSERT INTO usage_records
            (id, tenant_id, subscription_id, metric_name, quantity, recorded_at, aggregated)
           VALUES ($1, $2, $3, $4, $5, $6, $7)
           RETURNING *"#,
    )
    .bind(usage.id)
    .bind(usage.tenant_id)
    .bind(usage.subscription_id)
    .bind(&usage.metric_name)
    .bind(usage.quantity)
    .bind(usage.recorded_at)
    .bind(usage.aggregated)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to record usage").with_source(e))
}

/// Aggregate usage for a metric within a time range (tenant-scoped).
pub async fn aggregate(
    pool: &PgPool,
    tenant_id: Uuid,
    metric_name: &str,
    start: chrono::DateTime<chrono::Utc>,
    end: chrono::DateTime<chrono::Utc>,
) -> Result<i64, AppError> {
    let result: (Option<i64>,) = sqlx::query_as(
        "SELECT COALESCE(SUM(quantity)::BIGINT, 0::BIGINT) FROM usage_records WHERE tenant_id = $1 AND metric_name = $2 AND recorded_at >= $3 AND recorded_at <= $4",
    )
    .bind(tenant_id)
    .bind(metric_name)
    .bind(start)
    .bind(end)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to aggregate usage").with_source(e))?;

    Ok(result.0.unwrap_or(0))
}

/// Get usage records for the current subscription period.
pub async fn get_current_period(
    pool: &PgPool,
    subscription_id: Uuid,
    tenant_id: Uuid,
) -> Result<Vec<UsageRecord>, AppError> {
    sqlx::query_as::<_, UsageRecord>(
        r#"SELECT ur.* FROM usage_records ur
           JOIN subscriptions s ON ur.subscription_id = s.id
           WHERE ur.subscription_id = $1 AND ur.tenant_id = $2
             AND ur.recorded_at >= s.current_period_start
             AND ur.recorded_at <= s.current_period_end
           ORDER BY ur.recorded_at DESC"#,
    )
    .bind(subscription_id)
    .bind(tenant_id)
    .fetch_all(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to get current period usage").with_source(e))
}
