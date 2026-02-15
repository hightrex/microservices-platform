//! Stripe webhook handling service.
//!
//! Verifies webhook signatures and processes events idempotently.
//! CRITICAL: Verify webhook signature before processing any event.

use platform_common::error::AppError;

/// Verify a Stripe webhook signature using HMAC-SHA256.
///
/// Stripe sends a `Stripe-Signature` header with format:
/// `t=<timestamp>,v1=<signature>`
///
/// We verify by computing HMAC-SHA256 of `{timestamp}.{payload}` using
/// the webhook secret, then comparing with the provided signature.
///
/// # Errors
///
/// Returns `AppError::unauthorized` if the signature is invalid.
pub fn verify_webhook_signature(
    payload: &[u8],
    signature_header: &str,
    webhook_secret: &str,
) -> Result<(), AppError> {
    use hmac::{Hmac, Mac};
    use sha2::Sha256;

    // Parse the Stripe-Signature header
    let mut timestamp = None;
    let mut signatures = Vec::new();

    for part in signature_header.split(',') {
        let kv: Vec<&str> = part.splitn(2, '=').collect();
        if kv.len() != 2 {
            continue;
        }
        match kv[0].trim() {
            "t" => timestamp = Some(kv[1].trim()),
            "v1" => signatures.push(kv[1].trim().to_string()),
            _ => {}
        }
    }

    let timestamp = timestamp
        .ok_or_else(|| AppError::unauthorized("Missing timestamp in webhook signature"))?;

    if signatures.is_empty() {
        return Err(AppError::unauthorized("Missing v1 signature in webhook"));
    }

    // Build the signed payload: "{timestamp}.{payload}"
    let signed_payload = format!(
        "{}.{}",
        timestamp,
        std::str::from_utf8(payload).unwrap_or("")
    );

    // Compute expected signature
    let mut mac = Hmac::<Sha256>::new_from_slice(webhook_secret.as_bytes())
        .map_err(|_| AppError::internal("Failed to create HMAC"))?;
    mac.update(signed_payload.as_bytes());
    let expected = hex::encode(mac.finalize().into_bytes());

    // Check if any v1 signature matches (constant-time comparison)
    let valid = signatures.iter().any(|sig| {
        // Use constant-time comparison to prevent timing attacks
        sig.len() == expected.len()
            && sig
                .bytes()
                .zip(expected.bytes())
                .fold(0u8, |acc, (a, b)| acc | (a ^ b))
                == 0
    });

    if !valid {
        return Err(AppError::unauthorized("Invalid webhook signature"));
    }

    // Optionally: check timestamp is within tolerance (5 min) to prevent replay attacks
    if let Ok(ts) = timestamp.parse::<i64>() {
        let now = chrono::Utc::now().timestamp();
        if (now - ts).abs() > 300 {
            return Err(AppError::unauthorized(
                "Webhook timestamp is too old (possible replay attack)",
            ));
        }
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use hmac::{Hmac, Mac};
    use sha2::Sha256;

    fn create_test_signature(payload: &str, secret: &str) -> String {
        let timestamp = chrono::Utc::now().timestamp();
        let signed_payload = format!("{timestamp}.{payload}");

        let mut mac = Hmac::<Sha256>::new_from_slice(secret.as_bytes()).unwrap();
        mac.update(signed_payload.as_bytes());
        let signature = hex::encode(mac.finalize().into_bytes());

        format!("t={timestamp},v1={signature}")
    }

    #[test]
    fn test_valid_webhook_signature() {
        let payload = r#"{"type":"invoice.paid"}"#;
        let secret = "whsec_test_secret";
        let sig_header = create_test_signature(payload, secret);

        let result = verify_webhook_signature(payload.as_bytes(), &sig_header, secret);
        assert!(result.is_ok());
    }

    #[test]
    fn test_invalid_webhook_signature() {
        let payload = r#"{"type":"invoice.paid"}"#;
        let secret = "whsec_test_secret";
        let sig_header = create_test_signature(payload, secret);

        // Use wrong secret
        let result = verify_webhook_signature(payload.as_bytes(), &sig_header, "wrong_secret");
        assert!(result.is_err());
    }

    #[test]
    fn test_tampered_payload() {
        let payload = r#"{"type":"invoice.paid"}"#;
        let secret = "whsec_test_secret";
        let sig_header = create_test_signature(payload, secret);

        // Tamper with payload
        let tampered = r#"{"type":"invoice.paid","amount":9999999}"#;
        let result = verify_webhook_signature(tampered.as_bytes(), &sig_header, secret);
        assert!(result.is_err());
    }

    #[test]
    fn test_missing_timestamp() {
        let result = verify_webhook_signature(b"payload", "v1=abc123", "secret");
        assert!(result.is_err());
    }

    #[test]
    fn test_missing_signature() {
        let result = verify_webhook_signature(b"payload", "t=12345", "secret");
        assert!(result.is_err());
    }
}
