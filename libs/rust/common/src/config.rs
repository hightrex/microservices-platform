//! Configuration loading module.
//!
//! Loads configuration from TOML files and environment variables,
//! mirroring the Go `libs/go/pkg/config` Viper-based pattern.
//! Environment variables override file values using `APP_` prefix
//! with `__` (double underscore) as the nested separator.

use figment::{
    providers::{Env, Format, Toml},
    Figment,
};
use serde::Deserialize;

/// Database configuration matching Go `database.Config`.
#[derive(Debug, Clone, Deserialize)]
pub struct DatabaseConfig {
    pub host: String,
    pub port: u16,
    pub user: String,
    pub password: String,
    #[serde(alias = "name")]
    pub database: String,
    #[serde(default = "default_ssl_mode")]
    pub ssl_mode: String,
    #[serde(default = "default_pool_size")]
    pub max_connections: u32,
    #[serde(default = "default_min_connections")]
    pub min_connections: u32,
}

fn default_ssl_mode() -> String {
    "disable".to_string()
}

fn default_pool_size() -> u32 {
    10
}

fn default_min_connections() -> u32 {
    2
}

impl DatabaseConfig {
    /// Build a PostgreSQL connection URL from this configuration.
    pub fn connection_url(&self) -> String {
        format!(
            "postgres://{}:{}@{}:{}/{}?sslmode={}",
            self.user, self.password, self.host, self.port, self.database, self.ssl_mode
        )
    }
}

/// Redis configuration matching Go `cache.Config`.
#[derive(Debug, Clone, Deserialize)]
pub struct RedisConfig {
    #[serde(default = "default_redis_host")]
    pub host: String,
    #[serde(default = "default_redis_port")]
    pub port: u16,
    #[serde(default)]
    pub password: String,
    #[serde(default)]
    pub db: i64,
}

fn default_redis_host() -> String {
    "localhost".to_string()
}

fn default_redis_port() -> u16 {
    6379
}

impl RedisConfig {
    /// Build a Redis connection URL from this configuration.
    pub fn connection_url(&self) -> String {
        if self.password.is_empty() {
            format!("redis://{}:{}/{}", self.host, self.port, self.db)
        } else {
            format!(
                "redis://:{}@{}:{}/{}",
                self.password, self.host, self.port, self.db
            )
        }
    }
}

/// Server configuration for HTTP services.
#[derive(Debug, Clone, Deserialize)]
pub struct ServerConfig {
    #[serde(default = "default_server_host")]
    pub host: String,
    #[serde(default = "default_server_port")]
    pub port: u16,
    #[serde(default)]
    pub cors_origins: Vec<String>,
}

fn default_server_host() -> String {
    "0.0.0.0".to_string()
}

fn default_server_port() -> u16 {
    8080
}

impl ServerConfig {
    /// Build the socket address string for binding.
    pub fn bind_address(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }
}

/// Load configuration from a TOML file and environment variables.
///
/// The config file is loaded first, then environment variables with the
/// `APP_` prefix override any values. Nested keys use `__` (double underscore)
/// as separator. For example, `APP_DATABASE__HOST=localhost` sets `database.host`.
///
/// # Errors
///
/// Returns an error if the configuration cannot be loaded or deserialized.
pub fn load_config<T: for<'de> Deserialize<'de>>(config_path: &str) -> Result<T, figment::Error> {
    Figment::new()
        .merge(Toml::file(config_path))
        .merge(Env::prefixed("APP_").split("__"))
        .extract()
}

/// Load configuration with a custom environment prefix.
///
/// # Errors
///
/// Returns an error if the configuration cannot be loaded or deserialized.
pub fn load_config_with_prefix<T: for<'de> Deserialize<'de>>(
    config_path: &str,
    env_prefix: &str,
) -> Result<T, figment::Error> {
    Figment::new()
        .merge(Toml::file(config_path))
        .merge(Env::prefixed(env_prefix).split("__"))
        .extract()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_database_config_connection_url() {
        let cfg = DatabaseConfig {
            host: "localhost".to_string(),
            port: 5432,
            user: "admin".to_string(),
            password: "secret".to_string(),
            database: "billing".to_string(),
            ssl_mode: "disable".to_string(),
            max_connections: 10,
            min_connections: 2,
        };
        assert_eq!(
            cfg.connection_url(),
            "postgres://admin:secret@localhost:5432/billing?sslmode=disable"
        );
    }

    #[test]
    fn test_redis_config_connection_url_no_password() {
        let cfg = RedisConfig {
            host: "localhost".to_string(),
            port: 6379,
            password: String::new(),
            db: 0,
        };
        assert_eq!(cfg.connection_url(), "redis://localhost:6379/0");
    }

    #[test]
    fn test_redis_config_connection_url_with_password() {
        let cfg = RedisConfig {
            host: "redis.internal".to_string(),
            port: 6380,
            password: "s3cret".to_string(),
            db: 2,
        };
        assert_eq!(
            cfg.connection_url(),
            "redis://:s3cret@redis.internal:6380/2"
        );
    }

    #[test]
    fn test_server_config_bind_address() {
        let cfg = ServerConfig {
            host: "0.0.0.0".to_string(),
            port: 8083,
            cors_origins: vec![],
        };
        assert_eq!(cfg.bind_address(), "0.0.0.0:8083");
    }
}
