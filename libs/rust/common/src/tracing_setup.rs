//! OpenTelemetry tracing setup module.
//!
//! Provides helpers for initializing OpenTelemetry with Jaeger exporter
//! and context propagation. This module is designed to be optional:
//! services can use basic `tracing` without OpenTelemetry, or enable
//! full distributed tracing by calling `init_tracing()`.
//!
//! For now, provides span creation helpers and context propagation
//! utilities that work with the standard `tracing` crate.

use tracing::Span;
use uuid::Uuid;

/// Tracing configuration.
#[derive(Debug, Clone, serde::Deserialize)]
pub struct TracingConfig {
    /// Whether OpenTelemetry tracing is enabled.
    #[serde(default)]
    pub enabled: bool,
    /// Service name for traces.
    #[serde(default = "default_service_name")]
    pub service_name: String,
    /// Jaeger/OTLP endpoint (e.g., "http://localhost:4317").
    #[serde(default)]
    pub endpoint: String,
    /// Sampling ratio (0.0 to 1.0, default 1.0 = sample everything).
    #[serde(default = "default_sample_ratio")]
    pub sample_ratio: f64,
}

fn default_service_name() -> String {
    "unknown-service".to_string()
}

fn default_sample_ratio() -> f64 {
    1.0
}

impl Default for TracingConfig {
    fn default() -> Self {
        Self {
            enabled: false,
            service_name: default_service_name(),
            endpoint: String::new(),
            sample_ratio: default_sample_ratio(),
        }
    }
}

/// Create a new span for an operation with request context.
///
/// This is a helper that creates consistently-structured spans
/// across all services, including tenant_id and correlation_id.
pub fn create_span(
    operation: &str,
    tenant_id: Option<Uuid>,
    correlation_id: Option<&str>,
) -> Span {
    let tid = tenant_id
        .map(|id| id.to_string())
        .unwrap_or_default();
    let cid = correlation_id.unwrap_or("");

    tracing::info_span!(
        "operation",
        otel.name = %operation,
        tenant_id = %tid,
        correlation_id = %cid,
    )
}

/// Generate a new correlation ID for request tracing.
pub fn new_correlation_id() -> String {
    Uuid::new_v4().to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_tracing_config() {
        let cfg = TracingConfig::default();
        assert!(!cfg.enabled);
        assert_eq!(cfg.service_name, "unknown-service");
        assert!(cfg.endpoint.is_empty());
    }

    #[test]
    fn test_new_correlation_id() {
        let id = new_correlation_id();
        assert!(Uuid::parse_str(&id).is_ok());
    }
}
