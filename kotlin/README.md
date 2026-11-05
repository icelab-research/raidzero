# sf-voice-media-kotlin

Kotlin/JVM SDK for the sf-voice media API.

Version: `0.1.1`

## Installation

Gradle:

```kotlin
dependencies {
    implementation("sh.sf-voice:sf-voice-media-kotlin:0.1.1")
}
```

For local development in this repo:

```sh
gradle jar
```

The local jar is written to:

```txt
build/libs/sf-voice-media-0.1.1.jar
```

## Usage

```kotlin
import com.sfvoice.media.IngestRequest
import com.sfvoice.media.MediaType
import com.sfvoice.media.SearchMatchType
import com.sfvoice.media.SearchRequest
import com.sfvoice.media.SfVoiceMediaClient

suspend fun main() {
    val client = SfVoiceMediaClient(
        apiKey = System.getenv("SF_VOICE_API_KEY"),
        baseUrl = "https://api.sf-voice.com",
    )

    try {
        val ingest = client.ingest(
            IngestRequest(
                source = "url",
                assetId = "recording-1",
                url = "https://example.com/recording.mp4",
                mediaType = MediaType.Video,
            )
        )

        val task = client.pollTask(ingest.taskId, intervalMs = 2_000, timeoutMs = 120_000)

        val search = client.search(
            SearchRequest(
                query = "product launch",
                assetIds = listOf(task.assetId),
                types = listOf(SearchMatchType.Conversation),
                threshold = 0.7f,
            )
        )

        println(search.results)
    } finally {
        client.close()
    }
}
```

## API

The client exposes suspend functions:

- `ingest(request)` - submit URL or S3 media for indexing.
- `getTask(taskId)` - fetch task state.
- `pollTask(taskId, intervalMs, timeoutMs)` - wait until a task is terminal.
- `listAssets(page, limit)` - list indexed assets.
- `getAsset(assetId)` - fetch one asset.
- `deleteAsset(assetId)` - soft-delete an asset.
- `search(request)` - search indexed media with natural language.
- `listPrefixes()` - list native transcript prefixes.
- monitor CRUD plus `listMonitorEvents(...)` - manage queries and page events.
- `alert(text, callback, ...)` - poll matches in a supervised coroutine.

Always call `handle.stop()` for an alert. It joins the coroutine, deletes the
temporary monitor, and throws any terminal polling or cleanup failure.

## Examples

- [`../apps/sf-voice/kotlin-example`](../apps/sf-voice/kotlin-example) - Ktor
  REST proxy example.
