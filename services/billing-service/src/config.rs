//! Billing service configuration.

use platform_common::config::{DatabaseConfig, RedisConfig, ServerConfig};
use platform_common::logger::LoggerConfig;
use serde::Deserialize;

/// Top-level billing service configuration.
#[derive(Debug, Clone, Deserialize)]
pub struct BillingConfig {
    pub server: ServerConfig,
    pub database: DatabaseConfig,
    pub redis: RedisConfig,
    pub stripe: StripeConfig,
    pub logger: LoggerConfig,
    pub invoice: InvoiceConfig,
    pub auth: AuthenticationConfig,
}

/// Authentication configuration for JWT validation.
#[derive(Debug, Clone, Deserialize)]
pub struct AuthenticationConfig {
    /// JWT signing secret (shared with auth-service).
    pub jwt_secret: String,
}

/// Stripe integration configuration.
#[derive(Debug, Clone, Deserialize)]
pub struct StripeConfig {
    /// Stripe secret API key (sk_test_... or sk_live_...).
    pub secret_key: String,
    /// Stripe webhook signing secret (whsec_...).
    pub webhook_secret: String,
    /// Stripe publishable key (pk_test_... or pk_live_...).
    #[serde(default)]
    pub publishable_key: String,
    /// Whether to use Stripe test mode.
    #[serde(default = "default_test_mode")]
    pub test_mode: bool,
}

fn default_test_mode() -> bool {
    true
}

/// Invoice configuration.
#[derive(Debug, Clone, Deserialize)]
pub struct InvoiceConfig {
    /// Number of days until invoice is due.
    #[serde(default = "default_due_days")]
    pub due_days: u32,
    /// Late payment fee percentage.
    #[serde(default = "default_late_fee")]
    pub late_fee_percentage: f64,
}

fn default_due_days() -> u32 {
    30
}

fn default_late_fee() -> f64 {
    1.5
}
