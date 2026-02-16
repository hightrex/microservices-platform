//! API route registration with middleware stack.

use crate::config::{InvoiceConfig, StripeConfig};
use crate::handlers::{
    invoice_handler, plan_handler, subscription_handler, usage_handler, webhook_handler,
};
use axum::{
    extract::Request,
    middleware,
    middleware::Next,
    response::Response,
    routing::{get, post},
    Json, Router,
};
use platform_common::{health, metrics};
use platform_messaging::producer::Producer;
use platform_middleware::auth::{self, AuthConfig, UserContext};
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
    pub auth_config: AuthConfig,
}

/// Extract TenantContext from request extensions (set by auth middleware).
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

/// Extract UserContext from request extensions (set by auth middleware).
impl axum::extract::FromRequestParts<AppState> for UserContext {
    type Rejection = platform_common::error::AppError;

    async fn from_request_parts(
        parts: &mut http::request::Parts,
        _state: &AppState,
    ) -> Result<Self, Self::Rejection> {
        parts
            .extensions
            .get::<UserContext>()
            .cloned()
            .ok_or_else(|| {
                platform_common::error::AppError::unauthorized("Authentication required")
            })
    }
}

/// RBAC layer: require org_admin or admin roles for billing write operations.
async fn require_billing_write_roles(request: Request, next: Next) -> Response {
    auth::require_roles(
        vec![
            "admin".to_string(),
            "super_admin".to_string(),
            "org_owner".to_string(),
            "org_admin".to_string(),
        ],
        request,
        next,
    )
    .await
}

/// Create the Axum router with all routes and middleware.
pub fn create_router(state: AppState, cfg: &crate::config::BillingConfig) -> Router {
    let auth_cfg = state.auth_config.clone();

    // Public routes (no auth required)
    let public_routes = Router::new()
        .route("/health", get(health_handler))
        .route("/metrics", get(metrics_handler))
        .route(
            "/api/v1/billing/webhook",
            post(webhook_handler::handle_webhook),
        )
        .route("/api/v1/billing/plans", get(plan_handler::list_plans));

    // Read-only protected routes (any authenticated user)
    let read_routes = Router::new()
        .route(
            "/api/v1/billing/subscription",
            get(subscription_handler::get_subscription),
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
        );

    // Write protected routes (require org_admin/admin roles)
    let write_routes = Router::new()
        .route(
            "/api/v1/billing/subscription",
            post(subscription_handler::create_subscription)
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
            "/api/v1/billing/usage/record",
            post(usage_handler::record_usage),
        )
        .layer(middleware::from_fn(require_billing_write_roles));

    // Combine protected routes with auth middleware
    let protected_routes = Router::new()
        .merge(read_routes)
        .merge(write_routes)
        .layer(middleware::from_fn_with_state(
            auth_cfg,
            auth::auth_middleware,
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
