//! # raidzero-media
//!
//! async Rust client for the raidzero media API.
//!
//! ## quick start
//!
//! ```no_run
//! use raidzero_media::{RaidzeroMedia, types::IngestRequest};
//! use std::time::Duration;
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     let client = RaidzeroMedia::new("https://api.raidzero.icu", "your-api-key");
//!
//!     let ingest = client
//!         .ingest(&IngestRequest::from_url_with_asset_id("video-1", "https://example.com/video.mp4"))
//!         .await?;
//!
//!     let task = client
//!         .poll_task(ingest.task_id, Duration::from_secs(2), Duration::from_secs(120))
//!         .await?;
//!
//!     println!("status: {:?}", task.status);
//!     Ok(())
//! }
//! ```

pub mod client;
pub mod error;
pub mod types;

// flatten the most-used items to the crate root for ergonomic imports
pub use client::RaidzeroMedia;
pub use error::RaidzeroMediaError;
