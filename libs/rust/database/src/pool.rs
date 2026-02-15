//! Postgres connection pool management.
//!
//! Uses `sqlx::PgPool` with configuration from `platform_common::config`.
//! Provides health check integration and connection lifecycle management.

use platform_common::config::DatabaseConfig;
use sqlx::postgres::{PgConnectOptions, PgPoolOptions};
use sqlx::PgPool;
use std::str::FromStr;
use std::time::Duration;

/// Create a new Postgres connection pool from configuration.
///
/// Configures connection limits, timeouts, and health check intervals.
/// Pings the database to verify connectivity before returning.
///
/// # Errors
///
/// Returns an error if the connection cannot be established.
pub async fn connect(config: &DatabaseConfig) -> Result<PgPool, sqlx::Error> {
    let connect_options = PgConnectOptions::from_str(&config.connection_url())
        .map_err(|e| sqlx::Error::Configuration(e.into()))?;

    let pool = PgPoolOptions::new()
        .max_connections(config.max_connections)
        .min_connections(config.min_connections)
        .acquire_timeout(Duration::from_secs(30))
        .idle_timeout(Duration::from_secs(600))
        .max_lifetime(Duration::from_secs(3600))
        .test_before_acquire(true)
        .connect_with(connect_options)
        .await?;

    tracing::info!(
        host = %config.host,
        port = %config.port,
        database = %config.database,
        max_connections = %config.max_connections,
        "Connected to database"
    );

    Ok(pool)
}

/// Check if the database connection pool is healthy.
///
/// Executes a simple query to verify connectivity.
///
/// # Errors
///
/// Returns an error string if the health check fails.
pub async fn health_check(pool: &PgPool) -> Result<(), String> {
    sqlx::query("SELECT 1")
        .execute(pool)
        .await
        .map(|_| ())
        .map_err(|e| format!("database health check failed: {e}"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_database_config_connection_url() {
        let config = DatabaseConfig {
            host: "db.example.com".to_string(),
            port: 5432,
            user: "app".to_string(),
            password: "pass".to_string(),
            database: "billing".to_string(),
            ssl_mode: "require".to_string(),
            max_connections: 20,
            min_connections: 5,
        };

        let url = config.connection_url();
        assert!(url.starts_with("postgres://app:pass@db.example.com:5432/billing"));
        assert!(url.contains("sslmode=require"));
    }
}
