# sf-voice-media-java

Java 17 SDK for the sf-voice media API.

Version: `0.1.1`

## Installation

Gradle:

```kotlin
dependencies {
    implementation("sh.sf-voice:sf-voice-media-java:0.1.1")
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

```java
import com.sfvoice.media.SfVoiceMediaClient;
import com.sfvoice.media.SfVoiceMediaException;
import com.sfvoice.media.models.IngestRequest;
import com.sfvoice.media.models.SearchRequest;
import com.sfvoice.media.models.SearchResponse;
import com.sfvoice.media.models.Task;

import java.util.List;
import java.util.Map;

public class Example {
    public static void main(String[] args) throws Exception {
        SfVoiceMediaClient client = new SfVoiceMediaClient.Builder()
            .apiKey(System.getenv("SF_VOICE_API_KEY"))
            .baseUrl("https://api.sf-voice.com")
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
        } catch (SfVoiceMediaException error) {
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

- [`../apps/sf-voice/java-example`](../apps/sf-voice/java-example) - Spring Boot
  REST proxy example.
