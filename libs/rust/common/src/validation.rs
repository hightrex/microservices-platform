//! Input validation module.
//!
//! Provides a generic `validate_request()` function that integrates with
//! the `validator` crate derive macros. Validation errors are formatted
//! to match the platform API error standard.

use crate::error::AppError;
use validator::Validate;

/// Validate a request struct using `validator` derive macros.
///
/// Returns `Ok(())` if validation passes, or an `AppError::validation_error`
/// with formatted details listing all failed constraints.
///
/// # Example
///
/// ```ignore
/// use validator::Validate;
///
/// #[derive(Validate)]
/// struct CreateUserRequest {
///     #[validate(email)]
///     email: String,
///     #[validate(length(min = 8, max = 128))]
///     password: String,
/// }
///
/// let req = CreateUserRequest { email: "bad".into(), password: "short".into() };
/// let result = validate_request(&req);
/// assert!(result.is_err());
/// ```
///
/// # Errors
///
/// Returns `AppError` with validation error details.
pub fn validate_request<T: Validate>(request: &T) -> Result<(), AppError> {
    request.validate().map_err(|errors| {
        let details = format_validation_errors(&errors);
        AppError::validation_error("Validation failed").with_details(details)
    })
}

/// Format `validator::ValidationErrors` into a human-readable string.
///
/// Produces output like:
/// ```text
/// email: must be a valid email; password: length must be between 8 and 128
/// ```
fn format_validation_errors(errors: &validator::ValidationErrors) -> String {
    let mut messages = Vec::new();

    for (field, field_errors) in errors.field_errors() {
        let field_messages: Vec<String> = field_errors
            .iter()
            .map(|e| {
                e.message
                    .as_ref()
                    .map(|m| m.to_string())
                    .unwrap_or_else(|| {
                        format!("failed validation: {}", e.code)
                    })
            })
            .collect();
        messages.push(format!("{}: {}", field, field_messages.join(", ")));
    }

    messages.join("; ")
}

#[cfg(test)]
mod tests {
    use super::*;
    use validator::Validate;

    #[derive(Validate)]
    struct TestRequest {
        #[validate(length(min = 1, message = "name is required"))]
        name: String,
        #[validate(email(message = "must be a valid email"))]
        email: String,
    }

    #[test]
    fn test_validate_request_success() {
        let req = TestRequest {
            name: "Alice".to_string(),
            email: "alice@example.com".to_string(),
        };
        assert!(validate_request(&req).is_ok());
    }

    #[test]
    fn test_validate_request_failure() {
        let req = TestRequest {
            name: String::new(),
            email: "not-an-email".to_string(),
        };
        let err = validate_request(&req).unwrap_err();
        assert_eq!(err.code, "VALIDATION_ERROR");
        assert!(err.details.is_some());
    }
}
