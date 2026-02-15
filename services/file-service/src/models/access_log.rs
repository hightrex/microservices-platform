//! File access log domain models.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::Type;
use uuid::Uuid;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Type)]
#[sqlx(type_name = "access_action", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum AccessAction {
    Upload,
    Download,
    Delete,
    View,
}

#[derive(Debug, Clone, Serialize, sqlx::FromRow)]
pub struct FileAccessLog {
    pub id: Uuid,
    pub file_id: Uuid,
    pub tenant_id: Uuid,
    pub user_id: Uuid,
    pub action: AccessAction,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
    pub created_at: DateTime<Utc>,
}
