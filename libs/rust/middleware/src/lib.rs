//! Platform Middleware Library
//!
//! Axum middleware for authentication, tenant extraction, logging,
//! metrics, CORS, and rate limiting. Mirrors the Go middleware in
//! `libs/go/pkg/middleware`.

pub mod auth;
pub mod cors;
pub mod logging;
pub mod metrics_middleware;
pub mod rate_limit;
pub mod tenant;
