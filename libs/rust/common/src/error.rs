//! Standardized error types for the platform.
//!
//! Mirrors the Go `libs/go/pkg/errors` package. Provides `AppError` with
//! HTTP status code mapping and automatic Axum response conversion.
//! Error responses follow the platform API standard:
//!
//! ```json
//! {
//!   "success": false,
//!   "error": {
//!     "code": "RESOURCE_NOT_FOUND",
//!     "message": "User not found",
//!     "details": "Optional, non-sensitive details"
//!   }
//! }
//! ```

use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use serde::Serialize;

/// Standardized error codes matching the Go shared library.
pub mod codes {
    pub const BAD_REQUEST: &str = "BAD_REQUEST";
    pub const VALIDATION_ERROR: &str = "VALIDATION_ERROR";
    pub const UNAUTHORIZED: &str = "UNAUTHORIZED";
    pub const FORBIDDEN: &str = "FORBIDDEN";
    pub const RESOURCE_NOT_FOUND: &str = "RESOURCE_NOT_FOUND";
    pub const CONFLICT: &str = "CONFLICT";
    pub const RATE_LIMITED: &str = "RATE_LIMITED";
    pub const INTERNAL_ERROR: &str = "INTERNAL_ERROR";
    pub const SERVICE_UNAVAILABLE: &str = "SERVICE_UNAVAILABLE";
    pub const DATABASE_ERROR: &str = "DATABASE_ERROR";
    pub const EXTERNAL_SERVICE_ERROR: &str = "EXTERNAL_SERVICE_ERROR";
}

/// Application error type for the platform.
///
/// Carries an HTTP status code, a stable error code string, a user-facing
/// message, and optional non-sensitive details. Never exposes stack traces
/// or internal details to clients.
#[derive(Debug, thiserror::Error)]
#[error("{message}")]
pub struct AppError {
    /// HTTP status code for the response.
    pub status: StatusCode,
    /// Stable error code for client consumption.
    pub code: String,
    /// Human-readable error message (safe for clients).
    pub message: String,
    /// Optional non-sensitive details.
    pub details: Option<String>,
    /// Internal error for logging (never sent to client).
    #[source]
    pub source: Option<Box<dyn std::error::Error + Send + Sync>>,
}

/// JSON response body for error responses.
#[derive(Serialize)]
struct ErrorResponseBody {
    success: bool,
    error: ErrorDetail,
}

/// Error detail within the response body.
#[derive(Serialize)]
struct ErrorDetail {
    code: String,
    message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    details: Option<String>,
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        // Log the internal error for debugging (never sent to client)
        if let Some(ref source) = self.source {
            tracing::error!(
                error.code = %self.code,
                error.message = %self.message,
                error.source = %source,
                "Request error"
            );
        }

        let body = ErrorResponseBody {
            success: false,
            error: ErrorDetail {
                code: self.code,
                message: self.message,
                details: self.details,
            },
        };

        (self.status, axum::Json(body)).into_response()
    }
}

// --- Convenience constructors (matching Go helpers) ---

impl AppError {
    /// Create a new `AppError` with full control over all fields.
    pub fn new(status: StatusCode, code: impl Into<String>, message: impl Into<String>) -> Self {
        Self {
            status,
            code: code.into(),
            message: message.into(),
            details: None,
            source: None,
        }
    }

    /// Attach optional non-sensitive details.
    pub fn with_details(mut self, details: impl Into<String>) -> Self {
        self.details = Some(details.into());
        self
    }

    /// Attach an internal source error (logged, never sent to client).
    pub fn with_source(mut self, source: impl std::error::Error + Send + Sync + 'static) -> Self {
        self.source = Some(Box::new(source));
        self
    }

    // --- Named constructors ---

    pub fn bad_request(message: impl Into<String>) -> Self {
        Self::new(StatusCode::BAD_REQUEST, codes::BAD_REQUEST, message)
    }

    pub fn validation_error(message: impl Into<String>) -> Self {
        Self::new(StatusCode::BAD_REQUEST, codes::VALIDATION_ERROR, message)
    }

    pub fn unauthorized(message: impl Into<String>) -> Self {
        Self::new(StatusCode::UNAUTHORIZED, codes::UNAUTHORIZED, message)
    }

    pub fn forbidden(message: impl Into<String>) -> Self {
        Self::new(StatusCode::FORBIDDEN, codes::FORBIDDEN, message)
    }

    pub fn not_found(message: impl Into<String>) -> Self {
        Self::new(StatusCode::NOT_FOUND, codes::RESOURCE_NOT_FOUND, message)
    }

    pub fn conflict(message: impl Into<String>) -> Self {
        Self::new(StatusCode::CONFLICT, codes::CONFLICT, message)
    }

    pub fn rate_limited(message: impl Into<String>) -> Self {
        Self::new(StatusCode::TOO_MANY_REQUESTS, codes::RATE_LIMITED, message)
    }

    pub fn internal(message: impl Into<String>) -> Self {
        Self::new(
            StatusCode::INTERNAL_SERVER_ERROR,
            codes::INTERNAL_ERROR,
            message,
        )
    }

    pub fn service_unavailable(message: impl Into<String>) -> Self {
        Self::new(
            StatusCode::SERVICE_UNAVAILABLE,
            codes::SERVICE_UNAVAILABLE,
            message,
        )
    }

    pub fn database_error(message: impl Into<String>) -> Self {
        Self::new(
            StatusCode::INTERNAL_SERVER_ERROR,
            codes::DATABASE_ERROR,
            message,
        )
    }

    pub fn external_service_error(message: impl Into<String>) -> Self {
        Self::new(
            StatusCode::BAD_GATEWAY,
            codes::EXTERNAL_SERVICE_ERROR,
            message,
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_bad_request_error() {
        let err = AppError::bad_request("Invalid input");
        assert_eq!(err.status, StatusCode::BAD_REQUEST);
        assert_eq!(err.code, codes::BAD_REQUEST);
        assert_eq!(err.message, "Invalid input");
    }

    #[test]
    fn test_not_found_error() {
        let err = AppError::not_found("User not found");
        assert_eq!(err.status, StatusCode::NOT_FOUND);
        assert_eq!(err.code, codes::RESOURCE_NOT_FOUND);
    }

    #[test]
    fn test_error_with_details() {
        let err = AppError::validation_error("Invalid email").with_details("Must be a valid email");
        assert_eq!(err.details.as_deref(), Some("Must be a valid email"));
    }

    #[test]
    fn test_internal_error() {
        let err = AppError::internal("Something went wrong");
        assert_eq!(err.status, StatusCode::INTERNAL_SERVER_ERROR);
        assert_eq!(err.code, codes::INTERNAL_ERROR);
    }

    #[test]
    fn test_error_display() {
        let err = AppError::bad_request("test message");
        assert_eq!(format!("{err}"), "test message");
    }
}
