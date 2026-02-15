//! Stripe webhook HTTP handler.

use crate::routes::AppState;
use crate::service::webhook_service;
use axum::{
    body::Bytes,
    extract::State,
    http::{HeaderMap, StatusCode},
    response::IntoResponse,
    Json,
};

/// POST /api/v1/billing/webhook — Stripe webhook endpoint (public, no auth).
///
/// Verifies the Stripe signature before processing any event.
pub async fn handle_webhook(
    State(state): State<AppState>,
    headers: HeaderMap,
    body: Bytes,
) -> impl IntoResponse {
    // Extract the Stripe-Signature header
    let signature = match headers.get("Stripe-Signature").and_then(|v| v.to_str().ok()) {
        Some(sig) => sig.to_string(),
        None => {
            tracing::warn!("Webhook request missing Stripe-Signature header");
            return (
                StatusCode::BAD_REQUEST,
                Json(serde_json::json!({
                    "success": false,
                    "error": {
                        "code": "BAD_REQUEST",
                        "message": "Missing Stripe-Signature header"
                    }
                })),
            );
        }
    };

    // Verify signature
    if let Err(e) = webhook_service::verify_webhook_signature(
        &body,
        &signature,
        &state.stripe_config.webhook_secret,
    ) {
        tracing::warn!(error = %e, "Webhook signature verification failed");
        return (
            StatusCode::UNAUTHORIZED,
            Json(serde_json::json!({
                "success": false,
                "error": {
                    "code": "UNAUTHORIZED",
                    "message": "Invalid webhook signature"
                }
            })),
        );
    }

    // Parse the event payload
    let event: serde_json::Value = match serde_json::from_slice(&body) {
        Ok(v) => v,
        Err(e) => {
            tracing::error!(error = %e, "Failed to parse webhook payload");
            return (
                StatusCode::BAD_REQUEST,
                Json(serde_json::json!({
                    "success": false,
                    "error": {
                        "code": "BAD_REQUEST",
                        "message": "Invalid webhook payload"
                    }
                })),
            );
        }
    };

    let event_type = event
        .get("type")
        .and_then(|v| v.as_str())
        .unwrap_or("unknown");

    tracing::info!(event_type = %event_type, "Processing Stripe webhook");

    // Process event based on type
    // TODO(phase-2): Implement full event processing for each type (BILLING-42)
    match event_type {
        "invoice.payment_succeeded" => {
            tracing::info!("Invoice payment succeeded");
        }
        "invoice.payment_failed" => {
            tracing::warn!("Invoice payment failed");
        }
        "customer.subscription.updated" => {
            tracing::info!("Subscription updated via Stripe");
        }
        "customer.subscription.deleted" => {
            tracing::info!("Subscription deleted via Stripe");
        }
        "payment_intent.succeeded" => {
            tracing::info!("Payment intent succeeded");
        }
        "payment_intent.payment_failed" => {
            tracing::warn!("Payment intent failed");
        }
        _ => {
            tracing::debug!(event_type = %event_type, "Unhandled webhook event type");
        }
    }

    (
        StatusCode::OK,
        Json(serde_json::json!({
            "success": true,
            "message": "Webhook processed"
        })),
    )
}
