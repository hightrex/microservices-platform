//! Stripe webhook HTTP handler.

use crate::models::subscription::SubscriptionStatus;
use crate::repository::{plan_repo, subscription_repo};
use crate::routes::AppState;
use crate::service::webhook_service;
use axum::{
    body::Bytes,
    extract::State,
    http::{HeaderMap, StatusCode},
    response::IntoResponse,
    Json,
};
use platform_messaging::schema::{event_types, streams};
use uuid::Uuid;

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

    // Extract tenant_id from the Stripe event metadata (set during subscription creation).
    // Reject events without a valid tenant_id to prevent multi-tenancy isolation violations.
    let tenant_id = match event
        .pointer("/data/object/metadata/tenant_id")
        .and_then(|v| v.as_str())
        .and_then(|s| Uuid::parse_str(s).ok())
    {
        Some(id) if !id.is_nil() => id,
        _ => {
            tracing::warn!(
                event_type = %event_type,
                "Webhook event missing valid tenant_id in metadata, rejecting"
            );
            return (
                StatusCode::OK,
                Json(serde_json::json!({
                    "success": true,
                    "message": "Webhook acknowledged but skipped — missing tenant_id"
                })),
            );
        }
    };

    // Extract invoice/subscription ID from the Stripe payload
    let object_id = event
        .pointer("/data/object/id")
        .and_then(|v| v.as_str())
        .unwrap_or("unknown");

    // Extract user_id from Stripe metadata if available (set during subscription creation).
    // Webhook events may not have a user_id — the notification consumer handles this gracefully.
    let user_id = event
        .pointer("/data/object/metadata/user_id")
        .and_then(|v| v.as_str())
        .unwrap_or("");

    match event_type {
        "invoice.payment_succeeded" => {
            let amount = event
                .pointer("/data/object/amount_paid")
                .and_then(|v| v.as_i64())
                .unwrap_or(0);
            let currency = event
                .pointer("/data/object/currency")
                .and_then(|v| v.as_str())
                .unwrap_or("usd");

            tracing::info!(
                invoice_id = %object_id,
                tenant_id = %tenant_id,
                amount = amount,
                "Invoice payment succeeded"
            );

            if let Err(e) = state
                .producer
                .publish(
                    streams::BILLING_EVENTS,
                    event_types::INVOICE_PAID,
                    tenant_id,
                    serde_json::json!({
                        "invoice_id": object_id,
                        "user_id": user_id,
                        "amount": amount,
                        "currency": currency,
                    }),
                )
                .await
            {
                tracing::error!(error = %e, invoice_id = %object_id, "Failed to publish invoice.paid event");
            }
        }
        "invoice.payment_failed" => {
            let reason = event
                .pointer("/data/object/last_finalization_error/message")
                .and_then(|v| v.as_str())
                .unwrap_or("Payment declined");

            tracing::warn!(
                invoice_id = %object_id,
                tenant_id = %tenant_id,
                reason = %reason,
                "Invoice payment failed"
            );

            if let Err(e) = state
                .producer
                .publish(
                    streams::BILLING_EVENTS,
                    event_types::INVOICE_FAILED,
                    tenant_id,
                    serde_json::json!({
                        "invoice_id": object_id,
                        "user_id": user_id,
                        "reason": reason,
                    }),
                )
                .await
            {
                tracing::error!(error = %e, invoice_id = %object_id, "Failed to publish invoice.failed event");
            }
        }
        "customer.subscription.updated" => {
            tracing::info!(
                subscription_id = %object_id,
                tenant_id = %tenant_id,
                "Subscription updated via Stripe"
            );

            // Sync subscription status to database
            let stripe_status = event
                .pointer("/data/object/status")
                .and_then(|v| v.as_str())
                .unwrap_or("active");
            if let Some(sub) = subscription_repo::get_by_stripe_id(&state.db_pool, object_id, tenant_id)
                .await
                .unwrap_or(None)
            {
                let new_status = match stripe_status {
                    "active" => Some(SubscriptionStatus::Active),
                    "past_due" => Some(SubscriptionStatus::PastDue),
                    "trialing" => Some(SubscriptionStatus::Trialing),
                    "paused" => Some(SubscriptionStatus::Paused),
                    "canceled" => Some(SubscriptionStatus::Canceled),
                    other => {
                        tracing::warn!(
                            stripe_status = %other,
                            subscription_id = %object_id,
                            "Unrecognized Stripe subscription status, skipping DB update"
                        );
                        None
                    }
                };
                if let Some(status) = new_status {
                    if let Err(e) = subscription_repo::update_status(&state.db_pool, sub.id, tenant_id, status).await {
                        tracing::error!(error = %e, subscription_id = %object_id, "Failed to update subscription status from webhook");
                    }
                }
            }

            // Look up the plan name so the notification consumer can produce
            // a meaningful message (avoids "changed to the '' plan" text).
            let plan_name = if let Some(ref sub) = subscription_repo::get_by_stripe_id(&state.db_pool, object_id, tenant_id)
                .await
                .unwrap_or(None)
            {
                plan_repo::get_by_id(&state.db_pool, sub.plan_id)
                    .await
                    .ok()
                    .flatten()
                    .map(|p| p.name)
                    .unwrap_or_default()
            } else {
                String::new()
            };

            if let Err(e) = state
                .producer
                .publish(
                    streams::BILLING_EVENTS,
                    event_types::SUBSCRIPTION_UPDATED,
                    tenant_id,
                    serde_json::json!({
                        "subscription_id": object_id,
                        "user_id": user_id,
                        "plan_name": plan_name,
                        "source": "stripe_webhook",
                    }),
                )
                .await
            {
                tracing::error!(error = %e, subscription_id = %object_id, "Failed to publish subscription.updated event");
            }
        }
        "customer.subscription.deleted" => {
            tracing::info!(
                subscription_id = %object_id,
                tenant_id = %tenant_id,
                "Subscription deleted via Stripe"
            );

            // Sync subscription cancellation to database
            if let Some(sub) = subscription_repo::get_by_stripe_id(&state.db_pool, object_id, tenant_id)
                .await
                .unwrap_or(None)
            {
                if let Err(e) = subscription_repo::cancel(&state.db_pool, sub.id, tenant_id, true).await {
                    tracing::error!(error = %e, subscription_id = %object_id, "Failed to cancel subscription from webhook");
                }
            }

            if let Err(e) = state
                .producer
                .publish(
                    streams::BILLING_EVENTS,
                    event_types::SUBSCRIPTION_CANCELED,
                    tenant_id,
                    serde_json::json!({
                        "subscription_id": object_id,
                        "user_id": user_id,
                        "source": "stripe_webhook",
                        "immediate": true,
                    }),
                )
                .await
            {
                tracing::error!(error = %e, subscription_id = %object_id, "Failed to publish subscription.canceled event");
            }
        }
        "payment_intent.succeeded" => {
            tracing::info!(
                payment_intent_id = %object_id,
                tenant_id = %tenant_id,
                "Payment intent succeeded"
            );
        }
        "payment_intent.payment_failed" => {
            tracing::warn!(
                payment_intent_id = %object_id,
                tenant_id = %tenant_id,
                "Payment intent failed"
            );
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
