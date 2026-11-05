# sf-voice C++ example

Single-header C++17 helper for calling the sf-voice media API. Not a published
package — copy it into your project or pull it in via CMake FetchContent.

Depends on `nlohmann_json` and `cpr`.

## Pull in via CMake

```cmake
include(FetchContent)
FetchContent_Declare(
  sf_voice
  GIT_REPOSITORY https://github.com/icelab-research/raidzero
  GIT_TAG        main
  SOURCE_SUBDIR  cpp
)
FetchContent_MakeAvailable(sf_voice)

target_link_libraries(your_target PRIVATE sf_voice::sf_voice)
```

Or copy `include/sf_voice/` directly into your project and link against CPR and
nlohmann/json yourself.

## Local build

```sh
cmake -S . -B build -DSF_VOICE_FETCH_DEPS=ON -DSF_VOICE_INSTALL=OFF
cmake --build build
```

## Usage

```cpp
#include <cstdlib>
#include <iostream>
#include <sf_voice/sf_voice.hpp>

int main() {
    sf_voice::SfVoiceMedia client(std::getenv("SF_VOICE_API_KEY"));

    sf_voice::IngestRequest req;
    req.url = "https://example.com/call.mp3";
    req.media_kind = "audio";

    auto ingest = client.ingest(req).get();
    if (!ingest.ok) {
        std::cerr << ingest.error.code << ": " << ingest.error.message << "\n";
        return 1;
    }

    auto task = client.poll_task(ingest.value.task_id).get();
    if (!task.ok) { return 1; }

    sf_voice::SearchRequest search_req;
    search_req.query = "customer asks about pricing";

    auto search = client.search(search_req).get();
    std::cout << search.value.results.size() << " results\n";
}
```

## Methods

All methods return `std::future<Result<T>>`. Call `.get()` to block for the
result.

- `ingest(request)` — submit URL or S3 media for indexing
- `get_task(task_id)` — fetch task state
- `poll_task(task_id, interval, timeout)` — wait until ready or failed
- `list_assets(request)` — list indexed assets
- `get_asset(asset_id)` — fetch one asset
- `delete_asset(asset_id)` — soft-delete an asset
- `search(request)` — natural language search over indexed media
- `list_prefixes()` — list native transcript prefixes
- monitor CRUD plus `list_monitor_events(...)` — manage queries and page events
- `alert(text, callback, options)` — poll matches in a background thread

Call `handle.stop_checked()` to join polling and delete the temporary monitor.
Inspect the returned `Result<Empty>` for polling, callback, or cleanup failures.

## Example app

[`../apps/chips`](../apps/chips) — CMake CLI demo using this helper.
