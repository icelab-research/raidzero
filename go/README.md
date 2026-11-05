# sf-voice-media-go

Go SDK for the sf-voice media API.

Version: `v0.1.1`

## Installation

```sh
go get github.com/icelab-research/raidzero/go@v0.1.1
```

For local development in this repo:

```sh
go test ./...
```

## Usage

```go
package main

import (
 "context"
 "errors"
 "fmt"
 "os"

 sfvoice "github.com/icelab-research/raidzero/go"
)

func main() {
 ctx := context.Background()
 client := sfvoice.New("https://api.sf-voice.com", os.Getenv("SF_VOICE_API_KEY"))

 ingest, err := client.Ingest(ctx, sfvoice.IngestRequest{
  Source:    "url",
  AssetID:   "recording-1",
  URL:       "https://example.com/recording.mp4",
  MediaKind: "video",
  Metadata:  map[string]any{"title": "product demo"},
 })
 if err != nil {
  panic(err)
 }

 task, err := client.PollTask(ctx, ingest.TaskID, 1500, 120000)
 if err != nil {
  var apiErr *sfvoice.Error
  if errors.As(err, &apiErr) {
   fmt.Println(apiErr.Code, apiErr.Status, apiErr.Message)
  }
  panic(err)
 }

 results, err := client.Search(ctx, sfvoice.SearchRequest{
  Query:     "product launch",
  AssetIDs:  []string{task.AssetID},
  Types:     []string{"conversation"},
  Threshold: 0.7,
 })
 if err != nil {
  panic(err)
 }

 fmt.Println(results.Results)
}
```

## API

The client exposes:

- `Ingest(ctx, request)` - submit URL or S3 media for indexing.
- `GetTask(ctx, taskID)` - fetch task state.
- `PollTask(ctx, taskID, intervalMs, timeoutMs)` - wait until a task is
  terminal.
- `ListAssets(ctx, page, limit)` - list indexed assets.
- `GetAsset(ctx, assetID)` - fetch one asset.
- `DeleteAsset(ctx, assetID)` - soft-delete an asset.
- `Search(ctx, request)` - search indexed media with natural language.
- `ListPrefixes(ctx)` - list native transcript prefixes.
- `CreateMonitor`, `ListMonitors`, `GetMonitor`, `UpdateMonitor`, and
  `DeleteMonitor` - manage continuous rich-media queries.
- `ListMonitorEvents(ctx, monitorID, matchedOnly, limit, offset)` - page events.
- `Alert(ctx, text, callback, options)` - poll matches in the background.

Always call `handle.Stop()` for an alert. It waits for polling, deletes the
temporary monitor, and returns polling, callback, or cleanup failures.

## Examples

- [`../apps/livecart`](../apps/livecart) - CLI demo for ingest, poll, search,
  listing, and cleanup.
