# @sf-voice/media

TypeScript SDK for the sf-voice media API.

Version: `0.1.2`

## Installation

```sh
pnpm add @sf-voice/media@latest
```

```sh
npm install @sf-voice/media@latest
```

```sh
bun install @sf-voice/media@latest
```

## Requirements

- An sf-voice API key.
- The media API base URL.
- A runtime with `fetch`, `FormData`, and `Blob` support.

## Client Setup

```ts
import { SfVoiceMedia } from "@sf-voice/media";

const client = new SfVoiceMedia({
  apiKey: process.env.SF_VOICE_API_KEY!,
});
```

| Option      | Required | Description                                                                                                    |
| ----------- | -------: | -------------------------------------------------------------------------------------------------------------- |
| `apiKey`    |      yes | API key sent as an `Authorization: Bearer` token.                                                              |
| `baseUrl`   |       no | Base URL for the sf-voice media API. Defaults to `SF_VOICE_BASE_URL` env var, then `https://api.sf-voice.com`. |
| `timeoutMs` |       no | Per-request timeout in milliseconds. Defaults to `30_000`.                                                     |

To point the client at a local server, set
`SF_VOICE_BASE_URL=http://localhost:8080` in your environment — no code change
needed. Or pass it explicitly:

```ts
const client = new SfVoiceMedia({
  apiKey: "...",
  baseUrl: "http://localhost:8080",
});
```

## Core Concepts

| Field         | Description                                                                                                                                        |
| ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `asset_id`    | Customer-provided unique id for the media asset. Use your own stable id so you can correlate results with your system. Required on ingest.         |
| `asset_class` | Optional logical group for assets, for example one customer, workspace, repository, or project. Use this to keep search scoped to the right group. |
| `types`       | Optional media surfaces to index or search. Allowed values: `"video"`, `"audio"`, `"transcript"`.                                                  |
| `metadata`    | Optional flat key/value metadata. Values must be strings, numbers, or booleans.                                                                    |
| `threshold`   | Optional minimum match score from `0.0` to `1.0`. Higher values return fewer, more confident search results.                                       |

## Quickstart

```ts
import { SfVoiceMedia, SfVoiceMediaError } from "@sf-voice/media";

const client = new SfVoiceMedia({
  apiKey: process.env.SF_VOICE_API_KEY!,
});

try {
  const ingest = await client.ingest({
    source: "url",
    asset_id: "video_123",
    asset_class: "customer_acme",
    url: "https://example.com/recording.mp4",
    media_kind: "video",
    types: ["video", "audio", "transcript"],
    metadata: {
      title: "product demo",
      customer_id: "acme",
    },
  });

  const task = await client.pollTask(ingest.task_id, {
    intervalMs: 2_000,
    timeoutMs: 120_000,
  });

  if (task.status === "failed") {
    throw new Error(task.error ?? "ingest task failed");
  }

  const search = await client.search({
    query: "where does the customer mention pricing?",
    asset_class: "customer_acme",
    types: ["transcript"],
    threshold: 0.7,
    limit: 10,
  });

  console.log(search.results);
} catch (error) {
  if (error instanceof SfVoiceMediaError) {
    console.error(error.code, error.status, error.message);
  }
  throw error;
}
```

## Ingest

`ingest(request)` submits media for indexing and returns immediately with a task
id.

```ts
const ingest = await client.ingest({
  source: "url",
  asset_id: "video_123",
  asset_class: "customer_acme",
  url: "https://example.com/recording.mp4",
  media_kind: "video",
  types: ["video", "transcript"],
  metadata: { title: "product demo" },
});
```

### Ingest Input

Common fields:

| Field             | Required | Description                                                                                                                                                                    |
| ----------------- | -------: | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `asset_id`        |      yes | Your unique id for the asset.                                                                                                                                                  |
| `asset_class`     |       no | Logical group for the asset. Search can later use this as a scope.                                                                                                             |
| `media_kind`      |       no | `"video"` or `"audio"`.                                                                                                                                                        |
| `types`           |       no | Surfaces to index: `"video"`, `"audio"`, `"transcript"`.                                                                                                                       |
| `metadata`        |       no | Flat key/value metadata for your own correlation.                                                                                                                              |
| `prefix`          |       no | Knowledge-bank namespace. Routes the asset through the native transcript pipeline (whisper + local embeddings). Must be 1–128 chars, letters/numbers/hyphens/underscores only. |
| `transcript_only` |       no | When `true`, skips TwelveLabs entirely and uses the native pipeline. Requires `prefix`.                                                                                        |

