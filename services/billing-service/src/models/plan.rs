//! Plan domain models.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::Type;
use uuid::Uuid;

/// Billing interval for a plan.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Type)]
#[sqlx(type_name = "billing_interval", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum BillingInterval {
    Month,
    Year,
}

/// Database model for billing plans.
#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Plan {
    pub id: Uuid,
    pub name: String,
    pub stripe_price_id: Option<String>,
    pub billing_interval: BillingInterval,
    pub price_cents: i64,
    pub currency: String,
    pub features: serde_json::Value,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Public plan response.
#[derive(Debug, Clone, Serialize)]
pub struct PlanResponse {
    pub id: Uuid,
    pub name: String,
    pub billing_interval: BillingInterval,
    pub price_cents: i64,
    pub currency: String,
    pub features: serde_json::Value,
}

impl From<Plan> for PlanResponse {
    fn from(p: Plan) -> Self {
        Self {
            id: p.id,
            name: p.name,
            billing_interval: p.billing_interval,
            price_cents: p.price_cents,
            currency: p.currency,
            features: p.features,
        }
    }
}
