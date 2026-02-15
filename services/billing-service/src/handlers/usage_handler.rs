//! Usage HTTP handlers.

use crate::models::usage::RecordUsageRequest;
use crate::routes::AppState;
use crate::service::usage_service;
use axum::{extract::{Query, State}, http::StatusCode, response::IntoResponse, Json};
use platform_common::error::AppError;
use platform_common::tenant::TenantContext;
use platform_common::validation::validate_request;
use serde::Deserialize;

/// Usage query parameters.
#[derive(Debug, Deserialize)]
pub struct UsageQuery {
    pub metric_name: String,
}

/// GET /api/v1/billing/usage — get current usage
pub async fn get_usage(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    Query(query): Query<UsageQuery>,
) -> Result<impl IntoResponse, AppError> {
    let usage =
        usage_service::get_usage(&state.db_pool, &tenant_ctx, &query.metric_name).await?;

    Ok(Json(serde_json::json!({
        "success": true,
        "data": usage
    })))
}

/// GET /api/v1/billing/usage/quota — quota status
pub async fn get_quota_status(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    Query(query): Query<UsageQuery>,
) -> Result<impl IntoResponse, AppError> {
    // Default limit; in production, this would come from the plan's features
    let limit = 10000;
    let quota = usage_service::get_quota_status(
        &state.db_pool,
        &tenant_ctx,
        &query.metric_name,
        limit,
    )
    .await?;

    Ok(Json(serde_json::json!({
        "success": true,
        "data": quota
    })))
}

/// POST /api/v1/billing/usage/record — record usage (internal API)
pub async fn record_usage(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    Json(req): Json<RecordUsageRequest>,
) -> Result<impl IntoResponse, AppError> {
    validate_request(&req)?;

    let usage = usage_service::record_usage(
        &state.db_pool,
        &state.producer,
        &tenant_ctx,
        &req.metric_name,
        req.quantity,
    )
    .await?;

    Ok((
        StatusCode::CREATED,
        Json(serde_json::json!({
            "success": true,
            "data": {
                "id": usage.id,
                "metric_name": usage.metric_name,
                "quantity": usage.quantity,
                "recorded_at": usage.recorded_at,
            },
            "message": "Usage recorded"
        })),
    ))
}
