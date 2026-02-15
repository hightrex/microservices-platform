//! Core file operations — upload, download, delete, list.

use crate::config::{QuotaConfig, S3Config, UploadConfig};
use crate::models::file::{File, FileResponse, FileStatus, VirusScanStatus};
use crate::repository::{file_repo, quota_repo};
use crate::service::security;
use platform_common::error::AppError;
use platform_common::tenant::TenantContext;
use platform_database::repository::{PaginatedResponse, Pagination};
use platform_messaging::producer::Producer;
use platform_messaging::schema::{event_types, streams};
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

/// Upload metadata (from multipart form).
pub struct UploadMetadata {
    pub filename: String,
    pub content_type: String,
    pub size_bytes: i64,
    pub user_id: Uuid,
    pub checksum_sha256: Option<String>,
}

/// Process a file upload: validate, check quota, store metadata.
///
/// The actual S3 upload is handled separately (either server-side or via presigned URL).
pub async fn create_file_record(
    pool: &PgPool,
    producer: &Arc<Producer>,
    tenant_ctx: &TenantContext,
    metadata: UploadMetadata,
    s3_config: &S3Config,
    quota_config: &QuotaConfig,
    upload_config: &UploadConfig,
) -> Result<File, AppError> {
    // Validate filename (path traversal prevention)
    let safe_filename = security::validate_filename(&metadata.filename)?;

    // Validate content type
    security::validate_content_type(&metadata.content_type, &upload_config.allowed_content_types)?;

    // Check file size limit
    if metadata.size_bytes > quota_config.max_file_size_bytes {
        return Err(AppError::bad_request(format!(
            "File too large: {} bytes exceeds limit of {} bytes",
            metadata.size_bytes, quota_config.max_file_size_bytes
        )));
    }

    // Check quota
    let has_quota = quota_repo::check_quota(
        pool,
        tenant_ctx.tenant_id,
        metadata.size_bytes,
        quota_config.default_limit_bytes,
    )
    .await?;

    if !has_quota {
        // Publish quota exceeded event
        let _ = producer
            .publish(
                streams::FILE_EVENTS,
                event_types::QUOTA_EXCEEDED,
                tenant_ctx.tenant_id,
                serde_json::json!({
                    "attempted_bytes": metadata.size_bytes,
                }),
            )
            .await;
        return Err(AppError::bad_request("Storage quota exceeded"));
    }

    let file_id = Uuid::new_v4();
    let bucket = format!("{}-files", s3_config.bucket_prefix);
    let s3_key = security::generate_s3_key(&tenant_ctx.tenant_id, &file_id, &safe_filename);
    let now = chrono::Utc::now();

    let file = File {
        id: file_id,
        tenant_id: tenant_ctx.tenant_id,
        user_id: metadata.user_id,
        org_id: tenant_ctx.org_id.unwrap_or(tenant_ctx.tenant_id),
        filename: safe_filename,
        content_type: metadata.content_type,
        size_bytes: metadata.size_bytes,
        s3_key,
        s3_bucket: bucket,
        checksum_sha256: metadata.checksum_sha256,
        status: FileStatus::Available,
        virus_scan_status: VirusScanStatus::Pending,
        thumbnail_s3_key: None,
        uploaded_at: Some(now),
        deleted_at: None,
        created_at: now,
        updated_at: now,
    };

    let created = file_repo::create(pool, &file).await?;

    // Increment quota usage
    let _ = quota_repo::increment_usage(pool, tenant_ctx.tenant_id, metadata.size_bytes).await;

    // Publish event
    let _ = producer
        .publish(
            streams::FILE_EVENTS,
            event_types::FILE_UPLOADED,
            tenant_ctx.tenant_id,
            serde_json::json!({
                "file_id": created.id,
                "filename": created.filename,
                "size_bytes": created.size_bytes,
                "content_type": created.content_type,
            }),
        )
        .await;

    tracing::info!(
        file_id = %created.id,
        tenant_id = %tenant_ctx.tenant_id,
        filename = %created.filename,
        size_bytes = created.size_bytes,
        "File uploaded"
    );

    Ok(created)
}

/// Get a file by ID (tenant-scoped).
pub async fn get_file(
    pool: &PgPool,
    tenant_ctx: &TenantContext,
    file_id: Uuid,
) -> Result<File, AppError> {
    file_repo::get_by_id(pool, file_id, tenant_ctx.tenant_id)
        .await?
        .ok_or_else(|| AppError::not_found("File not found"))
}

/// List files for a tenant with pagination.
pub async fn list_files(
    pool: &PgPool,
    tenant_ctx: &TenantContext,
    pagination: &Pagination,
) -> Result<PaginatedResponse<FileResponse>, AppError> {
    let (files, total) = file_repo::list(pool, tenant_ctx.tenant_id, pagination).await?;
    let responses: Vec<FileResponse> = files.into_iter().map(FileResponse::from).collect();
    Ok(PaginatedResponse::new(responses, pagination, total))
}

/// Soft-delete a file and decrement quota.
pub async fn delete_file(
    pool: &PgPool,
    producer: &Arc<Producer>,
    tenant_ctx: &TenantContext,
    file_id: Uuid,
) -> Result<File, AppError> {
    let file = get_file(pool, tenant_ctx, file_id).await?;

    if file.status == FileStatus::Deleted {
        return Err(AppError::bad_request("File is already deleted"));
    }

    let deleted = file_repo::soft_delete(pool, file_id, tenant_ctx.tenant_id).await?;

    // Decrement quota
    let _ = quota_repo::decrement_usage(pool, tenant_ctx.tenant_id, file.size_bytes).await;

    // Publish event
    let _ = producer
        .publish(
            streams::FILE_EVENTS,
            event_types::FILE_DELETED,
            tenant_ctx.tenant_id,
            serde_json::json!({
                "file_id": file_id,
                "size_bytes": file.size_bytes,
            }),
        )
        .await;

    Ok(deleted)
}
