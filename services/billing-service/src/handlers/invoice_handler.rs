//! Invoice HTTP handlers.

use crate::routes::AppState;
use crate::service::invoice_service;
use axum::{extract::{Path, Query, State}, response::IntoResponse, Json};
use platform_common::error::AppError;
use platform_common::tenant::TenantContext;
use platform_database::repository::Pagination;
use serde::Deserialize;
use uuid::Uuid;

use crate::models::invoice::InvoiceResponse;

/// Pagination query parameters.
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

/// GET /api/v1/billing/invoices — list invoices (tenant-scoped, paginated)
pub async fn list_invoices(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    Query(params): Query<PaginationParams>,
) -> Result<impl IntoResponse, AppError> {
    let pagination = Pagination::new(params.page, params.per_page);
    let response =
        invoice_service::list_invoices(&state.db_pool, &tenant_ctx, &pagination).await?;

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

/// GET /api/v1/billing/invoices/:id — get invoice details
pub async fn get_invoice(
    State(state): State<AppState>,
    tenant_ctx: TenantContext,
    Path(id): Path<Uuid>,
) -> Result<impl IntoResponse, AppError> {
    let invoice = invoice_service::get_invoice(&state.db_pool, &tenant_ctx, id).await?;

    Ok(Json(serde_json::json!({
        "success": true,
        "data": InvoiceResponse::from(invoice)
    })))
}
