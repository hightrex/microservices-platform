//! Plan repository — plans are global (not tenant-scoped).

use crate::models::plan::Plan;
use platform_common::error::AppError;
use sqlx::PgPool;
use uuid::Uuid;

/// Get a plan by ID.
pub async fn get_by_id(pool: &PgPool, id: Uuid) -> Result<Option<Plan>, AppError> {
    sqlx::query_as::<_, Plan>("SELECT * FROM plans WHERE id = $1")
        .bind(id)
        .fetch_optional(pool)
        .await
        .map_err(|e| AppError::database_error("Failed to get plan").with_source(e))
}

/// Get a plan by Stripe price ID.
pub async fn get_by_stripe_price_id(
    pool: &PgPool,
    stripe_price_id: &str,
) -> Result<Option<Plan>, AppError> {
    sqlx::query_as::<_, Plan>("SELECT * FROM plans WHERE stripe_price_id = $1")
        .bind(stripe_price_id)
        .fetch_optional(pool)
        .await
        .map_err(|e| AppError::database_error("Failed to get plan by Stripe ID").with_source(e))
}

/// List all active plans.
pub async fn list_active(pool: &PgPool) -> Result<Vec<Plan>, AppError> {
    sqlx::query_as::<_, Plan>(
        "SELECT * FROM plans WHERE is_active = true ORDER BY price_cents ASC",
    )
    .fetch_all(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to list plans").with_source(e))
}
