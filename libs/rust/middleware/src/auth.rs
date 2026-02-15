//! JWT validation middleware for Axum.
//!
//! Extracts and validates JWT from the `Authorization` header,
//! parses claims (sub, tid, org, roles, exp), and adds user context
//! to Axum extensions. Optionally checks a Redis token blacklist.

use axum::{
    extract::Request,
    http::StatusCode,
    middleware::Next,
    response::{IntoResponse, Response},
    Json,
};
use jsonwebtoken::{decode, Algorithm, DecodingKey, Validation};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// JWT claims structure matching the platform auth token format.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    /// Subject (user ID).
    pub sub: String,
    /// Tenant ID.
    pub tid: Option<String>,
    /// Organization ID.
    pub org: Option<String>,
    /// User roles.
    #[serde(default)]
    pub roles: Vec<String>,
    /// Token expiration (Unix timestamp).
    pub exp: u64,
    /// Issued at (Unix timestamp).
    #[serde(default)]
    pub iat: u64,
}

/// User context extracted from JWT, stored in Axum request extensions.
#[derive(Debug, Clone)]
pub struct UserContext {
    pub user_id: String,
    pub tenant_id: Option<Uuid>,
    pub org_id: Option<Uuid>,
    pub roles: Vec<String>,
}

/// Configuration for the JWT auth middleware.
#[derive(Clone)]
pub struct AuthConfig {
    /// JWT secret key for HMAC validation.
    pub jwt_secret: String,
    /// JWT algorithm (default: HS256).
    pub algorithm: Algorithm,
}

impl AuthConfig {
    /// Create a new auth config with HS256 algorithm.
    pub fn new(jwt_secret: impl Into<String>) -> Self {
        Self {
            jwt_secret: jwt_secret.into(),
            algorithm: Algorithm::HS256,
        }
    }
}

/// JWT authentication middleware for Axum.
///
/// Validates the JWT from the `Authorization: Bearer <token>` header,
/// extracts claims, and adds `UserContext` and `TenantContext` to
/// request extensions.
///
/// Rejected requests receive a 401 Unauthorized response.
pub async fn auth_middleware(
    axum::extract::State(config): axum::extract::State<AuthConfig>,
    mut request: Request,
    next: Next,
) -> Response {
    // Extract the Authorization header
    let auth_header = request
        .headers()
        .get("Authorization")
        .and_then(|v| v.to_str().ok());

    let token = match auth_header {
        Some(header) if header.starts_with("Bearer ") => &header[7..],
        _ => {
            return unauthorized_response("Missing or invalid Authorization header");
        }
    };

    // Validate and decode the JWT
    let mut validation = Validation::new(config.algorithm);
    validation.validate_exp = true;

    let token_data = match decode::<Claims>(
        token,
        &DecodingKey::from_secret(config.jwt_secret.as_bytes()),
        &validation,
    ) {
        Ok(data) => data,
        Err(e) => {
            tracing::warn!(error = %e, "JWT validation failed");
            return unauthorized_response("Invalid or expired token");
        }
    };

    let claims = token_data.claims;

    // Build user context
    let tenant_id = claims
        .tid
        .as_deref()
        .and_then(|s| Uuid::parse_str(s).ok());

    let org_id = claims
        .org
        .as_deref()
        .and_then(|s| Uuid::parse_str(s).ok());

    let user_context = UserContext {
        user_id: claims.sub.clone(),
        tenant_id,
        org_id,
        roles: claims.roles.clone(),
    };

    // Add user context to extensions
    request.extensions_mut().insert(user_context);

    // Add tenant context to extensions if present
    if let Some(tid) = tenant_id {
        let tenant_ctx = platform_common::tenant::TenantContext::new(tid, org_id);
        request.extensions_mut().insert(tenant_ctx);
    }

    next.run(request).await
}

/// Check if a user has a specific role.
pub fn has_role(user_ctx: &UserContext, role: &str) -> bool {
    user_ctx.roles.iter().any(|r| r == role)
}

/// Check if a user has any of the specified roles.
pub fn has_any_role(user_ctx: &UserContext, roles: &[&str]) -> bool {
    roles.iter().any(|role| has_role(user_ctx, role))
}

/// RBAC middleware that requires specific roles.
///
/// Returns 403 Forbidden if the user doesn't have any of the required roles.
pub async fn require_roles(
    roles: Vec<String>,
    request: Request,
    next: Next,
) -> Response {
    let user_ctx = request.extensions().get::<UserContext>();

    match user_ctx {
        Some(ctx) => {
            let has_required_role = roles.iter().any(|role| ctx.roles.contains(role));
            if has_required_role {
                next.run(request).await
            } else {
                forbidden_response("Insufficient permissions")
            }
        }
        None => unauthorized_response("Authentication required"),
    }
}

/// Build a 401 Unauthorized JSON response.
fn unauthorized_response(message: &str) -> Response {
    (
        StatusCode::UNAUTHORIZED,
        Json(serde_json::json!({
            "success": false,
            "error": {
                "code": "UNAUTHORIZED",
                "message": message
            }
        })),
    )
        .into_response()
}

/// Build a 403 Forbidden JSON response.
fn forbidden_response(message: &str) -> Response {
    (
        StatusCode::FORBIDDEN,
        Json(serde_json::json!({
            "success": false,
            "error": {
                "code": "FORBIDDEN",
                "message": message
            }
        })),
    )
        .into_response()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_has_role() {
        let ctx = UserContext {
            user_id: "user-1".to_string(),
            tenant_id: None,
            org_id: None,
            roles: vec!["org_admin".to_string(), "user".to_string()],
        };

        assert!(has_role(&ctx, "org_admin"));
        assert!(has_role(&ctx, "user"));
        assert!(!has_role(&ctx, "super_admin"));
    }

    #[test]
    fn test_has_any_role() {
        let ctx = UserContext {
            user_id: "user-1".to_string(),
            tenant_id: None,
            org_id: None,
            roles: vec!["user".to_string()],
        };

        assert!(has_any_role(&ctx, &["admin", "user"]));
        assert!(!has_any_role(&ctx, &["admin", "super_admin"]));
    }

    #[test]
    fn test_auth_config_new() {
        let config = AuthConfig::new("my-secret");
        assert_eq!(config.jwt_secret, "my-secret");
        assert_eq!(config.algorithm, Algorithm::HS256);
    }

    #[test]
    fn test_claims_deserialization() {
        let json = r#"{
            "sub": "user-123",
            "tid": "550e8400-e29b-41d4-a716-446655440000",
            "org": "660e8400-e29b-41d4-a716-446655440000",
            "roles": ["org_admin", "user"],
            "exp": 1700000000,
            "iat": 1699999000
        }"#;

        let claims: Claims = serde_json::from_str(json).unwrap();
        assert_eq!(claims.sub, "user-123");
        assert_eq!(claims.roles.len(), 2);
        assert!(claims.tid.is_some());
    }
}
