//! Subscription repository — all queries are tenant-scoped.

use crate::models::subscription::{Subscription, SubscriptionStatus};
use platform_common::error::AppError;
use sqlx::PgPool;
use uuid::Uuid;

/// Create a new subscription (tenant-scoped).
pub async fn create(pool: &PgPool, sub: &Subscription) -> Result<Subscription, AppError> {
    sqlx::query_as::<_, Subscription>(
        r#"INSERT INTO subscriptions
            (id, tenant_id, org_id, plan_id, stripe_subscription_id, status,
             current_period_start, current_period_end, cancel_at, canceled_at, trial_end)
           VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
           RETURNING *"#,
    )
    .bind(sub.id)
    .bind(sub.tenant_id)
    .bind(sub.org_id)
    .bind(sub.plan_id)
    .bind(&sub.stripe_subscription_id)
    .bind(sub.status)
    .bind(sub.current_period_start)
    .bind(sub.current_period_end)
    .bind(sub.cancel_at)
    .bind(sub.canceled_at)
    .bind(sub.trial_end)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to create subscription").with_source(e))
}

/// Get subscription by ID (tenant-scoped).
pub async fn get_by_id(
    pool: &PgPool,
    id: Uuid,
    tenant_id: Uuid,
) -> Result<Option<Subscription>, AppError> {
    sqlx::query_as::<_, Subscription>(
        "SELECT * FROM subscriptions WHERE id = $1 AND tenant_id = $2",
    )
    .bind(id)
    .bind(tenant_id)
    .fetch_optional(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to get subscription").with_source(e))
}

/// Get the current active subscription for a tenant.
pub async fn get_by_tenant(
    pool: &PgPool,
    tenant_id: Uuid,
) -> Result<Option<Subscription>, AppError> {
    sqlx::query_as::<_, Subscription>(
        "SELECT * FROM subscriptions WHERE tenant_id = $1 AND status IN ('active', 'trialing', 'past_due') ORDER BY created_at DESC LIMIT 1",
    )
    .bind(tenant_id)
    .fetch_optional(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to get tenant subscription").with_source(e))
}

/// Update subscription status.
pub async fn update_status(
    pool: &PgPool,
    id: Uuid,
    tenant_id: Uuid,
    status: SubscriptionStatus,
) -> Result<Subscription, AppError> {
    sqlx::query_as::<_, Subscription>(
        "UPDATE subscriptions SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3 RETURNING *",
    )
    .bind(status)
    .bind(id)
    .bind(tenant_id)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to update subscription status").with_source(e))
}

/// Cancel a subscription (set canceled_at and optionally cancel_at).
pub async fn cancel(
    pool: &PgPool,
    id: Uuid,
    tenant_id: Uuid,
    immediate: bool,
) -> Result<Subscription, AppError> {
    let now = chrono::Utc::now();
    if immediate {
        sqlx::query_as::<_, Subscription>(
            "UPDATE subscriptions SET status = 'canceled', canceled_at = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3 RETURNING *",
        )
        .bind(now)
        .bind(id)
        .bind(tenant_id)
        .fetch_one(pool)
        .await
    } else {
        sqlx::query_as::<_, Subscription>(
            "UPDATE subscriptions SET cancel_at = current_period_end, canceled_at = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3 RETURNING *",
        )
        .bind(now)
        .bind(id)
        .bind(tenant_id)
        .fetch_one(pool)
        .await
    }
    .map_err(|e| AppError::database_error("Failed to cancel subscription").with_source(e))
}

/// Update the plan for a subscription (upgrade/downgrade).
pub async fn update_plan(
    pool: &PgPool,
    id: Uuid,
    tenant_id: Uuid,
    new_plan_id: Uuid,
) -> Result<Subscription, AppError> {
    sqlx::query_as::<_, Subscription>(
        "UPDATE subscriptions SET plan_id = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3 RETURNING *",
    )
    .bind(new_plan_id)
    .bind(id)
    .bind(tenant_id)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to update subscription plan").with_source(e))
}
