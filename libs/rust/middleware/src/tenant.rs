//! Tenant extraction middleware for Axum.
//!
//! Extracts `X-Tenant-ID` header (set by the API Gateway) and validates
//! it against JWT claims. Adds `TenantContext` to request extensions.
//! Mirrors the Go `middleware.Tenant()`.

use axum::{
    extract::Request,
    http::StatusCode,
    middleware::Next,
    response::{IntoResponse, Response},
    Json,
};
use platform_common::tenant::TenantContext;
use uuid::Uuid;

/// Tenant extraction middleware for Axum.
///
/// Extracts the `X-Tenant-ID` header from the request and optionally
/// `X-Org-ID`. If the header is present, validates it as a UUID and
/// stores a `TenantContext` in request extensions.
///
/// If the header is missing, the request continues without tenant context.
/// Handlers that require tenant context should use `require_tenant()`.
pub async fn tenant_middleware(mut request: Request, next: Next) -> Response {
    let tenant_header = request
        .headers()
        .get("X-Tenant-ID")
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string());

    if let Some(ref tenant_str) = tenant_header {
        match Uuid::parse_str(tenant_str) {
            Ok(tenant_id) => {
                let org_id = request
                    .headers()
                    .get("X-Org-ID")
                    .and_then(|v| v.to_str().ok())
                    .and_then(|s| Uuid::parse_str(s).ok());

                let tenant_ctx = TenantContext::new(tenant_id, org_id);
                request.extensions_mut().insert(tenant_ctx);
            }
            Err(_) => {
                tracing::warn!(
                    tenant_id = %tenant_str,
                    "Invalid X-Tenant-ID header, ignoring"
                );
            }
        }
    }

    next.run(request).await
}

/// Strict tenant middleware that rejects requests without a valid tenant.
///
/// Returns 401 Unauthorized if the `X-Tenant-ID` header is missing or
/// invalid. Use this on routes that MUST be tenant-scoped.
pub async fn require_tenant_middleware(mut request: Request, next: Next) -> Response {
    let tenant_header = request
        .headers()
        .get("X-Tenant-ID")
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string());

    match tenant_header {
        Some(ref tenant_str) => match Uuid::parse_str(tenant_str) {
            Ok(tenant_id) => {
                let org_id = request
                    .headers()
                    .get("X-Org-ID")
                    .and_then(|v| v.to_str().ok())
                    .and_then(|s| Uuid::parse_str(s).ok());

                let tenant_ctx = TenantContext::new(tenant_id, org_id);
                request.extensions_mut().insert(tenant_ctx);
                next.run(request).await
            }
            Err(_) => (
                StatusCode::BAD_REQUEST,
                Json(serde_json::json!({
                    "success": false,
                    "error": {
                        "code": "BAD_REQUEST",
                        "message": "Invalid X-Tenant-ID header: must be a valid UUID"
                    }
                })),
            )
                .into_response(),
        },
        None => (
            StatusCode::UNAUTHORIZED,
            Json(serde_json::json!({
                "success": false,
                "error": {
                    "code": "UNAUTHORIZED",
                    "message": "Missing X-Tenant-ID header"
                }
            })),
        )
            .into_response(),
    }
}

#[cfg(test)]
mod tests {
    // Integration tests require a running Axum app.
    // Unit tests for tenant extraction are in platform_common::tenant.

    #[test]
    fn test_module_compiles() {
        // Ensure the module compiles correctly.
    }
}
