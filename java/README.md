# raidzero-media-java

Java 17 SDK for the raidzero media API.

Version: `0.2.1`

## Installation

Gradle:

```kotlin
dependencies {
    implementation("icu.raidzero:raidzero-media-java:0.2.1")
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

```java
import icu.raidzero.media.RaidzeroMediaClient;
import icu.raidzero.media.RaidzeroMediaException;
import icu.raidzero.media.models.IngestRequest;
import icu.raidzero.media.models.SearchRequest;
import icu.raidzero.media.models.SearchResponse;
import icu.raidzero.media.models.Task;

import java.util.List;
import java.util.Map;

public class Example {
    public static void main(String[] args) throws Exception {
        RaidzeroMediaClient client = new RaidzeroMediaClient.Builder()
            .apiKey(System.getenv("RAIDZERO_API_KEY"))
            .baseUrl("https://api.raidzero.icu")
            .build();

        try {
            var ingest = client.ingest(
                IngestRequest.fromUrl("recording-1", "https://example.com/recording.mp4")
                    .mediaKind("video")
                    .metadata(Map.of("title", "product demo"))
                    .build()
            );

            Task task = client.pollTask(ingest.getTaskId(), 2_000, 120_000);

            SearchResponse search = client.search(
                SearchRequest.query("product launch")
                    .assetIds(List.of(task.getAssetId()))
                    .types(List.of("conversation"))
                    .threshold(0.7)
                    .build()
            );

            System.out.println(search.getResults());
        } catch (RaidzeroMediaException error) {
            System.err.println(error.getCode() + " " + error.getStatus() + " " + error.getMessage());
            throw error;
        }
    }
}
```

## API

The client exposes blocking methods:

- `ingest(request)` - submit URL or S3 media for indexing.
- `getTask(taskId)` - fetch task state.
- `pollTask(taskId, intervalMs, timeoutMs)` - wait until a task is terminal.
- `listAssets(page, limit)` - list indexed assets.
- `getAsset(assetId)` - fetch one asset.
- `deleteAsset(assetId)` - soft-delete an asset.
- `search(request)` - search indexed media with natural language.
- `listPrefixes()` - list native transcript prefixes.
- monitor CRUD plus `listMonitorEvents(...)` - manage queries and page events.
- `alert(text, callback, options)` - poll matches in a background thread.

Always call `handle.stop()` for an alert. It joins the worker, deletes the
temporary monitor, and throws any terminal polling or cleanup failure.

## Examples

- [`../apps/raidzero/java-example`](../apps/raidzero/java-example) - Spring Boot
  REST proxy example.
