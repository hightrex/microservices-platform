//! Invoice repository — all queries tenant-scoped.

use crate::models::invoice::{Invoice, InvoiceStatus};
use platform_common::error::AppError;
use platform_database::repository::Pagination;
use sqlx::PgPool;
use uuid::Uuid;

/// Create a new invoice (tenant-scoped).
pub async fn create(pool: &PgPool, invoice: &Invoice) -> Result<Invoice, AppError> {
    sqlx::query_as::<_, Invoice>(
        r#"INSERT INTO invoices
            (id, tenant_id, subscription_id, stripe_invoice_id, invoice_number,
             status, amount_due_cents, amount_paid_cents, currency, due_date, paid_at, pdf_url)
           VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
           RETURNING *"#,
    )
    .bind(invoice.id)
    .bind(invoice.tenant_id)
    .bind(invoice.subscription_id)
    .bind(&invoice.stripe_invoice_id)
    .bind(&invoice.invoice_number)
    .bind(invoice.status)
    .bind(invoice.amount_due_cents)
    .bind(invoice.amount_paid_cents)
    .bind(&invoice.currency)
    .bind(invoice.due_date)
    .bind(invoice.paid_at)
    .bind(&invoice.pdf_url)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to create invoice").with_source(e))
}

/// Get invoice by ID (tenant-scoped).
pub async fn get_by_id(
    pool: &PgPool,
    id: Uuid,
    tenant_id: Uuid,
) -> Result<Option<Invoice>, AppError> {
    sqlx::query_as::<_, Invoice>(
        "SELECT * FROM invoices WHERE id = $1 AND tenant_id = $2",
    )
    .bind(id)
    .bind(tenant_id)
    .fetch_optional(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to get invoice").with_source(e))
}

/// List invoices for a tenant with pagination.
pub async fn list_by_tenant(
    pool: &PgPool,
    tenant_id: Uuid,
    pagination: &Pagination,
) -> Result<(Vec<Invoice>, i64), AppError> {
    let total: (i64,) =
        sqlx::query_as("SELECT COUNT(*) FROM invoices WHERE tenant_id = $1")
            .bind(tenant_id)
            .fetch_one(pool)
            .await
            .map_err(|e| AppError::database_error("Failed to count invoices").with_source(e))?;

    let invoices = sqlx::query_as::<_, Invoice>(
        "SELECT * FROM invoices WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
    )
    .bind(tenant_id)
    .bind(pagination.limit())
    .bind(pagination.offset())
    .fetch_all(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to list invoices").with_source(e))?;

    Ok((invoices, total.0))
}

/// Update invoice status.
pub async fn update_status(
    pool: &PgPool,
    id: Uuid,
    tenant_id: Uuid,
    status: InvoiceStatus,
) -> Result<Invoice, AppError> {
    sqlx::query_as::<_, Invoice>(
        "UPDATE invoices SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3 RETURNING *",
    )
    .bind(status)
    .bind(id)
    .bind(tenant_id)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to update invoice status").with_source(e))
}
