//! Subscription service — business logic for subscription lifecycle.

use crate::models::subscription::{Subscription, SubscriptionStatus};
use crate::repository::{plan_repo, subscription_repo};
use platform_common::error::AppError;
use platform_common::tenant::TenantContext;
use platform_messaging::producer::Producer;
use platform_messaging::schema::{event_types, streams};
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

/// Create a new subscription for a tenant.
pub async fn create_subscription(
    pool: &PgPool,
    producer: &Arc<Producer>,
    tenant_ctx: &TenantContext,
    plan_id: Uuid,
) -> Result<Subscription, AppError> {
    // Validate plan exists
    let plan = plan_repo::get_by_id(pool, plan_id)
        .await?
        .ok_or_else(|| AppError::not_found("Plan not found"))?;

    if !plan.is_active {
        return Err(AppError::bad_request("Plan is not active"));
    }

    // Check tenant doesn't already have an active subscription
    if let Some(existing) = subscription_repo::get_by_tenant(pool, tenant_ctx.tenant_id).await? {
        if matches!(
            existing.status,
            SubscriptionStatus::Active | SubscriptionStatus::Trialing
        ) {
            return Err(AppError::conflict(
                "Tenant already has an active subscription. Cancel or upgrade the existing one.",
            ));
        }
    }

    let now = chrono::Utc::now();
    let period_end = match plan.billing_interval {
        crate::models::plan::BillingInterval::Month => now + chrono::Duration::days(30),
        crate::models::plan::BillingInterval::Year => now + chrono::Duration::days(365),
    };

    let subscription = Subscription {
        id: Uuid::new_v4(),
        tenant_id: tenant_ctx.tenant_id,
        org_id: tenant_ctx.org_id.unwrap_or(tenant_ctx.tenant_id),
        plan_id,
        stripe_subscription_id: None,
        status: SubscriptionStatus::Active,
        current_period_start: now,
        current_period_end: period_end,
        cancel_at: None,
        canceled_at: None,
        trial_end: None,
        created_at: now,
        updated_at: now,
    };

    let created = subscription_repo::create(pool, &subscription).await?;

    // Publish event
    let _ = producer
        .publish(
            streams::BILLING_EVENTS,
            event_types::SUBSCRIPTION_CREATED,
            tenant_ctx.tenant_id,
            serde_json::json!({
                "subscription_id": created.id,
                "plan_id": plan_id,
                "plan_name": plan.name,
                "status": "active",
            }),
        )
        .await;

    tracing::info!(
        subscription_id = %created.id,
        tenant_id = %tenant_ctx.tenant_id,
        plan_id = %plan_id,
        "Subscription created"
    );

    Ok(created)
}

/// Get the current subscription for a tenant.
pub async fn get_subscription(
    pool: &PgPool,
    tenant_ctx: &TenantContext,
) -> Result<Option<Subscription>, AppError> {
    subscription_repo::get_by_tenant(pool, tenant_ctx.tenant_id).await
}

/// Upgrade or downgrade a subscription to a new plan.
pub async fn change_plan(
    pool: &PgPool,
    producer: &Arc<Producer>,
    tenant_ctx: &TenantContext,
    new_plan_id: Uuid,
    _immediate: bool,
) -> Result<Subscription, AppError> {
    let subscription = subscription_repo::get_by_tenant(pool, tenant_ctx.tenant_id)
        .await?
        .ok_or_else(|| AppError::not_found("No active subscription found"))?;

    // Validate new plan exists
    let new_plan = plan_repo::get_by_id(pool, new_plan_id)
        .await?
        .ok_or_else(|| AppError::not_found("New plan not found"))?;

    if !new_plan.is_active {
        return Err(AppError::bad_request("New plan is not active"));
    }

    if subscription.plan_id == new_plan_id {
        return Err(AppError::bad_request("Already on this plan"));
    }

    let updated =
        subscription_repo::update_plan(pool, subscription.id, tenant_ctx.tenant_id, new_plan_id)
            .await?;

    // Publish event
    let _ = producer
        .publish(
            streams::BILLING_EVENTS,
            event_types::SUBSCRIPTION_UPDATED,
            tenant_ctx.tenant_id,
            serde_json::json!({
                "subscription_id": updated.id,
                "old_plan_id": subscription.plan_id,
                "new_plan_id": new_plan_id,
                "plan_name": new_plan.name,
            }),
        )
        .await;

    Ok(updated)
}

/// Cancel a subscription.
pub async fn cancel_subscription(
    pool: &PgPool,
    producer: &Arc<Producer>,
    tenant_ctx: &TenantContext,
    immediate: bool,
) -> Result<Subscription, AppError> {
    let subscription = subscription_repo::get_by_tenant(pool, tenant_ctx.tenant_id)
        .await?
        .ok_or_else(|| AppError::not_found("No active subscription found"))?;

    let canceled =
        subscription_repo::cancel(pool, subscription.id, tenant_ctx.tenant_id, immediate).await?;

    // Publish event
    let _ = producer
        .publish(
            streams::BILLING_EVENTS,
            event_types::SUBSCRIPTION_CANCELED,
            tenant_ctx.tenant_id,
            serde_json::json!({
                "subscription_id": canceled.id,
                "immediate": immediate,
                "cancel_at": canceled.cancel_at,
            }),
        )
        .await;

    tracing::info!(
        subscription_id = %canceled.id,
        tenant_id = %tenant_ctx.tenant_id,
        immediate = immediate,
        "Subscription canceled"
    );

    Ok(canceled)
}

/// Pause a subscription.
pub async fn pause_subscription(
    pool: &PgPool,
    tenant_ctx: &TenantContext,
) -> Result<Subscription, AppError> {
    let subscription = subscription_repo::get_by_tenant(pool, tenant_ctx.tenant_id)
        .await?
        .ok_or_else(|| AppError::not_found("No active subscription found"))?;

    subscription_repo::update_status(
        pool,
        subscription.id,
        tenant_ctx.tenant_id,
        SubscriptionStatus::Paused,
    )
    .await
}

/// Resume a paused subscription.
pub async fn resume_subscription(
    pool: &PgPool,
    tenant_ctx: &TenantContext,
) -> Result<Subscription, AppError> {
    let subscription = subscription_repo::get_by_tenant(pool, tenant_ctx.tenant_id)
        .await?
        .ok_or_else(|| AppError::not_found("No subscription found"))?;

    if subscription.status != SubscriptionStatus::Paused {
        return Err(AppError::bad_request("Subscription is not paused"));
    }

    subscription_repo::update_status(
        pool,
        subscription.id,
        tenant_ctx.tenant_id,
        SubscriptionStatus::Active,
    )
    .await
}
