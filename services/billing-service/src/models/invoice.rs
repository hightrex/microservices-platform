//! Invoice domain models.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::Type;
use uuid::Uuid;

/// Invoice status enum.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Type)]
#[sqlx(type_name = "invoice_status", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum InvoiceStatus {
    Draft,
    Open,
    Paid,
    Void,
    Uncollectible,
}

/// Database model for invoices.
#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Invoice {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub subscription_id: Uuid,
    pub stripe_invoice_id: Option<String>,
    pub invoice_number: String,
    pub status: InvoiceStatus,
    pub amount_due_cents: i64,
    pub amount_paid_cents: i64,
    pub currency: String,
    pub due_date: Option<DateTime<Utc>>,
    pub paid_at: Option<DateTime<Utc>>,
    pub pdf_url: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Public invoice response.
#[derive(Debug, Clone, Serialize)]
pub struct InvoiceResponse {
    pub id: Uuid,
    pub invoice_number: String,
    pub status: InvoiceStatus,
    pub amount_due_cents: i64,
    pub amount_paid_cents: i64,
    pub currency: String,
    pub due_date: Option<DateTime<Utc>>,
    pub paid_at: Option<DateTime<Utc>>,
    pub pdf_url: Option<String>,
    pub created_at: DateTime<Utc>,
}

impl From<Invoice> for InvoiceResponse {
    fn from(i: Invoice) -> Self {
        Self {
            id: i.id,
            invoice_number: i.invoice_number,
            status: i.status,
            amount_due_cents: i.amount_due_cents,
            amount_paid_cents: i.amount_paid_cents,
            currency: i.currency,
            due_date: i.due_date,
            paid_at: i.paid_at,
            pdf_url: i.pdf_url,
            created_at: i.created_at,
        }
    }
}
