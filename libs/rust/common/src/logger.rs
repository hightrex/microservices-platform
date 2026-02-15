//! Structured logging setup module.
//!
//! Initializes `tracing_subscriber` with JSON or pretty formatting
//! depending on the environment. Mirrors the Go `libs/go/pkg/logger`
//! pattern: JSON in production, human-readable in development.

use tracing_subscriber::{fmt, layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

/// Logger configuration.
#[derive(Debug, Clone, serde::Deserialize)]
pub struct LoggerConfig {
    /// Log level: trace, debug, info, warn, error (default: info).
    #[serde(default = "default_level")]
    pub level: String,
    /// Environment: "local", "dev", "staging", "production".
    #[serde(default = "default_environment")]
    pub environment: String,
}

fn default_level() -> String {
    "info".to_string()
}

fn default_environment() -> String {
    "local".to_string()
}

impl Default for LoggerConfig {
    fn default() -> Self {
        Self {
            level: default_level(),
            environment: default_environment(),
        }
    }
}

/// Initialize the global tracing subscriber.
///
/// - In `local` / `dev` environments: uses human-readable, colorized output.
/// - In all other environments: uses JSON-structured output.
///
/// The log level can be controlled via the `RUST_LOG` environment variable
/// (which takes precedence) or via the `level` config field.
///
/// # Panics
///
/// Panics if the subscriber cannot be set (e.g., called twice).
/// This should only be called once during service startup in `main()`.
pub fn init(config: &LoggerConfig) {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new(&config.level));

    let is_dev = matches!(config.environment.as_str(), "local" | "dev");

    if is_dev {
        tracing_subscriber::registry()
            .with(env_filter)
            .with(
                fmt::layer()
                    .with_target(true)
                    .with_thread_ids(false)
                    .with_file(true)
                    .with_line_number(true),
            )
            .init();
    } else {
        tracing_subscriber::registry()
            .with(env_filter)
            .with(fmt::layer().json().with_target(true))
            .init();
    }

    tracing::info!(
        environment = %config.environment,
        level = %config.level,
        "Logger initialized"
    );
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let cfg = LoggerConfig::default();
        assert_eq!(cfg.level, "info");
        assert_eq!(cfg.environment, "local");
    }
}
