# raidzero-media-kotlin

Kotlin/JVM SDK for the raidzero media API.

Version: `0.2.1`

## Installation

Gradle:

```kotlin
dependencies {
    implementation("icu.raidzero:raidzero-media-kotlin:0.2.1")
}
```

For local development in this repo:

```sh
gradle jar
```

The local jar is written to:

```txt
build/libs/raidzero-media-0.2.1.jar
```

## Usage

```kotlin
import icu.raidzero.media.IngestRequest
import icu.raidzero.media.MediaType
import icu.raidzero.media.SearchMatchType
import icu.raidzero.media.SearchRequest
import icu.raidzero.media.RaidzeroMediaClient

suspend fun main() {
    val client = RaidzeroMediaClient(
        apiKey = System.getenv("RAIDZERO_API_KEY"),
        baseUrl = "https://api.raidzero.icu",
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

- [`../apps/raidzero/kotlin-example`](../apps/raidzero/kotlin-example) - Ktor
  REST proxy example.
