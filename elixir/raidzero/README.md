# raidzero — Elixir SDK

Elixir client for the raidzero media API.

Version: `0.2.1`

Ingest audio or video from a URL or S3, let raidzero index speech, visuals, and
on-screen text, then search your entire media library with natural language.
Results include millisecond timestamps so you can deep-link to the exact moment.

## Installation

```elixir
def deps do
  [{:raidzero, "~> 0.2.1"}]
end
```

## Usage

```elixir
client = RaidzeroMedia.new(System.fetch_env!("RAIDZERO_API_KEY"))

# ingest a media file — returns immediately with a task id
{:ok, %{task_id: tid}} =
  RaidzeroMedia.ingest(client, %{source: :url, url: "https://example.com/call.mp4"})

# wait for indexing (raises RaidzeroMedia.Error on failure or timeout)
task = RaidzeroMedia.poll_task!(client, tid)

# search with natural language
{:ok, %{results: results}} =
  RaidzeroMedia.search(client, %{
    query: "quarterly targets",
    types: [:conversation],
    threshold: 0.7
  })

Enum.each(results, fn r ->
  IO.puts("#{r.asset_id} at #{r.start_ms}ms — #{r.match_type}")
end)
```

## Error handling

All functions except `poll_task!` return `{:ok, result}` or
`{:error, %RaidzeroMedia.Error{}}`. `poll_task!` raises on task failure or
timeout.

```elixir
case RaidzeroMedia.search(client, %{query: "roadmap"}) do
  {:ok, %{results: results}} -> handle(results)
  {:error, %RaidzeroMedia.Error{code: code, message: msg}} ->
    Logger.error("[#{code}] #{msg}")
end
```

## Examples

There is no standalone Elixir SDK app example yet. This README is the primary
usage example for the package.

## Configuration

```elixir
client = RaidzeroMedia.new("sk-...",
  base_url: "https://staging.api.raidzero.icu",
  http_opts: [receive_timeout: 10_000]
)
```

## Match types

Search results carry a `match_type` string — what the query matched against:

| Value             | What was searched     |
| ----------------- | --------------------- |
| `"conversation"`  | Transcribed speech    |
| `"visual"`        | Visual scene analysis |
| `"text_in_video"` | On-screen text (OCR)  |

## Observability

Use `list_prefixes/1`, monitor CRUD, and `list_monitor_events/3` to operate
continuous rich-media queries. `alert/4` adds bounded background polling and
oldest-to-newest callbacks.

Always call `RaidzeroMedia.stop_alert(handle, client)`. It waits for the worker,
deletes the temporary monitor, and returns polling, callback, or cleanup errors.
