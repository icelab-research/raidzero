# sf_voice_media

Async Rust SDK for the sf-voice media API.

Version: `0.1.1`

## Installation

```sh
cargo add sf_voice_media@0.1.1
```

For local development in this repo:

```sh
cargo test -p sf_voice_media
```

## Usage

```rust
use std::time::Duration;

use sf_voice_media::{
    types::{IngestRequest, MediaMetadata, MediaType, SearchRequest, SearchType},
    SfVoiceMedia,
};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = SfVoiceMedia::new(
        "https://api.sf-voice.com",
        std::env::var("SF_VOICE_API_KEY")?,
    );

    let ingest = client
        .ingest(
            &IngestRequest::from_url_with_asset_id("recording-1", "https://example.com/recording.mp4")
                .media_kind(MediaType::Video)
                .metadata(MediaMetadata {
                    title: Some("product demo".into()),
                    tags: Some(vec!["demo".into()]),
                }),
        )
        .await?;

    let task = client
        .poll_task(
            ingest.task_id,
            Duration::from_secs(2),
            Duration::from_secs(120),
        )
        .await?;

    let search = client
        .search(
            &SearchRequest::new("product launch")
                .asset_ids(vec![task.asset_id.clone()])
                .types(vec![SearchType::Conversation])
                .threshold(0.7),
        )
        .await?;

    println!("{:?}", search.results);
    Ok(())
}
```

## API

The client exposes:

- `ingest(&IngestRequest)` - submit URL or S3 media for indexing.
- `get_task(task_id)` - fetch task state.
- `poll_task(task_id, interval, timeout)` - wait until a task is terminal.
- `list_assets(page, limit)` - list indexed assets.
- `get_asset(id)` - fetch one asset.
- `delete_asset(id)` - soft-delete an asset.
- `search(&SearchRequest)` - search indexed media with natural language.
- `list_prefixes()` - list native transcript prefixes.
- monitor CRUD plus `list_monitor_events(...)` - manage queries and page events.
- `alert(text, callback, interval, options)` - poll matches in the background.

Always call `handle.stop().await` for an alert. It aborts polling, deletes the
temporary monitor, and returns polling, callback, or cleanup failures.

## Examples

There is no standalone Rust app example yet. The crate-level examples live in
[`src/lib.rs`](src/lib.rs) and the client examples live in
[`src/client.rs`](src/client.rs).
