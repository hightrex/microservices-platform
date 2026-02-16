//! Usage service — metered billing and quota enforcement.

use crate::models::usage::{QuotaStatusResponse, UsageRecord, UsageResponse};
use crate::repository::{subscription_repo, usage_repo};
use platform_common::error::AppError;
use platform_common::tenant::TenantContext;
use platform_messaging::producer::Producer;
use platform_messaging::schema::{event_types, streams};
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

/// Record a usage data point for a tenant.
pub async fn record_usage(
    pool: &PgPool,
    producer: &Arc<Producer>,
    tenant_ctx: &TenantContext,
    metric_name: &str,
    quantity: i64,
    user_id: Uuid,
) -> Result<UsageRecord, AppError> {
    // Get the tenant's active subscription
    let subscription = subscription_repo::get_by_tenant(pool, tenant_ctx.tenant_id)
        .await?
        .ok_or_else(|| AppError::bad_request("No active subscription found"))?;

    let now = chrono::Utc::now();
    let usage = UsageRecord {
        id: Uuid::new_v4(),
        tenant_id: tenant_ctx.tenant_id,
        subscription_id: subscription.id,
        metric_name: metric_name.to_string(),
        quantity,
        recorded_at: now,
        aggregated: false,
        created_at: now,
    };

    let recorded = usage_repo::record(pool, &usage).await?;

    // Publish event
    if let Err(e) = producer
        .publish(
            streams::BILLING_EVENTS,
            event_types::USAGE_RECORDED,
            tenant_ctx.tenant_id,
            serde_json::json!({
                "user_id": user_id,
                "metric_name": metric_name,
                "quantity": quantity,
                "subscription_id": subscription.id,
            }),
        )
        .await
    {
        tracing::error!(error = %e, metric_name = %metric_name, "Failed to publish usage.recorded event");
    }

    Ok(recorded)
}

/// Get aggregated usage for a metric in the current billing period.
pub async fn get_usage(
    pool: &PgPool,
    tenant_ctx: &TenantContext,
    metric_name: &str,
) -> Result<UsageResponse, AppError> {
    let subscription = subscription_repo::get_by_tenant(pool, tenant_ctx.tenant_id)
        .await?
        .ok_or_else(|| AppError::bad_request("No active subscription found"))?;

    let total = usage_repo::aggregate(
        pool,
        tenant_ctx.tenant_id,
        metric_name,
        subscription.current_period_start,
        subscription.current_period_end,
    )
    .await?;

    Ok(UsageResponse {
        metric_name: metric_name.to_string(),
        total_quantity: total,
        period_start: subscription.current_period_start,
        period_end: subscription.current_period_end,
    })
}

/// Check quota status for a tenant's metric.
pub async fn get_quota_status(
    pool: &PgPool,
    tenant_ctx: &TenantContext,
    metric_name: &str,
    limit: i64,
) -> Result<QuotaStatusResponse, AppError> {
    let usage = get_usage(pool, tenant_ctx, metric_name).await?;
    let remaining = (limit - usage.total_quantity).max(0);

    Ok(QuotaStatusResponse {
        metric_name: metric_name.to_string(),
        current_usage: usage.total_quantity,
        limit,
        remaining,
        exceeded: usage.total_quantity >= limit,
    })
}
