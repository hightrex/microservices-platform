//! Request/response logging middleware for Axum.
//!
//! Logs request method, path, status, duration, request ID,
//! tenant ID, and user ID. Mirrors the Go `middleware.RequestLogger()`.

use axum::{extract::Request, middleware::Next, response::Response};
use std::time::Instant;
use uuid::Uuid;

/// Request logging middleware for Axum.
///
/// Logs every request with structured fields including method, path,
/// status code, duration, and extracted context (tenant, user, request ID).
pub async fn request_logger(request: Request, next: Next) -> Response {
    let start = Instant::now();

    let method = request.method().to_string();
    let path = request.uri().path().to_string();
    let query = request.uri().query().unwrap_or("").to_string();

    // Extract request ID (from header or generate new one)
    let request_id = request
        .headers()
        .get("X-Request-ID")
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string())
        .unwrap_or_else(|| Uuid::new_v4().to_string());

    // Extract tenant and user IDs from headers
    let tenant_id = request
        .headers()
        .get("X-Tenant-ID")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("-")
        .to_string();

    let user_id = request
        .headers()
        .get("X-User-ID")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("-")
        .to_string();

    let response = next.run(request).await;

    let duration = start.elapsed();
    let status = response.status().as_u16();

    tracing::info!(
        method = %method,
        path = %path,
        query = %query,
        status = status,
        duration_ms = duration.as_millis() as u64,
        request_id = %request_id,
        tenant_id = %tenant_id,
        user_id = %user_id,
        "Request completed"
    );

    response
}
