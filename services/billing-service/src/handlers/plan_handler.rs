//! Plan HTTP handlers.

use crate::models::plan::PlanResponse;
use crate::repository::plan_repo;
use crate::routes::AppState;
use axum::{extract::State, response::IntoResponse, Json};
use platform_common::error::AppError;

/// GET /api/v1/billing/plans — list available plans
pub async fn list_plans(
    State(state): State<AppState>,
) -> Result<impl IntoResponse, AppError> {
    let plans = plan_repo::list_active(&state.db_pool).await?;
    let responses: Vec<PlanResponse> = plans.into_iter().map(PlanResponse::from).collect();

    Ok(Json(serde_json::json!({
        "success": true,
        "data": responses
    })))
}
