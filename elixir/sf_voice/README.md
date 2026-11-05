# sf_voice — Elixir SDK

Elixir client for the sf-voice media API.

Version: `0.1.1`

Ingest audio or video from a URL or S3, let sf-voice index speech, visuals, and
on-screen text, then search your entire media library with natural language.
Results include millisecond timestamps so you can deep-link to the exact moment.

## Installation

```elixir
def deps do
  [{:sf_voice, "~> 0.1.1"}]
end
```

## Usage

```elixir
client = SfVoiceMedia.new(System.fetch_env!("SF_VOICE_API_KEY"))

# ingest a media file — returns immediately with a task id
{:ok, %{task_id: tid}} =
  SfVoiceMedia.ingest(client, %{source: :url, url: "https://example.com/call.mp4"})

# wait for indexing (raises SfVoiceMedia.Error on failure or timeout)
task = SfVoiceMedia.poll_task!(client, tid)

# search with natural language
{:ok, %{results: results}} =
  SfVoiceMedia.search(client, %{
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
`{:error, %SfVoiceMedia.Error{}}`. `poll_task!` raises on task failure or
timeout.

```elixir
case SfVoiceMedia.search(client, %{query: "roadmap"}) do
  {:ok, %{results: results}} -> handle(results)
  {:error, %SfVoiceMedia.Error{code: code, message: msg}} ->
    Logger.error("[#{code}] #{msg}")
end
```

## Examples

There is no standalone Elixir SDK app example yet. This README is the primary
usage example for the package.

## Configuration

```elixir
client = SfVoiceMedia.new("sk-...",
  base_url: "https://staging.api.sf-voice.com",
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

Always call `SfVoiceMedia.stop_alert(handle, client)`. It waits for the worker,
deletes the temporary monitor, and returns polling, callback, or cleanup errors.
