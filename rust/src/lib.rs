//! # sf-voice-media
//!
//! async Rust client for the sf-voice media API.
//!
//! ## quick start
//!
//! ```no_run
//! use sf_voice_media::{SfVoiceMedia, types::IngestRequest};
//! use std::time::Duration;
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     let client = SfVoiceMedia::new("https://api.sf-voice.com", "your-api-key");
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
pub use client::SfVoiceMedia;
pub use error::SfVoiceMediaError;
