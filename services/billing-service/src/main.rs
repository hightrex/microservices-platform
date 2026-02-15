//! Billing Service — Subscription management with Stripe integration.
//!
//! Provides subscription lifecycle management, usage metering, invoicing,
//! and proration for the platform's multi-tenant billing system.

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
    // Load configuration
    let cfg: config::BillingConfig =
        platform_common::config::load_config("config/config.toml")
            .map_err(|e| anyhow::anyhow!("Failed to load config: {e}"))?;

    // Initialize logging
    logger::init(&cfg.logger);

    tracing::info!("Starting Billing Service");

    // Connect to database
    let db_pool = pool::connect(&cfg.database).await?;

    // Run migrations
    let migrator = sqlx::migrate!("./migrations");
    platform_database::migrations::run_migrations(&db_pool, migrator).await?;

    // Connect to Redis for event publishing
    let producer = Producer::new(&cfg.redis.connection_url(), "billing-service")
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

    // Set up metrics
    let metrics_registry = metrics::MetricsRegistry::new("billing");

    // Build application state
    let app_state = routes::AppState {
        db_pool: db_pool.clone(),
        producer: Arc::new(producer),
        health_manager: health_manager.clone(),
        metrics_registry: metrics_registry.clone(),
        stripe_config: cfg.stripe.clone(),
        invoice_config: cfg.invoice.clone(),
    };

    // Build router
    let app = routes::create_router(app_state, &cfg);

    // Start server
    let bind_addr = cfg.server.bind_address();
    tracing::info!(address = %bind_addr, "Billing Service listening");

    let listener = tokio::net::TcpListener::bind(&bind_addr).await?;

    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await?;

    tracing::info!("Billing Service shut down");
    Ok(())
}

/// Listens for SIGINT/SIGTERM for graceful shutdown.
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
