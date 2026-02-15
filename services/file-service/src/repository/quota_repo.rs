//! Storage quota repository — tenant-scoped.

use crate::models::quota::StorageQuota;
use platform_common::error::AppError;
use sqlx::PgPool;
use uuid::Uuid;

/// Get or create the quota record for a tenant (upsert).
pub async fn get_or_create(
    pool: &PgPool,
    tenant_id: Uuid,
    default_quota_bytes: i64,
) -> Result<StorageQuota, AppError> {
    sqlx::query_as::<_, StorageQuota>(
        r#"INSERT INTO storage_quotas (tenant_id, quota_bytes, used_bytes)
           VALUES ($1, $2, 0)
           ON CONFLICT (tenant_id)
           DO UPDATE SET updated_at = NOW()
           RETURNING *"#,
    )
    .bind(tenant_id)
    .bind(default_quota_bytes)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to get/create quota").with_source(e))
}

/// Atomically increment used bytes (after upload).
pub async fn increment_usage(
    pool: &PgPool,
    tenant_id: Uuid,
    bytes: i64,
) -> Result<StorageQuota, AppError> {
    sqlx::query_as::<_, StorageQuota>(
        "UPDATE storage_quotas SET used_bytes = used_bytes + $1, updated_at = NOW() WHERE tenant_id = $2 RETURNING *",
    )
    .bind(bytes)
    .bind(tenant_id)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to increment quota usage").with_source(e))
}

/// Atomically decrement used bytes (after delete).
pub async fn decrement_usage(
    pool: &PgPool,
    tenant_id: Uuid,
    bytes: i64,
) -> Result<StorageQuota, AppError> {
    sqlx::query_as::<_, StorageQuota>(
        "UPDATE storage_quotas SET used_bytes = GREATEST(used_bytes - $1, 0), updated_at = NOW() WHERE tenant_id = $2 RETURNING *",
    )
    .bind(bytes)
    .bind(tenant_id)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to decrement quota usage").with_source(e))
}

/// Check if uploading additional bytes would exceed the quota.
pub async fn check_quota(
    pool: &PgPool,
    tenant_id: Uuid,
    additional_bytes: i64,
    default_quota_bytes: i64,
) -> Result<bool, AppError> {
    let quota = get_or_create(pool, tenant_id, default_quota_bytes).await?;
    Ok(quota.used_bytes + additional_bytes <= quota.quota_bytes)
}
