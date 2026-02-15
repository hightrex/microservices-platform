//! API route registration.

use crate::config::{QuotaConfig, S3Config, UploadConfig};
use crate::handlers::{file_handler, quota_handler};
use axum::{
    middleware,
    routing::{get, post},
    Json, Router,
};
use platform_common::{health, metrics};
use platform_messaging::producer::Producer;
use sqlx::PgPool;
use std::sync::Arc;

#[derive(Clone)]
pub struct AppState {
    pub db_pool: PgPool,
    pub producer: Arc<Producer>,
    pub health_manager: Arc<health::HealthManager>,
    pub metrics_registry: metrics::MetricsRegistry,
    pub s3_config: S3Config,
    pub quota_config: QuotaConfig,
    pub upload_config: UploadConfig,
}

/// Extract TenantContext from request extensions.
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

pub fn create_router(state: AppState, cfg: &crate::config::FileServiceConfig) -> Router {
    let public_routes = Router::new()
        .route("/health", get(health_handler))
        .route("/metrics", get(metrics_handler));

    let protected_routes = Router::new()
        .route("/api/v1/files/upload", post(file_handler::upload_file))
        .route("/api/v1/files", get(file_handler::list_files))
        .route(
            "/api/v1/files/{id}",
            get(file_handler::get_file).delete(file_handler::delete_file),
        )
        .route("/api/v1/files/quota", get(quota_handler::get_quota))
        .layer(middleware::from_fn(
            platform_middleware::tenant::require_tenant_middleware,
        ));

    let cors_layer = platform_middleware::cors::create_cors_layer(&cfg.server.cors_origins);

    Router::new()
        .merge(public_routes)
        .merge(protected_routes)
        .layer(middleware::from_fn(
            platform_middleware::logging::request_logger,
        ))
        .layer(cors_layer)
        .layer(tower_http::limit::RequestBodyLimitLayer::new(
            cfg.quota.max_file_size_bytes as usize,
        ))
        .with_state(state)
}

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
