//! File HTTP handlers.

use crate::models::access_log::{AccessAction, FileAccessLog};
use crate::models::file::FileResponse;
use crate::repository::access_log_repo;
use crate::routes::AppState;
use crate::service::{file_service, file_service::UploadMetadata};
use axum::{
    extract::{Multipart, Path, Query, State},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use platform_common::error::AppError;
use platform_common::tenant::TenantContext;
use platform_database::repository::Pagination;
use platform_middleware::auth::UserContext;
use serde::Deserialize;
use uuid::Uuid;

#[derive(Debug, Deserialize)]
pub struct PaginationParams {
    #[serde(default = "default_page")]
    pub page: u32,
    #[serde(default = "default_per_page")]
    pub per_page: u32,
}

fn default_page() -> u32 {
    1
}

fn default_per_page() -> u32 {
    20
}

/// POST /api/v1/files/upload — multipart file upload
pub async fn upload_file(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    user_ctx: UserContext,
    mut multipart: Multipart,
) -> Result<impl IntoResponse, AppError> {
    // Extract user_id from the authenticated JWT context
    let user_id = Uuid::parse_str(&user_ctx.user_id)
        .map_err(|_| AppError::bad_request("Invalid user ID in authentication token"))?;

    let max_file_size = state.quota_config.max_file_size_bytes;
    let mut filename = None;
    let mut content_type = None;
    let mut size_bytes = 0i64;
    let mut found_file = false;

    // Process multipart fields
    while let Some(mut field) = multipart
        .next_field()
        .await
        .map_err(|e| AppError::bad_request(format!("Invalid multipart data: {e}")))?
    {
        let field_name = field.name().unwrap_or("").to_string();

        if field_name == "file" {
            found_file = true;
            filename = field.file_name().map(|s| s.to_string());
            content_type = field.content_type().map(|s| s.to_string());

            // Stream the file data in chunks to avoid loading entire file into memory.
            // Enforce size limit incrementally rather than after full read.
            let mut total_bytes = 0i64;
            while let Some(chunk) = field
                .chunk()
                .await
                .map_err(|e| AppError::bad_request(format!("Failed to read file chunk: {e}")))?
            {
                total_bytes += chunk.len() as i64;
                if total_bytes > max_file_size {
                    return Err(AppError::bad_request(format!(
                        "File too large: exceeds limit of {} bytes",
                        max_file_size
                    )));
                }
            }
            size_bytes = total_bytes;
        }
    }

    if !found_file {
        return Err(AppError::bad_request("Missing 'file' field in multipart upload"));
    }

    let filename =
        filename.ok_or_else(|| AppError::bad_request("File field must have a filename"))?;

    let content_type = content_type
        .or_else(|| mime_guess::from_path(&filename).first().map(|m| m.to_string()))
        .unwrap_or_else(|| "application/octet-stream".to_string());

    let metadata = UploadMetadata {
        filename: filename.clone(),
        content_type,
        size_bytes,
        user_id,
        checksum_sha256: None,
    };

    let file = file_service::create_file_record(
        &state.db_pool,
        &state.producer,
        &tenant_ctx,
        metadata,
        &state.s3_config,
        &state.quota_config,
        &state.upload_config,
    )
    .await?;

    Ok((
        StatusCode::CREATED,
        Json(serde_json::json!({
            "success": true,
            "data": FileResponse::from(file),
            "message": "File uploaded"
        })),
    ))
}

/// GET /api/v1/files/:id — get file metadata
pub async fn get_file(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    user_ctx: UserContext,
    Path(id): Path<Uuid>,
) -> Result<impl IntoResponse, AppError> {
    let file = file_service::get_file(&state.db_pool, &tenant_ctx, id).await?;

    // Record file access for audit trail
    let user_id = Uuid::parse_str(&user_ctx.user_id)
        .map_err(|_| AppError::bad_request("Invalid user ID in authentication token"))?;
    let access_log = FileAccessLog {
        id: Uuid::new_v4(),
        file_id: id,
        tenant_id: tenant_ctx.tenant_id,
        user_id,
        action: AccessAction::View,
        ip_address: None,
        user_agent: None,
        created_at: chrono::Utc::now(),
    };
    if let Err(e) = access_log_repo::create(&state.db_pool, &access_log).await {
        tracing::warn!(error = %e, file_id = %id, "Failed to record file access log");
    }

    Ok(Json(serde_json::json!({
        "success": true,
        "data": FileResponse::from(file)
    })))
}

/// DELETE /api/v1/files/:id — soft delete
pub async fn delete_file(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    Path(id): Path<Uuid>,
) -> Result<impl IntoResponse, AppError> {
    let file =
        file_service::delete_file(&state.db_pool, &state.producer, &tenant_ctx, id).await?;

    Ok(Json(serde_json::json!({
        "success": true,
        "data": FileResponse::from(file),
        "message": "File deleted"
    })))
}

/// GET /api/v1/files — list files (tenant-scoped, paginated)
pub async fn list_files(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    Query(params): Query<PaginationParams>,
) -> Result<impl IntoResponse, AppError> {
    let pagination = Pagination::new(params.page, params.per_page);
    let response = file_service::list_files(&state.db_pool, &tenant_ctx, &pagination).await?;

    Ok(Json(serde_json::json!({
        "success": true,
        "data": response.data,
        "pagination": {
            "page": response.page,
            "per_page": response.per_page,
            "total": response.total,
            "total_pages": response.total_pages,
        }
    })))
}
