//! Quota HTTP handlers.

use crate::routes::AppState;
use crate::service::quota_service;
use axum::{extract::State, response::IntoResponse, Json};
use platform_common::error::AppError;
use platform_common::tenant::TenantContext;

/// GET /api/v1/files/quota — current quota status
pub async fn get_quota(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
) -> Result<impl IntoResponse, AppError> {
    let quota = quota_service::get_quota_status(
        &state.db_pool,
        &tenant_ctx,
        state.quota_config.default_limit_bytes,
    )
    .await?;

    Ok(Json(serde_json::json!({
        "success": true,
        "data": quota
    })))
}
