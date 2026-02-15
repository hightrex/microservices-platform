//! Base repository traits and tenant-scoped query helpers.
//!
//! Enforces tenant isolation at the data layer. All tenant-owned
//! queries MUST go through the `TenantScoped` trait or use the
//! `TenantScope` helper to ensure proper `tenant_id` filtering.

use platform_common::tenant::TenantContext;
use sqlx::PgPool;
use uuid::Uuid;

/// Trait for repositories that operate on tenant-scoped data.
///
/// All methods receive a `TenantContext` to ensure tenant isolation
/// is enforced at the repository layer (not remembered per handler).
pub trait TenantScoped {
    /// Get the table name this repository operates on.
    fn table_name(&self) -> &str;
}

/// Helper for building tenant-scoped queries.
///
/// Ensures that every query touching tenant-owned data includes
/// the `tenant_id` constraint. This is the Rust equivalent of
/// Go's `tenant.NewScope(ctx)`.
#[derive(Debug, Clone)]
pub struct TenantScope {
    pub tenant_id: Uuid,
    pub org_id: Option<Uuid>,
}

impl TenantScope {
    /// Create a new tenant scope from a `TenantContext`.
    pub fn from_context(ctx: &TenantContext) -> Self {
        Self {
            tenant_id: ctx.tenant_id,
            org_id: ctx.org_id,
        }
    }

    /// Get the tenant ID for use in queries.
    pub fn tenant_id(&self) -> Uuid {
        self.tenant_id
    }
}

/// Pagination parameters for list queries.
#[derive(Debug, Clone)]
pub struct Pagination {
    pub page: u32,
    pub per_page: u32,
}

impl Pagination {
    pub fn new(page: u32, per_page: u32) -> Self {
        Self {
            page: page.max(1),
            per_page: per_page.clamp(1, 100),
        }
    }

    /// Calculate the SQL OFFSET value.
    pub fn offset(&self) -> i64 {
        ((self.page - 1) * self.per_page) as i64
    }

    /// Get the SQL LIMIT value.
    pub fn limit(&self) -> i64 {
        self.per_page as i64
    }
}

impl Default for Pagination {
    fn default() -> Self {
        Self {
            page: 1,
            per_page: 20,
        }
    }
}

/// Paginated response wrapper.
#[derive(Debug, Clone, serde::Serialize)]
pub struct PaginatedResponse<T: serde::Serialize> {
    pub data: Vec<T>,
    pub page: u32,
    pub per_page: u32,
    pub total: i64,
    pub total_pages: u32,
}

impl<T: serde::Serialize> PaginatedResponse<T> {
    pub fn new(data: Vec<T>, pagination: &Pagination, total: i64) -> Self {
        let total_pages = if total == 0 {
            0
        } else {
            ((total as f64) / (pagination.per_page as f64)).ceil() as u32
        };

        Self {
            data,
            page: pagination.page,
            per_page: pagination.per_page,
            total,
            total_pages,
        }
    }
}

/// Base repository providing common database access patterns.
///
/// Services compose this into their specific repositories.
#[derive(Clone)]
pub struct BaseRepository {
    pool: PgPool,
}

impl BaseRepository {
    /// Create a new base repository with the given connection pool.
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// Get a reference to the connection pool.
    pub fn pool(&self) -> &PgPool {
        &self.pool
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_pagination_defaults() {
        let p = Pagination::default();
        assert_eq!(p.page, 1);
        assert_eq!(p.per_page, 20);
        assert_eq!(p.offset(), 0);
        assert_eq!(p.limit(), 20);
    }

    #[test]
    fn test_pagination_offset() {
        let p = Pagination::new(3, 25);
        assert_eq!(p.offset(), 50);
        assert_eq!(p.limit(), 25);
    }

    #[test]
    fn test_pagination_clamps() {
        let p = Pagination::new(0, 200);
        assert_eq!(p.page, 1);
        assert_eq!(p.per_page, 100);
    }

    #[test]
    fn test_tenant_scope_from_context() {
        let ctx = TenantContext::new(Uuid::new_v4(), Some(Uuid::new_v4()));
        let scope = TenantScope::from_context(&ctx);
        assert_eq!(scope.tenant_id, ctx.tenant_id);
        assert_eq!(scope.org_id, ctx.org_id);
    }

    #[test]
    fn test_paginated_response() {
        let pagination = Pagination::new(2, 10);
        let data = vec![1, 2, 3, 4, 5];
        let response = PaginatedResponse::new(data, &pagination, 25);

        assert_eq!(response.page, 2);
        assert_eq!(response.per_page, 10);
        assert_eq!(response.total, 25);
        assert_eq!(response.total_pages, 3);
        assert_eq!(response.data.len(), 5);
    }

    #[test]
    fn test_paginated_response_empty() {
        let pagination = Pagination::default();
        let data: Vec<i32> = vec![];
        let response = PaginatedResponse::new(data, &pagination, 0);

        assert_eq!(response.total_pages, 0);
        assert!(response.data.is_empty());
    }
}
