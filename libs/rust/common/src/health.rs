//! Health check framework.
//!
//! Mirrors the Go `libs/go/pkg/health` package. Provides a `HealthCheck`
//! trait, a `HealthManager` for registering checks, and an Axum handler
//! for the `/health` endpoint.

use axum::{http::StatusCode, response::IntoResponse, Json};
use serde::Serialize;
use std::sync::Arc;
use tokio::sync::RwLock;

/// A simple health check implementation using a closure.
pub struct SimpleCheck {
    check_name: String,
    checker: Box<dyn Fn() -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), String>> + Send>> + Send + Sync>,
}

impl SimpleCheck {
    /// Create a new simple health check from a name and an async closure.
    pub fn new<F, Fut>(name: impl Into<String>, checker: F) -> Self
    where
        F: Fn() -> Fut + Send + Sync + 'static,
        Fut: std::future::Future<Output = Result<(), String>> + Send + 'static,
    {
        let check_name = name.into();
        Self {
            check_name,
            checker: Box::new(move || Box::pin(checker())),
        }
    }
}

/// Health check manager that aggregates multiple dependency checks.
pub struct HealthManager {
    checks: RwLock<Vec<(String, Arc<dyn Fn() -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), String>> + Send>> + Send + Sync>)>>,
}

impl HealthManager {
    /// Create a new empty health manager.
    pub fn new() -> Self {
        Self {
            checks: RwLock::new(Vec::new()),
        }
    }

    /// Register a health check.
    pub async fn add_check(&self, check: SimpleCheck) {
        let mut checks = self.checks.write().await;
        checks.push((check.check_name, Arc::from(check.checker)));
    }

    /// Run all health checks and return the aggregate result.
    pub async fn check_health(&self) -> HealthResponse {
        let checks = self.checks.read().await;
        let mut results = std::collections::HashMap::new();
        let mut all_healthy = true;

        for (name, checker) in checks.iter() {
            match checker().await {
                Ok(()) => {
                    results.insert(name.clone(), "UP".to_string());
                }
                Err(msg) => {
                    all_healthy = false;
                    results.insert(name.clone(), msg);
                }
            }
        }

        HealthResponse {
            status: if all_healthy {
                "UP".to_string()
            } else {
                "DOWN".to_string()
            },
            checks: results,
        }
    }

    /// Create an Axum handler for the `/health` endpoint.
    pub async fn handler(self: Arc<Self>) -> impl IntoResponse {
        let response = self.check_health().await;
        let status = if response.status == "UP" {
            StatusCode::OK
        } else {
            StatusCode::SERVICE_UNAVAILABLE
        };
        (status, Json(response))
    }
}

impl Default for HealthManager {
    fn default() -> Self {
        Self::new()
    }
}

/// Health check response body.
#[derive(Debug, Serialize)]
pub struct HealthResponse {
    pub status: String,
    pub checks: std::collections::HashMap<String, String>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_health_manager_all_up() {
        let manager = HealthManager::new();
        manager
            .add_check(SimpleCheck::new("test_dep", || async { Ok(()) }))
            .await;

        let response = manager.check_health().await;
        assert_eq!(response.status, "UP");
        assert_eq!(response.checks.get("test_dep").unwrap(), "UP");
    }

    #[tokio::test]
    async fn test_health_manager_one_down() {
        let manager = HealthManager::new();
        manager
            .add_check(SimpleCheck::new("healthy", || async { Ok(()) }))
            .await;
        manager
            .add_check(SimpleCheck::new("unhealthy", || async {
                Err("connection refused".to_string())
            }))
            .await;

        let response = manager.check_health().await;
        assert_eq!(response.status, "DOWN");
        assert_eq!(response.checks.get("healthy").unwrap(), "UP");
        assert_eq!(
            response.checks.get("unhealthy").unwrap(),
            "connection refused"
        );
    }

    #[tokio::test]
    async fn test_health_manager_empty() {
        let manager = HealthManager::new();
        let response = manager.check_health().await;
        assert_eq!(response.status, "UP");
        assert!(response.checks.is_empty());
    }
}
