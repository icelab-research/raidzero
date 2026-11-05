# sf-voice-media

Python SDK for the sf-voice media API.

Version: `0.2.0`

## Installation

```sh
pip install sf-voice-media==0.2.0
```

For local development in this repo:

```sh
python3 -m venv .venv
source .venv/bin/activate
pip install -e .
python3 -m compileall -q sf_voice
```

## Requirements

- An sf-voice API key.
- The media API base URL.

## Client Setup

```python
import os

from sf_voice import SfVoiceMedia

client = SfVoiceMedia(
    base_url="https://api.sf-voice.com",
    api_key=os.environ["SF_VOICE_API_KEY"],
    timeout_ms=30_000,
)
```

## Core Concepts

| Field         | Description                                                                                       |
| ------------- | ------------------------------------------------------------------------------------------------- |
| `asset_id`    | Customer-provided unique id for the media asset. Required on ingest.                              |
| `asset_class` | Optional logical group for assets, for example one customer, workspace, repository, or project.   |
| `types`       | Optional media surfaces to index or search. Allowed values: `"video"`, `"audio"`, `"transcript"`. |
| `metadata`    | Optional flat key/value metadata. Values must be strings, numbers, or booleans.                   |
| `threshold`   | Optional minimum match score from `0.0` to `1.0`.                                                 |

## Quickstart

```python
import os

from sf_voice import SfVoiceMedia, SfVoiceMediaError

client = SfVoiceMedia(
    base_url="https://api.sf-voice.com",
    api_key=os.environ["SF_VOICE_API_KEY"],
)

try:
    ingest = client.ingest(
        {
            "source": "url",
            "asset_id": "video_123",
            "asset_class": "customer_acme",
            "url": "https://example.com/recording.mp4",
            "media_kind": "video",
            "types": ["video", "audio", "transcript"],
            "metadata": {
                "title": "product demo",
                "customer_id": "acme",
            },
        }
    )

    task = client.poll_task(
        ingest.task_id,
        {"interval_ms": 2_000, "timeout_ms": 120_000},
    )

    if task.status == "failed":
        raise RuntimeError(task.error or "ingest task failed")

    search = client.search(
        {
            "query": "where does the customer mention pricing?",
            "asset_class": "customer_acme",
            "types": ["transcript"],
            "threshold": 0.7,
            "limit": 10,
        }
    )

    print(search.results)
except SfVoiceMediaError as error:
    print(error.code, error.status, error.message)
    raise
finally:
    client.close()
```

## Ingest

`ingest(request)` submits media for indexing and returns immediately with a task
id.

### URL Ingest

```python
ingest = client.ingest(
    {
        "source": "url",
        "asset_id": "video_123",
        "asset_class": "customer_acme",
        "url": "https://example.com/recording.mp4",
        "media_kind": "video",
        "types": ["video", "audio", "transcript"],
    }
)
```

### S3 Ingest

```python
ingest = client.ingest(
    {
        "source": "s3",
        "asset_id": "support_call_456",
        "asset_class": "customer_acme",
        "s3_key": "uploads/customer_acme/support_call_456.mp3",
        "media_kind": "audio",
        "types": ["audio", "transcript"],
    }
)
```

### File Ingest

```python
with open("demo.mp4", "rb") as file:
    ingest = client.ingest(
        {
            "source": "file",
            "asset_id": "browser_upload_789",
            "asset_class": "customer_acme",
            "file": file,
            "filename": "demo.mp4",
            "content_type": "video/mp4",
            "media_kind": "video",
            "types": ["video", "audio", "transcript"],
        }
    )
```

## Search

`search(request)` searches indexed media with natural language.

Search should usually be scoped with either `asset_ids` or `asset_class`.

```python
search = client.search(
    {
        "query": "refund policy",
        "asset_class": "customer_acme",
        "types": ["transcript"],
        "threshold": 0.7,
        "page": 1,
        "limit": 10,
    }
)
```

To intentionally search every asset, pass `{"scope": "all"}`.

## Assets

```python
assets = client.list_assets({"page": 1, "limit": 20})
asset = client.get_asset("video_123")
client.delete_asset("video_123")
```

## Async Client

```python
import os

from sf_voice import AsyncSfVoiceMedia

async with AsyncSfVoiceMedia(
    base_url="https://api.sf-voice.com",
    api_key=os.environ["SF_VOICE_API_KEY"],
) as client:
    ingest = await client.ingest(
        {
            "source": "url",
            "asset_id": "video_123",
            "url": "https://example.com/recording.mp4",
        }
    )
    task = await client.poll_task(ingest.task_id)
```

## Errors

Every non-2xx API response raises `SfVoiceMediaError`.

```python
from sf_voice import SfVoiceMediaError

try:
    client.search({"query": "pricing", "asset_class": "customer_acme"})
except SfVoiceMediaError as error:
    print(error.code)
    print(error.status)
    print(error.message)
    raise
```

`SfVoiceMediaRequestTimeoutError` is raised when a single HTTP request exceeds
the client timeout.

`SfVoiceMediaPollTimeoutError` is raised when `poll_task` exceeds its polling
timeout before the task reaches `"ready"` or `"failed"`.

## API Surface

```python
client.ingest(request) -> IngestResponse
client.get_task(task_id) -> Task
client.poll_task(task_id, options=None) -> Task
client.list_assets(params=None) -> AssetListResponse
client.get_asset(asset_id) -> Asset
client.delete_asset(asset_id) -> None
client.search(request) -> SearchResponse
client.list_prefixes() -> PrefixListResponse
client.create_monitor(request) -> Monitor
client.list_monitor_events(monitor_id, matched_only, limit, offset)
client.alert(text, callback, ...) -> AlertHandle
```

The async client exposes the same monitor surface with awaitable methods and an
`AsyncAlertHandle`. Always call `handle.stop()` or `await handle.stop()` so the
worker exits, the temporary monitor is deleted, and terminal errors surface.
