//! File upload security — validation, path traversal prevention, type checking.

use platform_common::error::AppError;

/// Validate a filename for path traversal attacks.
///
/// Rejects filenames containing `..`, `/`, `\`, null bytes, or
/// other potentially dangerous sequences.
pub fn validate_filename(filename: &str) -> Result<String, AppError> {
    if filename.is_empty() {
        return Err(AppError::bad_request("Filename cannot be empty"));
    }

    if filename.len() > 255 {
        return Err(AppError::bad_request("Filename too long (max 255 chars)"));
    }

    // Reject path traversal attempts
    if filename.contains("..") || filename.contains('/') || filename.contains('\\') {
        return Err(AppError::bad_request(
            "Filename contains invalid characters (path traversal attempt)",
        ));
    }

    // Reject null bytes
    if filename.contains('\0') {
        return Err(AppError::bad_request("Filename contains null bytes"));
    }

    // Reject control characters
    if filename.chars().any(|c| c.is_control()) {
        return Err(AppError::bad_request(
            "Filename contains control characters",
        ));
    }

    Ok(filename.to_string())
}

/// Validate content type against an allowlist.
pub fn validate_content_type(
    content_type: &str,
    allowed_types: &[String],
) -> Result<(), AppError> {
    if allowed_types.is_empty() {
        // If no allowlist configured, allow all (but log a warning)
        tracing::warn!("No content type allowlist configured, allowing all types");
        return Ok(());
    }

    if allowed_types.iter().any(|t| t == content_type) {
        Ok(())
    } else {
        Err(AppError::bad_request(format!(
            "Content type '{}' is not allowed",
            content_type
        )))
    }
}

/// Generate a safe S3 key for a file.
///
/// Format: `{tenant_id}/{uuid}/{filename}`
/// This prevents key collisions and provides tenant isolation in storage.
pub fn generate_s3_key(tenant_id: &uuid::Uuid, file_id: &uuid::Uuid, filename: &str) -> String {
    format!("{}/{}/{}", tenant_id, file_id, filename)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_valid_filename() {
        assert!(validate_filename("document.pdf").is_ok());
        assert!(validate_filename("my-file_v2.txt").is_ok());
        assert!(validate_filename("photo (1).jpg").is_ok());
    }

    #[test]
    fn test_path_traversal_filenames() {
        assert!(validate_filename("../../../etc/passwd").is_err());
        assert!(validate_filename("..\\windows\\system32").is_err());
        assert!(validate_filename("foo/../bar.txt").is_err());
        assert!(validate_filename("/etc/shadow").is_err());
    }

    #[test]
    fn test_null_byte_filename() {
        assert!(validate_filename("file\0.txt").is_err());
    }

    #[test]
    fn test_empty_filename() {
        assert!(validate_filename("").is_err());
    }

    #[test]
    fn test_long_filename() {
        let long_name = "a".repeat(256);
        assert!(validate_filename(&long_name).is_err());
    }

    #[test]
    fn test_validate_content_type() {
        let allowed = vec![
            "image/jpeg".to_string(),
            "image/png".to_string(),
            "application/pdf".to_string(),
        ];
        assert!(validate_content_type("image/jpeg", &allowed).is_ok());
        assert!(validate_content_type("application/pdf", &allowed).is_ok());
        assert!(validate_content_type("application/exe", &allowed).is_err());
    }

    #[test]
    fn test_generate_s3_key() {
        let tenant_id = uuid::Uuid::new_v4();
        let file_id = uuid::Uuid::new_v4();
        let key = generate_s3_key(&tenant_id, &file_id, "doc.pdf");
        assert!(key.starts_with(&tenant_id.to_string()));
        assert!(key.ends_with("doc.pdf"));
        assert!(key.contains(&file_id.to_string()));
    }
}
