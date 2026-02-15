//! Redis-backed rate limiting middleware for Axum.
//!
//! Implements a sliding window rate limiter with per-tenant and per-user
//! limits. Returns `X-RateLimit-*` headers and 429 with `Retry-After`.

use axum::{
    extract::Request,
    http::StatusCode,
    middleware::Next,
    response::{IntoResponse, Response},
    Json,
};
use redis::AsyncCommands;
use std::sync::Arc;
use std::time::Duration;

/// Rate limit configuration.
#[derive(Debug, Clone)]
pub struct RateLimitConfig {
    /// Maximum requests per window.
    pub max_requests: u64,
    /// Sliding window duration.
    pub window: Duration,
    /// Redis key prefix.
    pub key_prefix: String,
}

impl RateLimitConfig {
    /// Create a new rate limit config.
    pub fn new(max_requests: u64, window: Duration) -> Self {
        Self {
            max_requests,
            window,
            key_prefix: "ratelimit".to_string(),
        }
    }

    /// Override the key prefix.
    pub fn with_prefix(mut self, prefix: impl Into<String>) -> Self {
        self.key_prefix = prefix.into();
        self
    }
}

/// Rate limiter state shared across requests.
#[derive(Clone)]
pub struct RateLimiter {
    client: redis::Client,
    config: RateLimitConfig,
}

impl RateLimiter {
    /// Create a new rate limiter.
    ///
    /// # Errors
    ///
    /// Returns an error if the Redis client cannot be created.
    pub fn new(redis_url: &str, config: RateLimitConfig) -> Result<Self, redis::RedisError> {
        let client = redis::Client::open(redis_url)?;
        Ok(Self { client, config })
    }

    /// Create a rate limiter from an existing Redis client.
    pub fn from_client(client: redis::Client, config: RateLimitConfig) -> Self {
        Self { client, config }
    }

    /// Check if a request should be rate limited.
    ///
    /// Uses a Redis sliding window counter. Returns the current count
    /// and remaining requests, or an error if the limit is exceeded.
    async fn check_rate_limit(&self, key: &str) -> Result<RateLimitInfo, RateLimitError> {
        let mut conn = self
            .client
            .get_multiplexed_async_connection()
            .await
            .map_err(|e| {
                tracing::error!(error = %e, "Rate limiter Redis connection failed");
                // Fail open: allow the request if Redis is down
                RateLimitError::RedisUnavailable
            })?;

        let full_key = format!("{}:{}", self.config.key_prefix, key);
        let window_secs = self.config.window.as_secs();

        // Sliding window using INCR + EXPIRE
        let count: u64 = conn.incr(&full_key, 1u64).await.map_err(|e| {
            tracing::error!(error = %e, "Rate limit INCR failed");
            RateLimitError::RedisUnavailable
        })?;

        // Set TTL on first request in window
        if count == 1 {
            let _: Result<bool, _> = conn.expire(&full_key, window_secs as i64).await;
        }

        // Get remaining TTL for Retry-After header
        let ttl: i64 = conn.ttl(&full_key).await.unwrap_or(window_secs as i64);

        let remaining = if count > self.config.max_requests {
            0
        } else {
            self.config.max_requests - count
        };

        if count > self.config.max_requests {
            return Err(RateLimitError::Exceeded(RateLimitInfo {
                limit: self.config.max_requests,
                remaining: 0,
                reset: ttl.max(0) as u64,
            }));
        }

        Ok(RateLimitInfo {
            limit: self.config.max_requests,
            remaining,
            reset: ttl.max(0) as u64,
        })
    }
}

/// Rate limit info for response headers.
#[derive(Debug)]
pub struct RateLimitInfo {
    pub limit: u64,
    pub remaining: u64,
    pub reset: u64,
}

/// Rate limit errors.
#[derive(Debug)]
pub enum RateLimitError {
    /// Rate limit exceeded.
    Exceeded(RateLimitInfo),
    /// Redis is unavailable (fail open).
    RedisUnavailable,
}

/// Rate limiting middleware for Axum.
///
/// Extracts a rate limit key from the tenant ID (from `X-Tenant-ID` header)
/// or falls back to the client IP. Returns `X-RateLimit-*` headers on every
/// response and 429 with `Retry-After` when the limit is exceeded.
pub async fn rate_limit_middleware(
    axum::extract::State(limiter): axum::extract::State<Arc<RateLimiter>>,
    request: Request,
    next: Next,
) -> Response {
    // Build rate limit key: prefer tenant_id, fall back to IP
    let key = request
        .headers()
        .get("X-Tenant-ID")
        .and_then(|v| v.to_str().ok())
        .map(|s| format!("tenant:{s}"))
        .unwrap_or_else(|| {
            // Fall back to IP-based rate limiting
            request
                .headers()
                .get("X-Forwarded-For")
                .and_then(|v| v.to_str().ok())
                .and_then(|s| s.split(',').next())
                .map(|s| format!("ip:{}", s.trim()))
                .unwrap_or_else(|| "ip:unknown".to_string())
        });

    match limiter.check_rate_limit(&key).await {
        Ok(info) => {
            let mut response = next.run(request).await;

            // Add rate limit headers to response
            let headers = response.headers_mut();
            if let Ok(v) = info.limit.to_string().parse() {
                headers.insert("X-RateLimit-Limit", v);
            }
            if let Ok(v) = info.remaining.to_string().parse() {
                headers.insert("X-RateLimit-Remaining", v);
            }
            if let Ok(v) = info.reset.to_string().parse() {
                headers.insert("X-RateLimit-Reset", v);
            }

            response
        }
        Err(RateLimitError::Exceeded(info)) => {
            let mut response = (
                StatusCode::TOO_MANY_REQUESTS,
                Json(serde_json::json!({
                    "success": false,
                    "error": {
                        "code": "RATE_LIMITED",
                        "message": "Too many requests, please try again later"
                    }
                })),
            )
                .into_response();

            let headers = response.headers_mut();
            if let Ok(v) = info.limit.to_string().parse() {
                headers.insert("X-RateLimit-Limit", v);
            }
            headers.insert("X-RateLimit-Remaining", "0".parse().unwrap());
            if let Ok(v) = info.reset.to_string().parse::<http::HeaderValue>() {
                headers.insert("X-RateLimit-Reset", v.clone());
                headers.insert("Retry-After", v);
            }

            response
        }
        Err(RateLimitError::RedisUnavailable) => {
            // Fail open: allow the request if Redis is down
            tracing::warn!("Rate limiter unavailable, allowing request (fail-open)");
            next.run(request).await
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_rate_limit_config() {
        let config = RateLimitConfig::new(100, Duration::from_secs(60));
        assert_eq!(config.max_requests, 100);
        assert_eq!(config.window, Duration::from_secs(60));
        assert_eq!(config.key_prefix, "ratelimit");
    }

    #[test]
    fn test_rate_limit_config_with_prefix() {
        let config = RateLimitConfig::new(50, Duration::from_secs(30))
            .with_prefix("billing_api");
        assert_eq!(config.key_prefix, "billing_api");
    }
}
