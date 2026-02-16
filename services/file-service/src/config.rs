//! File service configuration.

use platform_common::config::{DatabaseConfig, RedisConfig, ServerConfig};
use platform_common::logger::LoggerConfig;
use serde::Deserialize;

#[derive(Debug, Clone, Deserialize)]
pub struct FileServiceConfig {
    pub server: ServerConfig,
    pub database: DatabaseConfig,
    pub redis: RedisConfig,
    pub s3: S3Config,
    pub logger: LoggerConfig,
    pub quota: QuotaConfig,
    pub upload: UploadConfig,
    pub auth: AuthenticationConfig,
}

/// Authentication configuration for JWT validation.
#[derive(Debug, Clone, Deserialize)]
pub struct AuthenticationConfig {
    /// JWT signing secret (shared with auth-service).
    pub jwt_secret: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct S3Config {
    pub endpoint: String,
    #[serde(default = "default_region")]
    pub region: String,
    pub access_key: String,
    pub secret_key: String,
    #[serde(default = "default_bucket_prefix")]
    pub bucket_prefix: String,
}

fn default_region() -> String {
    "us-east-1".to_string()
}

fn default_bucket_prefix() -> String {
    "platform".to_string()
}

#[derive(Debug, Clone, Deserialize)]
pub struct QuotaConfig {
    #[serde(default = "default_quota_limit")]
    pub default_limit_bytes: i64,
    #[serde(default = "default_max_file_size")]
    pub max_file_size_bytes: i64,
}

fn default_quota_limit() -> i64 {
    1_073_741_824 // 1 GB
}

fn default_max_file_size() -> i64 {
    104_857_600 // 100 MB
}

#[derive(Debug, Clone, Deserialize)]
pub struct UploadConfig {
    #[serde(default)]
    pub allowed_content_types: Vec<String>,
}