Source-specific fields:

| Source | Required Fields    | Description                                               |
| ------ | ------------------ | --------------------------------------------------------- |
| `url`  | `url`              | Ingest from a public or backend-accessible URL.           |
| `s3`   | `s3_key`           | Ingest from an S3 key already connected to sf-voice.      |
| `file` | `file`, `filename` | Upload a `Blob`, `ArrayBuffer`, or `Uint8Array` directly. |

### URL Ingest

```ts
const ingest = await client.ingest({
  source: "url",
  asset_id: "video_123",
  asset_class: "customer_acme",
  url: "https://example.com/recording.mp4",
  media_kind: "video",
  types: ["video", "audio", "transcript"],
});
```

### S3 Ingest

```ts
const ingest = await client.ingest({
  source: "s3",
  asset_id: "support_call_456",
  asset_class: "customer_acme",
  s3_key: "uploads/customer_acme/support_call_456.mp3",
  media_kind: "audio",
  types: ["audio", "transcript"],
});
```

### File Ingest

```ts
const file = await input.files?.[0]?.arrayBuffer();
if (!file) throw new Error("missing file");

const ingest = await client.ingest({
  source: "file",
  asset_id: "browser_upload_789",
  asset_class: "customer_acme",
  file,
  filename: "demo.mp4",
  content_type: "video/mp4",
  media_kind: "video",
  types: ["video", "audio", "transcript"],
});
```

### Native Transcript Pipeline

When `prefix` is set (and optionally `transcript_only: true`), the asset is
routed through the local whisper + embedding pipeline instead of TwelveLabs. All
assets under the same prefix share a Qdrant collection scoped to your org.

```ts
const ingest = await client.ingest({
  source: "file",
  asset_id: "call_2026_001",
  file,
  filename: "call.mp3",
  media_kind: "audio",
  prefix: "support-calls",
  transcript_only: true,
});
```

Search the prefix with a native query to get back results with matched
transcript text:

```ts
const results = await client.search({
  query: "customer mentioned a billing issue",
  prefix: "support-calls",
});

for (const r of results.results) {
  // r.text contains the matched transcript segment
  console.log(r.asset_id, r.text);
}
```

### Ingest Output

```ts
type IngestResponse = {
  asset_id: string;
  task_id: string;
  status: "pending";
};
```

Example:

```json
{
  "asset_id": "video_123",
  "task_id": "task_abc123",
  "status": "pending"
}
```

## Tasks And Polling

Use `getTask(taskId)` to fetch the current task state once. Use
`pollTask(taskId, options)` to wait until the task reaches `"ready"` or
`"failed"`.

```ts
const task = await client.pollTask("task_abc123", {
  intervalMs: 2_000,
  timeoutMs: 120_000,
});
```

### Task Output

```ts
type Task = {
  task_id: string;
  asset_id: string;
  asset_class?: string;
  types: Array<"video" | "audio" | "transcript">;
  status: "pending" | "indexing" | "ready" | "failed";
  error?: string;
  created_at: string;
  completed_at?: string;
};
```

Example:

```json
{
  "task_id": "task_abc123",
  "asset_id": "video_123",
  "asset_class": "customer_acme",
  "types": ["video", "audio", "transcript"],
  "status": "ready",
  "created_at": "2026-05-27T12:00:00Z",
  "completed_at": "2026-05-27T12:01:42Z"
}
```

## Search

`search(request)` searches indexed media with natural language.

Search should usually be scoped with either `asset_ids` or `asset_class`.

```ts
const search = await client.search({
  query: "where does the customer mention pricing?",
  asset_class: "customer_acme",
  types: ["transcript"],
  threshold: 0.7,
  page: 1,
  limit: 10,
});
```

### Search Input

| Field         | Required | Description                                                                                                                                    |
| ------------- | -------: | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `query`       |      yes | Natural-language search query.                                                                                                                 |
| `types`       |       no | Which surfaces to search: `"video"`, `"audio"`, `"transcript"`.                                                                                |
| `asset_ids`   |       no | Restrict search to specific customer asset ids.                                                                                                |
| `asset_class` |       no | Restrict search to one logical group. Recommended for customer-scoped search.                                                                  |
| `scope`       |       no | Set to `"all"` only when intentionally searching across every asset.                                                                           |
| `prefix`      |       no | Search the native Qdrant collection for this prefix instead of TwelveLabs. Results include a `text` field with the matched transcript segment. |
| `threshold`   |       no | Minimum match score from `0.0` to `1.0`. Defaults to the API default.                                                                          |
| `page`        |       no | Page number.                                                                                                                                   |
| `limit`       |       no | Max results per page.                                                                                                                          |

