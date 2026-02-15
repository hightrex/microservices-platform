//! Prometheus metrics module.
//!
//! Provides a metrics registry, common metric types (counter, histogram, gauge),
//! and an Axum handler for the `/metrics` endpoint.

use axum::response::IntoResponse;
use prometheus::{
    Encoder, HistogramOpts, HistogramVec, IntCounterVec, IntGauge, Opts, Registry, TextEncoder,
};
use std::sync::Arc;

/// Metrics registry wrapper for the platform.
///
/// Each service creates one `MetricsRegistry` and registers its metrics.
/// The `/metrics` endpoint uses `render()` to produce Prometheus text format.
#[derive(Clone)]
pub struct MetricsRegistry {
    registry: Arc<Registry>,
}

impl MetricsRegistry {
    /// Create a new metrics registry with a namespace prefix.
    pub fn new(namespace: &str) -> Self {
        let registry = Registry::new_custom(Some(namespace.to_string()), None)
            .unwrap_or_else(|_| Registry::new());
        Self {
            registry: Arc::new(registry),
        }
    }

    /// Register and return a new counter vector.
    ///
    /// # Errors
    ///
    /// Returns an error if the metric cannot be registered (e.g., duplicate name).
    pub fn register_counter_vec(
        &self,
        name: &str,
        help: &str,
        label_names: &[&str],
    ) -> Result<IntCounterVec, prometheus::Error> {
        let opts = Opts::new(name, help);
        let counter = IntCounterVec::new(opts, label_names)?;
        self.registry.register(Box::new(counter.clone()))?;
        Ok(counter)
    }

    /// Register and return a new histogram vector.
    ///
    /// # Errors
    ///
    /// Returns an error if the metric cannot be registered.
    pub fn register_histogram_vec(
        &self,
        name: &str,
        help: &str,
        label_names: &[&str],
        buckets: Vec<f64>,
    ) -> Result<HistogramVec, prometheus::Error> {
        let opts = HistogramOpts::new(name, help).buckets(buckets);
        let histogram = HistogramVec::new(opts, label_names)?;
        self.registry.register(Box::new(histogram.clone()))?;
        Ok(histogram)
    }

    /// Register and return a new gauge.
    ///
    /// # Errors
    ///
    /// Returns an error if the metric cannot be registered.
    pub fn register_gauge(
        &self,
        name: &str,
        help: &str,
    ) -> Result<IntGauge, prometheus::Error> {
        let opts = Opts::new(name, help);
        let gauge = IntGauge::with_opts(opts)?;
        self.registry.register(Box::new(gauge.clone()))?;
        Ok(gauge)
    }

    /// Render all metrics in Prometheus text exposition format.
    pub fn render(&self) -> String {
        let encoder = TextEncoder::new();
        let metric_families = self.registry.gather();
        let mut buffer = Vec::new();
        encoder
            .encode(&metric_families, &mut buffer)
            .unwrap_or_default();
        String::from_utf8(buffer).unwrap_or_default()
    }

    /// Create an Axum handler for the `/metrics` endpoint.
    pub fn handler(&self) -> impl IntoResponse + Clone {
        let output = self.render();
        (
            [(
                http::header::CONTENT_TYPE,
                "text/plain; version=0.0.4; charset=utf-8",
            )],
            output,
        )
    }
}

/// Default HTTP request duration buckets (in seconds).
pub fn default_duration_buckets() -> Vec<f64> {
    vec![0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_register_counter_vec() {
        let registry = MetricsRegistry::new("test");
        let counter = registry
            .register_counter_vec("requests_total", "Total requests", &["method", "status"])
            .unwrap();

        counter.with_label_values(&["GET", "200"]).inc();
        counter.with_label_values(&["GET", "200"]).inc();
        counter.with_label_values(&["POST", "201"]).inc();

        let output = registry.render();
        assert!(output.contains("test_requests_total"));
    }

    #[test]
    fn test_register_histogram_vec() {
        let registry = MetricsRegistry::new("test");
        let histogram = registry
            .register_histogram_vec(
                "request_duration_seconds",
                "Request duration",
                &["method"],
                default_duration_buckets(),
            )
            .unwrap();

        histogram.with_label_values(&["GET"]).observe(0.05);

        let output = registry.render();
        assert!(output.contains("test_request_duration_seconds"));
    }

    #[test]
    fn test_register_gauge() {
        let registry = MetricsRegistry::new("test");
        let gauge = registry
            .register_gauge("active_connections", "Active connections")
            .unwrap();

        gauge.set(42);

        let output = registry.render();
        assert!(output.contains("test_active_connections 42"));
    }
}
