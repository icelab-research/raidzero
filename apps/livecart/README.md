# sf-voice Go SDK — example

A runnable demo covering the full sf-voice media SDK surface.

| Capability                                                           | File                                                       |
| -------------------------------------------------------------------- | ---------------------------------------------------------- |
| Concurrent URL ingest (semaphore + WaitGroup)                        | `internal/example/ingest.go`                               |
| Task polling until ready                                             | `internal/example/ingest.go`                               |
| Typed error handling (`*sfvoice.Error`, `*sfvoice.PollTimeoutError`) | `internal/example/ingest.go`, `internal/example/assets.go` |
| Metadata tagging on ingest                                           | `cmd/example/main.go`                                      |
| S3 ingest path (commented)                                           | `internal/example/ingest.go`                               |
| Semantic search with score threshold + pagination                    | `internal/example/search.go`                               |
| Thumbnail URL in results                                             | `internal/example/search.go`                               |
| List assets (paginated)                                              | `internal/example/assets.go`                               |
| Fetch single asset detail                                            | `internal/example/assets.go`                               |
| Soft-delete assets                                                   | `internal/example/assets.go`                               |

## Setup

```bash
cp .env.example .env
# fill in SF_VOICE_API_KEY
```

| Variable            | Required | Description                             |
| ------------------- | -------- | --------------------------------------- |
| `SF_VOICE_API_KEY`  | yes      | Your API key                            |
| `SF_VOICE_BASE_URL` | no       | Defaults to `https://api.sf-voice.com`  |
| `SAMPLE_MEDIA_URL`  | no       | Fallback URL when `-urls` is not passed |

## Run

```bash
go run ./cmd/example \
  -urls "https://example.com/recording.mp4" \
  -query "product launch"
```

## Flags

| Flag           | Default             | Description                                                            |
| -------------- | ------------------- | ---------------------------------------------------------------------- |
| `-urls`        | `$SAMPLE_MEDIA_URL` | Comma-separated media URLs to ingest                                   |
| `-query`       | _(required)_        | Search query to run after ingest                                       |
| `-types`       |                     | Comma-separated match types: `visual`, `conversation`, `text_in_video` |
| `-threshold`   | `0`                 | Minimum search score 0–1; `0` uses the server default                  |
| `-page`        | `1`                 | Search result page (1-based)                                           |
| `-concurrency` | `4`                 | Max concurrent ingests                                                 |
| `-delete`      | `false`             | Soft-delete the freshly ingested assets after the demo                 |

## Examples

Filter by match type and score threshold:

```bash
go run ./cmd/example \
  -urls "https://example.com/a.mp4,https://example.com/b.mp3" \
  -query "pricing discussion" \
  -types "conversation" \
  -threshold 0.7
```

Ingest and clean up afterwards:

```bash
go run ./cmd/example \
  -urls "https://example.com/recording.mp4" \
  -query "onboarding" \
  -delete
```

## SDK quick-reference

```go
import (
    sfvoice "github.com/icelab-research/raidzero/go"
    "errors"
)

// create a client — safe for concurrent use, create once and reuse
client := sfvoice.New("https://api.sf-voice.com", os.Getenv("SF_VOICE_API_KEY"))

// custom http.Client for retries or proxies
client = sfvoice.NewWithHTTPClient(baseURL, apiKey, &http.Client{Timeout: 60 * time.Second})

// ingest from URL with metadata
resp, err := client.Ingest(ctx, sfvoice.IngestRequest{
    Source:   "url",
    URL:      "https://example.com/call.mp4",
    Metadata: map[string]string{"team": "sales"},
})

// ingest from S3
resp, err = client.Ingest(ctx, sfvoice.IngestRequest{
    Source: "s3",
    S3Key:  "recordings/2024/call.mp4",
})

// poll until the indexing task reaches a terminal state
task, err := client.PollTask(ctx, resp.TaskID, 1500 /*intervalMs*/, 300_000 /*timeoutMs*/)

// typed error handling
var apiErr *sfvoice.Error
if errors.As(err, &apiErr) {
    // apiErr.Code, apiErr.Status, apiErr.Message
}
var timeoutErr *sfvoice.PollTimeoutError
if errors.As(err, &timeoutErr) {
    // asset is still indexing — not a hard failure
}

// search with score threshold and pagination
results, err := client.Search(ctx, sfvoice.SearchRequest{
    Query:     "product launch",
    Types:     []string{"conversation"},
    AssetIDs:  []string{assetID},
    Threshold: 0.7,
    Page:      1,
    Limit:     20,
})

// asset management
list, err  := client.ListAssets(ctx, 1 /*page*/, 20 /*limit*/)
asset, err  = client.GetAsset(ctx, assetID)
err         = client.DeleteAsset(ctx, assetID) // soft-delete; returns nil on 204
```
