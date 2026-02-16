//! File Service — S3-compatible file storage with MinIO.
//!
//! Provides file upload/download, quota enforcement, virus scanning hooks,
//! and signed URL generation for the platform.

mod config;
mod handlers;
mod models;
mod repository;
mod routes;
mod service;

use platform_common::{health, logger, metrics};
use platform_database::pool;
use platform_messaging::producer::Producer;
use std::sync::Arc;
use tokio::signal;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let cfg: config::FileServiceConfig =
        platform_common::config::load_config("config/config.toml")
            .map_err(|e| anyhow::anyhow!("Failed to load config: {e}"))?;

    logger::init(&cfg.logger);
    tracing::info!("Starting File Service");

    let db_pool = pool::connect(&cfg.database).await?;

    let migrator = sqlx::migrate!("./migrations");
    platform_database::migrations::run_migrations(&db_pool, migrator).await?;

    let producer = Producer::new(&cfg.redis.connection_url(), "file-service")
        .map_err(|e| anyhow::anyhow!("Failed to create producer: {e}"))?;

    // Set up health checks
    let health_manager = Arc::new(health::HealthManager::new());
    {
        let pool = db_pool.clone();
        health_manager
            .add_check(health::SimpleCheck::new("postgres", move || {
                let pool = pool.clone();
                async move { pool::health_check(&pool).await }
            }))
            .await;
    }
    {
        let producer = producer.clone();
        health_manager
            .add_check(health::SimpleCheck::new("redis", move || {
                let producer = producer.clone();
                async move { producer.health_check().await }
            }))
            .await;
    }

    let metrics_registry = metrics::MetricsRegistry::new("files");

    // Build auth configuration for JWT validation
    let auth_config =
        platform_middleware::auth::AuthConfig::new(&cfg.auth.jwt_secret);

    let app_state = routes::AppState {
        db_pool: db_pool.clone(),
        producer: Arc::new(producer),
        health_manager: health_manager.clone(),
        metrics_registry: metrics_registry.clone(),
        s3_config: cfg.s3.clone(),
        quota_config: cfg.quota.clone(),
        upload_config: cfg.upload.clone(),
        auth_config,
    };

    let app = routes::create_router(app_state, &cfg);

    let bind_addr = cfg.server.bind_address();
    tracing::info!(address = %bind_addr, "File Service listening");

    let listener = tokio::net::TcpListener::bind(&bind_addr).await?;
    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await?;

    tracing::info!("File Service shut down");
    Ok(())
}

async fn shutdown_signal() {
    let ctrl_c = async {
        signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("failed to install SIGTERM handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => tracing::info!("Received SIGINT"),
        _ = terminate => tracing::info!("Received SIGTERM"),
    }
}
