//! Tenant context extraction and propagation.
//!
//! Mirrors the Go `libs/go/pkg/tenant` package. Provides `TenantContext`
//! that holds `tenant_id` and `org_id`, extracted from HTTP headers set
//! by the API Gateway. Tenant identity MUST come from auth context
//! (headers from gateway), never from request body/query params.

use crate::error::AppError;
use uuid::Uuid;

/// Tenant context carrying the tenant and organization identifiers.
///
/// This is extracted from gateway-forwarded headers (`X-Tenant-ID`, `X-Org-ID`)
/// and stored in Axum request extensions for downstream use.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TenantContext {
    pub tenant_id: Uuid,
    pub org_id: Option<Uuid>,
}

impl TenantContext {
    /// Create a new tenant context with both tenant and org IDs.
    pub fn new(tenant_id: Uuid, org_id: Option<Uuid>) -> Self {
        Self { tenant_id, org_id }
    }

    /// Create a tenant context with only a tenant ID.
    pub fn with_tenant_id(tenant_id: Uuid) -> Self {
        Self {
            tenant_id,
            org_id: None,
        }
    }
}

/// Extract `TenantContext` from Axum request extensions.
///
/// Returns `None` if no tenant context is set (e.g., public endpoints).
pub fn from_extensions(extensions: &http::Extensions) -> Option<&TenantContext> {
    extensions.get::<TenantContext>()
}

/// Require a `TenantContext` from Axum request extensions.
///
/// Returns `AppError::unauthorized` if no tenant context is set.
/// Use this in handlers/repositories that MUST be tenant-scoped.
///
/// # Errors
///
/// Returns `AppError` if tenant context is missing.
pub fn require_tenant(extensions: &http::Extensions) -> Result<&TenantContext, AppError> {
    from_extensions(extensions).ok_or_else(|| {
        AppError::unauthorized("Tenant context is required but not present")
    })
}

/// Extract tenant ID from HTTP headers (typically set by the API Gateway).
///
/// Looks for `X-Tenant-ID` and optionally `X-Org-ID`.
///
/// # Errors
///
/// Returns `AppError` if the `X-Tenant-ID` header is missing or invalid.
pub fn extract_from_headers(
    headers: &http::HeaderMap,
) -> Result<TenantContext, AppError> {
    let tenant_header = headers
        .get("X-Tenant-ID")
        .and_then(|v| v.to_str().ok())
        .ok_or_else(|| AppError::unauthorized("Missing X-Tenant-ID header"))?;

    let tenant_id = Uuid::parse_str(tenant_header).map_err(|_| {
        AppError::bad_request("Invalid X-Tenant-ID header: must be a valid UUID")
    })?;

    let org_id = headers
        .get("X-Org-ID")
        .and_then(|v| v.to_str().ok())
        .and_then(|v| Uuid::parse_str(v).ok());

    Ok(TenantContext::new(tenant_id, org_id))
}

#[cfg(test)]
mod tests {
    use super::*;
    use http::HeaderMap;

    #[test]
    fn test_extract_from_headers_success() {
        let mut headers = HeaderMap::new();
        let tid = Uuid::new_v4();
        let oid = Uuid::new_v4();
        headers.insert("X-Tenant-ID", tid.to_string().parse().unwrap());
        headers.insert("X-Org-ID", oid.to_string().parse().unwrap());

        let ctx = extract_from_headers(&headers).unwrap();
        assert_eq!(ctx.tenant_id, tid);
        assert_eq!(ctx.org_id, Some(oid));
    }

    #[test]
    fn test_extract_from_headers_missing_tenant() {
        let headers = HeaderMap::new();
        let result = extract_from_headers(&headers);
        assert!(result.is_err());
    }

    #[test]
    fn test_extract_from_headers_invalid_uuid() {
        let mut headers = HeaderMap::new();
        headers.insert("X-Tenant-ID", "not-a-uuid".parse().unwrap());
        let result = extract_from_headers(&headers);
        assert!(result.is_err());
    }

    #[test]
    fn test_extract_from_headers_no_org() {
        let mut headers = HeaderMap::new();
        let tid = Uuid::new_v4();
        headers.insert("X-Tenant-ID", tid.to_string().parse().unwrap());

        let ctx = extract_from_headers(&headers).unwrap();
        assert_eq!(ctx.tenant_id, tid);
        assert_eq!(ctx.org_id, None);
    }

    #[test]
    fn test_require_tenant_missing() {
        let extensions = http::Extensions::new();
        let result = require_tenant(&extensions);
        assert!(result.is_err());
    }

    #[test]
    fn test_require_tenant_present() {
        let mut extensions = http::Extensions::new();
        let ctx = TenantContext::with_tenant_id(Uuid::new_v4());
        extensions.insert(ctx.clone());

        let result = require_tenant(&extensions);
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), &ctx);
    }
}
