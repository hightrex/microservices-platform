//! Platform Common Library
//!
//! Shared utilities for all Rust microservices in the platform.
//! Provides configuration, error handling, logging, tenant context,
//! validation, health checks, tracing, and metrics.

pub mod config;
pub mod error;
pub mod health;
pub mod logger;
pub mod metrics;
pub mod tenant;
pub mod tracing_setup;
pub mod validation;
