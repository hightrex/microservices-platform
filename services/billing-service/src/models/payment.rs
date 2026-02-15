//! Payment domain models.
//!
//! CRITICAL: NEVER store card numbers, CVV, or full PANs.
//! Only reference Stripe payment intent IDs.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::Type;
use uuid::Uuid;

/// Payment status enum.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Type)]
#[sqlx(type_name = "payment_status", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum PaymentStatus {
    Succeeded,
    Pending,
    Failed,
}

/// Database model for payments.
/// NEVER contains card data — only references to Stripe objects.
#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Payment {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub invoice_id: Option<Uuid>,
    pub stripe_payment_intent_id: Option<String>,
    pub amount_cents: i64,
    pub currency: String,
    pub status: PaymentStatus,
    pub payment_method_type: Option<String>,
    pub receipt_url: Option<String>,
    pub failure_reason: Option<String>,
    pub paid_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

/// Public payment response (no sensitive data).
#[derive(Debug, Clone, Serialize)]
pub struct PaymentResponse {
    pub id: Uuid,
    pub amount_cents: i64,
    pub currency: String,
    pub status: PaymentStatus,
    pub payment_method_type: Option<String>,
    pub receipt_url: Option<String>,
    pub paid_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

impl From<Payment> for PaymentResponse {
    fn from(p: Payment) -> Self {
        Self {
            id: p.id,
            amount_cents: p.amount_cents,
            currency: p.currency,
            status: p.status,
            payment_method_type: p.payment_method_type,
            receipt_url: p.receipt_url,
            paid_at: p.paid_at,
            created_at: p.created_at,
        }
    }
}
