//! File HTTP handlers.

use crate::models::file::FileResponse;
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
    mut multipart: Multipart,
) -> Result<impl IntoResponse, AppError> {
    let mut filename = None;
    let mut content_type = None;
    let mut size_bytes = 0i64;
    let mut found_file = false;

    // Process multipart fields
    while let Some(field) = multipart
        .next_field()
        .await
        .map_err(|e| AppError::bad_request(format!("Invalid multipart data: {e}")))?
    {
        let field_name = field.name().unwrap_or("").to_string();

        if field_name == "file" {
            found_file = true;
            filename = field.file_name().map(|s| s.to_string());
            content_type = field.content_type().map(|s| s.to_string());

            // Read the file data to get the size
            // In production, this would stream directly to S3
            let data = field
                .bytes()
                .await
                .map_err(|e| AppError::bad_request(format!("Failed to read file: {e}")))?;
            size_bytes = data.len() as i64;
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

    // Use a placeholder user_id; in production this comes from the auth middleware
    let user_id = Uuid::nil();

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
    Path(id): Path<Uuid>,
) -> Result<impl IntoResponse, AppError> {
    let file = file_service::get_file(&state.db_pool, &tenant_ctx, id).await?;

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
