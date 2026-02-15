//! File access log repository — tenant-scoped.

use crate::models::access_log::FileAccessLog;
use platform_common::error::AppError;
use platform_database::repository::Pagination;
use sqlx::PgPool;
use uuid::Uuid;

pub async fn create(pool: &PgPool, log: &FileAccessLog) -> Result<FileAccessLog, AppError> {
    sqlx::query_as::<_, FileAccessLog>(
        r#"INSERT INTO file_access_log
            (id, file_id, tenant_id, user_id, action, ip_address, user_agent)
           VALUES ($1,$2,$3,$4,$5,$6,$7)
           RETURNING *"#,
    )
    .bind(log.id)
    .bind(log.file_id)
    .bind(log.tenant_id)
    .bind(log.user_id)
    .bind(log.action)
    .bind(&log.ip_address)
    .bind(&log.user_agent)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to create access log").with_source(e))
}

pub async fn list_by_file(
    pool: &PgPool,
    file_id: Uuid,
    tenant_id: Uuid,
    pagination: &Pagination,
) -> Result<(Vec<FileAccessLog>, i64), AppError> {
    let total: (i64,) = sqlx::query_as(
        "SELECT COUNT(*) FROM file_access_log WHERE file_id = $1 AND tenant_id = $2",
    )
    .bind(file_id)
    .bind(tenant_id)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to count access logs").with_source(e))?;

    let logs = sqlx::query_as::<_, FileAccessLog>(
        "SELECT * FROM file_access_log WHERE file_id = $1 AND tenant_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4",
    )
    .bind(file_id)
    .bind(tenant_id)
    .bind(pagination.limit())
    .bind(pagination.offset())
    .fetch_all(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to list access logs").with_source(e))?;

    Ok((logs, total.0))
}
