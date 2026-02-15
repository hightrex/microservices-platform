//! Prometheus metrics middleware for Axum.
//!
//! Tracks request duration histogram, request counter by method/path/status,
//! and active request gauge.

use axum::{extract::Request, middleware::Next, response::Response};
use prometheus::{HistogramVec, IntCounterVec, IntGauge};
use std::sync::Arc;
use std::time::Instant;

/// Metrics state shared across all requests.
#[derive(Clone)]
pub struct RequestMetrics {
    /// Request duration histogram (method, path, status).
    pub request_duration: HistogramVec,
    /// Request counter (method, path, status).
    pub request_count: IntCounterVec,
    /// Active request gauge.
    pub active_requests: IntGauge,
}

impl RequestMetrics {
    /// Create request metrics from a `MetricsRegistry`.
    ///
    /// # Errors
    ///
    /// Returns an error if metric registration fails.
    pub fn new(
        registry: &platform_common::metrics::MetricsRegistry,
    ) -> Result<Self, prometheus::Error> {
        let request_duration = registry.register_histogram_vec(
            "http_request_duration_seconds",
            "HTTP request duration in seconds",
            &["method", "path", "status"],
            platform_common::metrics::default_duration_buckets(),
        )?;

        let request_count = registry.register_counter_vec(
            "http_requests_total",
            "Total HTTP requests",
            &["method", "path", "status"],
        )?;

        let active_requests = registry.register_gauge(
            "http_active_requests",
            "Number of active HTTP requests",
        )?;

        Ok(Self {
            request_duration,
            request_count,
            active_requests,
        })
    }
}

/// Prometheus metrics middleware for Axum.
///
/// Records request duration, increments request counter, and tracks
/// active request gauge. Normalizes path patterns to avoid high-cardinality
/// label values (e.g., `/api/v1/users/:id` instead of `/api/v1/users/abc-123`).
pub async fn metrics_middleware(
    axum::extract::State(metrics): axum::extract::State<Arc<RequestMetrics>>,
    request: Request,
    next: Next,
) -> Response {
    let start = Instant::now();
    let method = request.method().to_string();

    // Use the matched route pattern if available, falling back to the raw path.
    // This avoids cardinality explosion from dynamic path segments.
    let path = request.uri().path().to_string();

    metrics.active_requests.inc();

    let response = next.run(request).await;

    let duration = start.elapsed().as_secs_f64();
    let status = response.status().as_u16().to_string();

    // Normalize path to prevent high-cardinality label explosion.
    // Replace UUID-like segments and numeric IDs with placeholders.
    let normalized_path = normalize_path(&path);

    metrics
        .request_duration
        .with_label_values(&[&method, &normalized_path, &status])
        .observe(duration);

    metrics
        .request_count
        .with_label_values(&[&method, &normalized_path, &status])
        .inc();

    metrics.active_requests.dec();

    response
}

/// Normalize a request path to prevent high-cardinality metric labels.
///
/// Replaces UUID segments and numeric IDs with `:id` placeholders.
fn normalize_path(path: &str) -> String {
    path.split('/')
        .map(|segment| {
            if segment.is_empty() {
                return segment.to_string();
            }
            // Replace UUID-like segments
            if uuid::Uuid::parse_str(segment).is_ok() {
                return ":id".to_string();
            }
            // Replace pure numeric segments
            if segment.chars().all(|c| c.is_ascii_digit()) && !segment.is_empty() {
                return ":id".to_string();
            }
            segment.to_string()
        })
        .collect::<Vec<_>>()
        .join("/")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_normalize_path_uuid() {
        let path = "/api/v1/users/550e8400-e29b-41d4-a716-446655440000";
        assert_eq!(normalize_path(path), "/api/v1/users/:id");
    }

    #[test]
    fn test_normalize_path_numeric() {
        let path = "/api/v1/invoices/12345";
        assert_eq!(normalize_path(path), "/api/v1/invoices/:id");
    }

    #[test]
    fn test_normalize_path_no_ids() {
        let path = "/api/v1/billing/plans";
        assert_eq!(normalize_path(path), "/api/v1/billing/plans");
    }

    #[test]
    fn test_normalize_path_mixed() {
        let path = "/api/v1/orgs/550e8400-e29b-41d4-a716-446655440000/users/42";
        assert_eq!(normalize_path(path), "/api/v1/orgs/:id/users/:id");
    }
}
