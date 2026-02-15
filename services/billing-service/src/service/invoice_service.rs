//! Invoice service — business logic for invoicing.

use crate::models::invoice::{Invoice, InvoiceResponse};
use crate::repository::invoice_repo;
use platform_common::error::AppError;
use platform_common::tenant::TenantContext;
use platform_database::repository::{PaginatedResponse, Pagination};
use sqlx::PgPool;
use uuid::Uuid;

/// List invoices for a tenant with pagination.
pub async fn list_invoices(
    pool: &PgPool,
    tenant_ctx: &TenantContext,
    pagination: &Pagination,
) -> Result<PaginatedResponse<InvoiceResponse>, AppError> {
    let (invoices, total) =
        invoice_repo::list_by_tenant(pool, tenant_ctx.tenant_id, pagination).await?;

    let responses: Vec<InvoiceResponse> = invoices.into_iter().map(InvoiceResponse::from).collect();

    Ok(PaginatedResponse::new(responses, pagination, total))
}

/// Get an invoice by ID (tenant-scoped).
pub async fn get_invoice(
    pool: &PgPool,
    tenant_ctx: &TenantContext,
    invoice_id: Uuid,
) -> Result<Invoice, AppError> {
    invoice_repo::get_by_id(pool, invoice_id, tenant_ctx.tenant_id)
        .await?
        .ok_or_else(|| AppError::not_found("Invoice not found"))
}
