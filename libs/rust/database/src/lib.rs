//! Platform Database Library
//!
//! Provides Postgres connection pooling, migration runner, and
//! base repository traits with tenant scoping enforcement.

pub mod migrations;
pub mod pool;
pub mod repository;
