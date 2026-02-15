//! Quota management service.

use crate::models::quota::QuotaResponse;
use crate::repository::quota_repo;
use platform_common::error::AppError;
use platform_common::tenant::TenantContext;
use sqlx::PgPool;

/// Get the current quota status for a tenant.
pub async fn get_quota_status(
    pool: &PgPool,
    tenant_ctx: &TenantContext,
    default_limit_bytes: i64,
) -> Result<QuotaResponse, AppError> {
    let quota =
        quota_repo::get_or_create(pool, tenant_ctx.tenant_id, default_limit_bytes).await?;
    Ok(QuotaResponse::from(quota))
}