### Search By Asset Class

```ts
const results = await client.search({
  query: "refund policy",
  asset_class: "customer_acme",
  types: ["transcript"],
});
```

### Search Specific Assets

```ts
const results = await client.search({
  query: "installation steps",
  asset_ids: ["video_123", "video_456"],
  types: ["video", "transcript"],
});
```

### Search All Assets

```ts
const results = await client.search({
  query: "security review",
  scope: "all",
  types: ["transcript"],
});
```

### Search Output

```ts
type SearchResponse = {
  results: Array<{
    asset_id: string;
    score: number;
    start_ms: number;
    end_ms: number;
    match_type: "video" | "audio" | "transcript";
    thumbnail_url?: string;
    text?: string; // only present on native prefix searches
  }>;
  page_info: {
    total: number;
    page: number;
    limit: number;
    next_page_token?: string;
  };
};
```

Example:

```json
{
  "results": [
    {
      "asset_id": "video_123",
      "score": 0.84,
      "start_ms": 42000,
      "end_ms": 58000,
      "match_type": "transcript",
      "thumbnail_url": "https://api.sf-voice.com/assets/video_123/thumb.jpg"
    }
  ],
  "page_info": {
    "total": 1,
    "page": 1,
    "limit": 10
  }
}
```

## Prefixes

`listPrefixes()` returns all native-pipeline prefixes the org has used.
`owned_by_caller` is `true` when the prefix was first created by the current API
key.

```ts
const { items } = await client.listPrefixes();

for (const p of items) {
  console.log(p.prefix, p.owned_by_caller);
}
```

## Assets

### List Assets

```ts
const assets = await client.listAssets({
  page: 1,
  limit: 20,
});
```

### Get Asset

```ts
const asset = await client.getAsset("video_123");
```

### Delete Asset

```ts
await client.deleteAsset("video_123");
```

### Asset Output

```ts
type Asset = {
  asset_id: string;
  asset_class?: string;
  media_type: "video" | "audio";
  source_type: "url" | "s3" | "file";
  types: Array<"video" | "audio" | "transcript">;
  status: "pending" | "indexing" | "ready" | "failed";
  metadata?: Record<string, string | number | boolean>;
  duration_ms?: number;
  created_at: string;
  updated_at: string;
};
```

## Errors

Every non-2xx API response throws `SfVoiceMediaError`.

```ts
import { SfVoiceMediaError } from "@sf-voice/media";

try {
  await client.search({
    query: "pricing",
    asset_class: "customer_acme",
  });
} catch (error) {
  if (error instanceof SfVoiceMediaError) {
    console.error(error.code);
    console.error(error.status);
    console.error(error.message);
  }
  throw error;
}
```

Known API error codes include:

- `bucket_not_connected`
- `s3_access_denied`
- `s3_key_not_found`
- `unsupported_format`
- `file_too_large`
- `provider_unavailable`
- `unauthorized`
- `not_found`
- `rate_limited`

`SfVoiceMediaRequestTimeoutError` is thrown when a single HTTP request exceeds
the client timeout.

`SfVoiceMediaPollTimeoutError` is thrown when `pollTask` exceeds its polling
timeout before the task reaches `"ready"` or `"failed"`.

## API Surface

```ts
client.ingest(request): Promise<IngestResponse>
client.getTask(taskId): Promise<Task>
client.pollTask(taskId, options?): Promise<Task>
client.listAssets(params?): Promise<AssetListResponse>
client.getAsset(assetId): Promise<Asset>
client.deleteAsset(assetId): Promise<void>
client.search(request): Promise<SearchResponse>
client.listPrefixes(): Promise<PrefixListResponse>
client.createMonitor(request): Promise<Monitor>
client.listMonitors(): Promise<MonitorListResponse>
client.getMonitor(monitorId): Promise<Monitor>
client.updateMonitor(monitorId, request): Promise<Monitor>
client.deleteMonitor(monitorId): Promise<void>
client.listMonitorEvents(monitorId, params?): Promise<MonitorEventListResponse>
client.alert(text, callback, options?): Promise<AlertHandle>
```

## Examples

- [`fifteenlabs`](../../apps/fifteenlabs) - browser ingest and search demo.
