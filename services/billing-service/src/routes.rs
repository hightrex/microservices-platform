//! API route registration with middleware stack.

use crate::config::{InvoiceConfig, StripeConfig};
use crate::handlers::{
    invoice_handler, plan_handler, subscription_handler, usage_handler, webhook_handler,
};
use axum::{
    middleware,
    routing::{get, post},
    Json, Router,
};
use platform_common::{health, metrics};
use platform_messaging::producer::Producer;
use sqlx::PgPool;
use std::sync::Arc;

/// Shared application state.
#[derive(Clone)]
pub struct AppState {
    pub db_pool: PgPool,
    pub producer: Arc<Producer>,
    pub health_manager: Arc<health::HealthManager>,
    pub metrics_registry: metrics::MetricsRegistry,
    pub stripe_config: StripeConfig,
    pub invoice_config: InvoiceConfig,
}

/// Extract TenantContext from request extensions (set by tenant middleware).
impl axum::extract::FromRequestParts<AppState> for platform_common::tenant::TenantContext {
    type Rejection = platform_common::error::AppError;

    async fn from_request_parts(
        parts: &mut http::request::Parts,
        _state: &AppState,
    ) -> Result<Self, Self::Rejection> {
        parts
            .extensions
            .get::<platform_common::tenant::TenantContext>()
            .cloned()
            .ok_or_else(|| {
                platform_common::error::AppError::unauthorized("Tenant context is required")
            })
    }
}

/// Create the Axum router with all routes and middleware.
pub fn create_router(state: AppState, cfg: &crate::config::BillingConfig) -> Router {
    // Public routes (no auth required)
    let public_routes = Router::new()
        .route("/health", get(health_handler))
        .route("/metrics", get(metrics_handler))
        .route(
            "/api/v1/billing/webhook",
            post(webhook_handler::handle_webhook),
        )
        .route("/api/v1/billing/plans", get(plan_handler::list_plans));

    // Protected routes (require tenant context)
    let protected_routes = Router::new()
        .route(
            "/api/v1/billing/subscription",
            get(subscription_handler::get_subscription)
                .post(subscription_handler::create_subscription)
                .put(subscription_handler::update_subscription)
                .delete(subscription_handler::cancel_subscription),
        )
        .route(
            "/api/v1/billing/subscription/pause",
            post(subscription_handler::pause_subscription),
        )
        .route(
            "/api/v1/billing/subscription/resume",
            post(subscription_handler::resume_subscription),
        )
        .route(
            "/api/v1/billing/invoices",
            get(invoice_handler::list_invoices),
        )
        .route(
            "/api/v1/billing/invoices/{id}",
            get(invoice_handler::get_invoice),
        )
        .route("/api/v1/billing/usage", get(usage_handler::get_usage))
        .route(
            "/api/v1/billing/usage/quota",
            get(usage_handler::get_quota_status),
        )
        .route(
            "/api/v1/billing/usage/record",
            post(usage_handler::record_usage),
        )
        .layer(middleware::from_fn(
            platform_middleware::tenant::require_tenant_middleware,
        ));

    // Apply middleware stack
    let cors_layer =
        platform_middleware::cors::create_cors_layer(&cfg.server.cors_origins);

    Router::new()
        .merge(public_routes)
        .merge(protected_routes)
        .layer(middleware::from_fn(
            platform_middleware::logging::request_logger,
        ))
        .layer(cors_layer)
        .layer(tower_http::limit::RequestBodyLimitLayer::new(
            1024 * 1024, // 1MB max body
        ))
        .with_state(state)
}

/// Health check handler.
async fn health_handler(
    axum::extract::State(state): axum::extract::State<AppState>,
) -> impl axum::response::IntoResponse {
    let response = state.health_manager.check_health().await;
    let status = if response.status == "UP" {
        http::StatusCode::OK
    } else {
        http::StatusCode::SERVICE_UNAVAILABLE
    };
    (status, Json(response))
}

/// Metrics handler.
async fn metrics_handler(
    axum::extract::State(state): axum::extract::State<AppState>,
) -> impl axum::response::IntoResponse {
    let output = state.metrics_registry.render();
    (
        [(
            http::header::CONTENT_TYPE,
            "text/plain; version=0.0.4; charset=utf-8",
        )],
        output,
    )
}
