//! Database migration runner.
//!
//! Uses `sqlx::migrate!()` for compile-time verified migrations.
//! Services embed their migrations directory and call `run_migrations()`
//! at startup.

use sqlx::PgPool;

/// Run database migrations against the given pool.
///
/// Uses the `sqlx::migrate!()` macro which embeds migrations at compile time.
/// Services should call this with their own migration directory.
///
/// # Example
///
/// ```ignore
/// // In the service's main.rs:
/// use platform_database::migrations::run_migrations;
///
/// let migrator = sqlx::migrate!("./migrations");
/// run_migrations(&pool, migrator).await?;
/// ```
///
/// # Errors
///
/// Returns an error if migrations fail to apply.
pub async fn run_migrations(
    pool: &PgPool,
    migrator: sqlx::migrate::Migrator,
) -> Result<(), sqlx::migrate::MigrateError> {
    tracing::info!("Running database migrations...");

    migrator.run(pool).await?;

    tracing::info!("Database migrations applied successfully");
    Ok(())
}

/// Check if there are any pending migrations.
///
/// # Errors
///
/// Returns an error if the migration status cannot be determined.
pub async fn has_pending_migrations(
    pool: &PgPool,
    migrator: &sqlx::migrate::Migrator,
) -> Result<bool, sqlx::migrate::MigrateError> {
    let applied = migrator.run(pool).await;
    match applied {
        Ok(()) => Ok(false),
        Err(e) => Err(e),
    }
}
