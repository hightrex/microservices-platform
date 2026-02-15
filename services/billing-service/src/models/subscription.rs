//! Subscription domain models.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::Type;
use uuid::Uuid;
use validator::Validate;

/// Subscription status enum stored in Postgres.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Type)]
#[sqlx(type_name = "subscription_status", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum SubscriptionStatus {
    Active,
    PastDue,
    Canceled,
    Paused,
    Trialing,
}

/// Database model for subscriptions.
#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Subscription {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub org_id: Uuid,
    pub plan_id: Uuid,
    pub stripe_subscription_id: Option<String>,
    pub status: SubscriptionStatus,
    pub current_period_start: DateTime<Utc>,
    pub current_period_end: DateTime<Utc>,
    pub cancel_at: Option<DateTime<Utc>>,
    pub canceled_at: Option<DateTime<Utc>>,
    pub trial_end: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Request to create a new subscription.
#[derive(Debug, Clone, Deserialize, Validate)]
pub struct CreateSubscriptionRequest {
    #[validate(length(min = 1, message = "plan_id is required"))]
    pub plan_id: String,
}

/// Request to update (upgrade/downgrade) a subscription.
#[derive(Debug, Clone, Deserialize, Validate)]
pub struct UpdateSubscriptionRequest {
    #[validate(length(min = 1, message = "new_plan_id is required"))]
    pub new_plan_id: String,
    /// If true, apply change immediately with proration.
    /// If false, schedule change for end of billing period.
    #[serde(default)]
    pub immediate: bool,
}

/// Request to cancel a subscription.
#[derive(Debug, Clone, Deserialize)]
pub struct CancelSubscriptionRequest {
    /// If true, cancel immediately. If false, cancel at end of period.
    #[serde(default)]
    pub immediate: bool,
}

/// Public subscription response (omits internal fields).
#[derive(Debug, Clone, Serialize)]
pub struct SubscriptionResponse {
    pub id: Uuid,
    pub plan_id: Uuid,
    pub status: SubscriptionStatus,
    pub current_period_start: DateTime<Utc>,
    pub current_period_end: DateTime<Utc>,
    pub cancel_at: Option<DateTime<Utc>>,
    pub canceled_at: Option<DateTime<Utc>>,
    pub trial_end: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

impl From<Subscription> for SubscriptionResponse {
    fn from(s: Subscription) -> Self {
        Self {
            id: s.id,
            plan_id: s.plan_id,
            status: s.status,
            current_period_start: s.current_period_start,
            current_period_end: s.current_period_end,
            cancel_at: s.cancel_at,
            canceled_at: s.canceled_at,
            trial_end: s.trial_end,
            created_at: s.created_at,
        }
    }
}
