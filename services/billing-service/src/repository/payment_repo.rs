//! Payment repository — all queries tenant-scoped.

use crate::models::payment::Payment;
use platform_common::error::AppError;
use platform_database::repository::Pagination;
use sqlx::PgPool;
use uuid::Uuid;

/// Create a new payment record (tenant-scoped).
pub async fn create(pool: &PgPool, payment: &Payment) -> Result<Payment, AppError> {
    sqlx::query_as::<_, Payment>(
        r#"INSERT INTO payments
            (id, tenant_id, invoice_id, stripe_payment_intent_id,
             amount_cents, currency, status, payment_method_type,
             receipt_url, failure_reason, paid_at)
           VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
           RETURNING *"#,
    )
    .bind(payment.id)
    .bind(payment.tenant_id)
    .bind(payment.invoice_id)
    .bind(&payment.stripe_payment_intent_id)
    .bind(payment.amount_cents)
    .bind(&payment.currency)
    .bind(payment.status)
    .bind(&payment.payment_method_type)
    .bind(&payment.receipt_url)
    .bind(&payment.failure_reason)
    .bind(payment.paid_at)
    .fetch_one(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to create payment").with_source(e))
}

/// List payments for a tenant with pagination.
pub async fn list_by_tenant(
    pool: &PgPool,
    tenant_id: Uuid,
    pagination: &Pagination,
) -> Result<(Vec<Payment>, i64), AppError> {
    let total: (i64,) =
        sqlx::query_as("SELECT COUNT(*) FROM payments WHERE tenant_id = $1")
            .bind(tenant_id)
            .fetch_one(pool)
            .await
            .map_err(|e| AppError::database_error("Failed to count payments").with_source(e))?;

    let payments = sqlx::query_as::<_, Payment>(
        "SELECT * FROM payments WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
    )
    .bind(tenant_id)
    .bind(pagination.limit())
    .bind(pagination.offset())
    .fetch_all(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to list payments").with_source(e))?;

    Ok((payments, total.0))
}

/// Get payments for an invoice.
pub async fn get_by_invoice(
    pool: &PgPool,
    invoice_id: Uuid,
    tenant_id: Uuid,
) -> Result<Vec<Payment>, AppError> {
    sqlx::query_as::<_, Payment>(
        "SELECT * FROM payments WHERE invoice_id = $1 AND tenant_id = $2 ORDER BY created_at DESC",
    )
    .bind(invoice_id)
    .bind(tenant_id)
    .fetch_all(pool)
    .await
    .map_err(|e| AppError::database_error("Failed to get invoice payments").with_source(e))
}
