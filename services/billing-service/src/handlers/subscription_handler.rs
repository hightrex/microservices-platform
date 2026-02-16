//! Subscription HTTP handlers.

use crate::routes::AppState;
use crate::service::subscription_service;
use axum::{extract::State, http::StatusCode, response::IntoResponse, Json};
use platform_common::error::AppError;
use platform_common::tenant::TenantContext;
use platform_common::validation::validate_request;
use platform_middleware::auth::UserContext;
use uuid::Uuid;

use crate::models::subscription::{
    CancelSubscriptionRequest, CreateSubscriptionRequest, SubscriptionResponse,
    UpdateSubscriptionRequest,
};

/// GET /api/v1/billing/subscription — get current subscription
pub async fn get_subscription(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
) -> Result<impl IntoResponse, AppError> {
    let subscription =
        subscription_service::get_subscription(&state.db_pool, &tenant_ctx).await?;

    match subscription {
        Some(sub) => Ok(Json(serde_json::json!({
            "success": true,
            "data": SubscriptionResponse::from(sub)
        }))),
        None => Ok(Json(serde_json::json!({
            "success": true,
            "data": null,
            "message": "No active subscription"
        }))),
    }
}

/// POST /api/v1/billing/subscription — create subscription
pub async fn create_subscription(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    user_ctx: UserContext,
    Json(req): Json<CreateSubscriptionRequest>,
) -> Result<impl IntoResponse, AppError> {
    validate_request(&req)?;

    let plan_id = uuid::Uuid::parse_str(&req.plan_id)
        .map_err(|_| AppError::bad_request("Invalid plan_id: must be a valid UUID"))?;
    let user_id = Uuid::parse_str(&user_ctx.user_id)
        .map_err(|_| AppError::bad_request("Invalid user ID in authentication token"))?;

    let subscription = subscription_service::create_subscription(
        &state.db_pool,
        &state.producer,
        &tenant_ctx,
        plan_id,
        user_id,
    )
    .await?;

    Ok((
        StatusCode::CREATED,
        Json(serde_json::json!({
            "success": true,
            "data": SubscriptionResponse::from(subscription),
            "message": "Subscription created"
        })),
    ))
}

/// PUT /api/v1/billing/subscription — upgrade/downgrade
pub async fn update_subscription(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    user_ctx: UserContext,
    Json(req): Json<UpdateSubscriptionRequest>,
) -> Result<impl IntoResponse, AppError> {
    validate_request(&req)?;

    let new_plan_id = uuid::Uuid::parse_str(&req.new_plan_id)
        .map_err(|_| AppError::bad_request("Invalid new_plan_id: must be a valid UUID"))?;
    let user_id = Uuid::parse_str(&user_ctx.user_id)
        .map_err(|_| AppError::bad_request("Invalid user ID in authentication token"))?;

    let subscription = subscription_service::change_plan(
        &state.db_pool,
        &state.producer,
        &tenant_ctx,
        new_plan_id,
        req.immediate,
        user_id,
    )
    .await?;

    Ok(Json(serde_json::json!({
        "success": true,
        "data": SubscriptionResponse::from(subscription),
        "message": "Subscription updated"
    })))
}

/// DELETE /api/v1/billing/subscription — cancel subscription
pub async fn cancel_subscription(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    user_ctx: UserContext,
    Json(req): Json<CancelSubscriptionRequest>,
) -> Result<impl IntoResponse, AppError> {
    let user_id = Uuid::parse_str(&user_ctx.user_id)
        .map_err(|_| AppError::bad_request("Invalid user ID in authentication token"))?;

    let subscription = subscription_service::cancel_subscription(
        &state.db_pool,
        &state.producer,
        &tenant_ctx,
        req.immediate,
        user_id,
    )
    .await?;

    Ok(Json(serde_json::json!({
        "success": true,
        "data": SubscriptionResponse::from(subscription),
        "message": if req.immediate { "Subscription canceled immediately" } else { "Subscription will cancel at end of period" }
    })))
}

/// POST /api/v1/billing/subscription/pause — pause subscription
pub async fn pause_subscription(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    user_ctx: UserContext,
) -> Result<impl IntoResponse, AppError> {
    let user_id = Uuid::parse_str(&user_ctx.user_id)
        .map_err(|_| AppError::bad_request("Invalid user ID in authentication token"))?;

    let subscription =
        subscription_service::pause_subscription(&state.db_pool, &state.producer, &tenant_ctx, user_id).await?;

    Ok(Json(serde_json::json!({
        "success": true,
        "data": SubscriptionResponse::from(subscription),
        "message": "Subscription paused"
    })))
}

/// POST /api/v1/billing/subscription/resume — resume subscription
pub async fn resume_subscription(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    user_ctx: UserContext,
) -> Result<impl IntoResponse, AppError> {
    let user_id = Uuid::parse_str(&user_ctx.user_id)
        .map_err(|_| AppError::bad_request("Invalid user ID in authentication token"))?;

    let subscription =
        subscription_service::resume_subscription(&state.db_pool, &state.producer, &tenant_ctx, user_id).await?;

    Ok(Json(serde_json::json!({
        "success": true,
        "data": SubscriptionResponse::from(subscription),
        "message": "Subscription resumed"
    })))
}
