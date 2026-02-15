//! File domain models.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::Type;
use uuid::Uuid;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Type)]
#[sqlx(type_name = "file_status", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum FileStatus {
    Uploading,
    Available,
    Deleted,
    Quarantined,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Type)]
#[sqlx(type_name = "virus_scan_status", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum VirusScanStatus {
    Pending,
    Clean,
    Infected,
    Error,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct File {
    pub id: Uuid,
    pub tenant_id: Uuid,
    pub user_id: Uuid,
    pub org_id: Uuid,
    pub filename: String,
    pub content_type: String,
    pub size_bytes: i64,
    pub s3_key: String,
    pub s3_bucket: String,
    pub checksum_sha256: Option<String>,
    pub status: FileStatus,
    pub virus_scan_status: VirusScanStatus,
    pub thumbnail_s3_key: Option<String>,
    pub uploaded_at: Option<DateTime<Utc>>,
    pub deleted_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize)]
pub struct FileResponse {
    pub id: Uuid,
    pub filename: String,
    pub content_type: String,
    pub size_bytes: i64,
    pub status: FileStatus,
    pub virus_scan_status: VirusScanStatus,
    pub checksum_sha256: Option<String>,
    pub uploaded_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

impl From<File> for FileResponse {
    fn from(f: File) -> Self {
        Self {
            id: f.id,
            filename: f.filename,
            content_type: f.content_type,
            size_bytes: f.size_bytes,
            status: f.status,
            virus_scan_status: f.virus_scan_status,
            checksum_sha256: f.checksum_sha256,
            uploaded_at: f.uploaded_at,
            created_at: f.created_at,
        }
    }
}
