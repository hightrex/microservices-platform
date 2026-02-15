//! Platform Messaging Library
//!
//! Redis Streams integration for event-driven architecture.
//! Provides producer, consumer, and event schema definitions
//! matching the Go `libs/go/pkg/messaging` package.

pub mod consumer;
pub mod producer;
pub mod schema;
