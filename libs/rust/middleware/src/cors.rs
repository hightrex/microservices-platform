//! CORS middleware configuration for Axum.
//!
//! Per foundation rules: No `*` origins. Uses explicit allowlists.
//! Wraps `tower-http` CORS layer with platform-safe defaults.

use http::HeaderValue;
use tower_http::cors::{AllowOrigin, CorsLayer};

/// Create a CORS middleware layer with the given allowed origins.
///
/// Per foundation rules, wildcard `*` origins are forbidden.
/// If no origins are provided, CORS requests are denied by default
/// (fail-closed behavior).
///
/// # Arguments
///
/// * `allowed_origins` - Explicit list of allowed origins.
///   Example: `["http://localhost:3000", "https://app.example.com"]`
pub fn create_cors_layer(allowed_origins: &[String]) -> CorsLayer {
    if allowed_origins.is_empty() {
        tracing::warn!("No CORS origins configured; all cross-origin requests will be denied");
        return CorsLayer::new();
    }

    let origins: Vec<HeaderValue> = allowed_origins
        .iter()
        .filter_map(|origin| {
            origin.parse::<HeaderValue>().ok().or_else(|| {
                tracing::warn!(origin = %origin, "Invalid CORS origin, skipping");
                None
            })
        })
        .collect();

    CorsLayer::new()
        .allow_origin(AllowOrigin::list(origins))
        .allow_methods([
            http::Method::GET,
            http::Method::POST,
            http::Method::PUT,
            http::Method::DELETE,
            http::Method::PATCH,
            http::Method::OPTIONS,
        ])
        .allow_headers([
            http::header::CONTENT_TYPE,
            http::header::AUTHORIZATION,
            http::header::ACCEPT,
            http::HeaderName::from_static("x-tenant-id"),
            http::HeaderName::from_static("x-request-id"),
            http::HeaderName::from_static("x-org-id"),
        ])
        .expose_headers([
            http::HeaderName::from_static("x-request-id"),
            http::HeaderName::from_static("x-ratelimit-limit"),
            http::HeaderName::from_static("x-ratelimit-remaining"),
            http::HeaderName::from_static("x-ratelimit-reset"),
        ])
        .max_age(std::time::Duration::from_secs(3600))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_create_cors_layer_with_origins() {
        let origins = vec![
            "http://localhost:3000".to_string(),
            "https://app.example.com".to_string(),
        ];
        // Should not panic
        let _layer = create_cors_layer(&origins);
    }

    #[test]
    fn test_create_cors_layer_empty_origins() {
        let origins: Vec<String> = vec![];
        // Should not panic; creates a restrictive layer
        let _layer = create_cors_layer(&origins);
    }
}
