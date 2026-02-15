//! File repository — all queries tenant-scoped.

use crate::models::file::{File, FileStatus, VirusScanStatus};
use platform_common::error::AppError;
use platform_database::repository::Pagination;
use sqlx::PgPool;
use uuid::Uuid;

pub async fn create(pool: &PgPool, file: &File) -> Result<File, AppError> {
    sqlx::query_as::<_, File>(
        r#"INSERT INTO files
            (id, tenant_id, user_id, org_id, filename, content_type, size_bytes,
             s3_key, s3_bucket, checksum_sha256, status, virus_scan_status,
             thumbnail_s3_key, uploaded_at)
           VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
           RETURNING *"#,
    )
    .bind(file.id)
    .bind(file.tenant_id)
    .bind(file.user_id)
    .bind(file.org_id)
    .bind(&file.filename)
    .bind(&file.content_type)
    .bind(file.size_bytes)
    .bind(&file.s3_key)
    .bind(&file.s3_bucket)
    .bind(&file.checksum_sha256)
    .bind(file.status)
    .bind(file.virus_scan_status)
    .bind(&file.thumbnail_s3_key)
    .bind(file.uploaded_at)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to create file record").with_source(e))
}

pub async fn get_by_id(pool: &PgPool, id: Uuid, tenant_id: Uuid) -> Result<Option<File>, AppError> {
    sqlx::query_as::<_, File>("SELECT * FROM files WHERE id = $1 AND tenant_id = $2")
        .bind(id)
        .bind(tenant_id)
        .fetch_optional(pool)
        .await
        .map_err(|e| AppError::database_error("Failed to get file").with_source(e))
}

pub async fn list(
    pool: &PgPool,
    tenant_id: Uuid,
    pagination: &Pagination,
) -> Result<(Vec<File>, i64), AppError> {
    let total: (i64,) = sqlx::query_as(
        "SELECT COUNT(*) FROM files WHERE tenant_id = $1 AND status != 'deleted'",
    )
    .bind(tenant_id)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to count files").with_source(e))?;

    let files = sqlx::query_as::<_, File>(
        "SELECT * FROM files WHERE tenant_id = $1 AND status != 'deleted' ORDER BY created_at DESC LIMIT $2 OFFSET $3",
    )
    .bind(tenant_id)
    .bind(pagination.limit())
    .bind(pagination.offset())
    .fetch_all(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to list files").with_source(e))?;

    Ok((files, total.0))
}

pub async fn update_status(
    pool: &PgPool,
    id: Uuid,
    tenant_id: Uuid,
    status: FileStatus,
) -> Result<File, AppError> {
    sqlx::query_as::<_, File>(
        "UPDATE files SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3 RETURNING *",
    )
    .bind(status)
    .bind(id)
    .bind(tenant_id)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to update file status").with_source(e))
}

pub async fn update_scan_status(
    pool: &PgPool,
    id: Uuid,
    scan_status: VirusScanStatus,
) -> Result<File, AppError> {
    sqlx::query_as::<_, File>(
        "UPDATE files SET virus_scan_status = $1, updated_at = NOW() WHERE id = $2 RETURNING *",
    )
    .bind(scan_status)
    .bind(id)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to update scan status").with_source(e))
}

pub async fn soft_delete(pool: &PgPool, id: Uuid, tenant_id: Uuid) -> Result<File, AppError> {
    sqlx::query_as::<_, File>(
        "UPDATE files SET status = 'deleted', deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 RETURNING *",
    )
    .bind(id)
    .bind(tenant_id)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to delete file").with_source(e))
}
