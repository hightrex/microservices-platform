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
use platform_middleware::auth::{self, AuthConfig, UserContext};
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

/// RBAC layer: require authenticated user with write roles for upload/delete.
async fn require_file_write_roles(request: axum::extract::Request, next: axum::middleware::Next) -> axum::response::Response {
    auth::require_roles(
        vec![
            "admin".to_string(),
            "super_admin".to_string(),
            "org_owner".to_string(),
            "org_admin".to_string(),
            "member".to_string(),
        ],
        request,
        next,
    )
    .await
}

pub fn create_router(state: AppState, cfg: &crate::config::FileServiceConfig) -> Router {
    let auth_cfg = state.auth_config.clone();

    let public_routes = Router::new()
        .route("/health", get(health_handler))
        .route("/metrics", get(metrics_handler));

    // Read-only routes (any authenticated user)
    let read_routes = Router::new()
        .route("/api/v1/files", get(file_handler::list_files))
        .route(
            "/api/v1/files/{id}",
            get(file_handler::get_file),
        )
        .route("/api/v1/files/quota", get(quota_handler::get_quota));

    // Write routes (require file write roles)
    let write_routes = Router::new()
        .route("/api/v1/files/upload", post(file_handler::upload_file))
        .route(
            "/api/v1/files/{id}",
            axum::routing::delete(file_handler::delete_file),
        )
        .layer(middleware::from_fn(require_file_write_roles));

    let protected_routes = Router::new()
        .merge(read_routes)
        .merge(write_routes)
        .layer(middleware::from_fn_with_state(
            auth_cfg,
            auth::auth_middleware,
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
