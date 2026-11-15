# RaidZero SDKs and documentation

Client libraries for RaidZero, a media observability product by [Ice Lab](https://icelab.sh).

## Install

| SDK | Package | Install |
| --- | --- | --- |
| TypeScript | [`@raidzero/media`](ts/sdk) | `npm install @raidzero/media@0.2.1` |
| Python | [`raidzero-media`](python) | `pip install raidzero-media==0.2.1` |
| Rust | [`raidzero_media`](rust) | `cargo add raidzero_media@0.2.1` |
| Go | [`github.com/icelab-research/raidzero/go`](go) | `go get github.com/icelab-research/raidzero/go@v0.2.1` |
| Java | [`icu.raidzero:raidzero-media-java`](java) | Maven Central version `0.2.1` |
| Kotlin | [`icu.raidzero:raidzero-media-kotlin`](kotlin) | Maven Central version `0.2.1` |
| Elixir | [`raidzero`](elixir/raidzero) | `{:raidzero, "~> 0.2.1"}` |

The [C++ helper](cpp) is source-only. The [Go observability helper](go-signoz)
is a separate Go module.

Start with the [quickstart](docs/quickstart.mdx) or [SDK overview](docs/sdks/overview.mdx).
Each SDK directory contains its API, installation, and build instructions.

This repository is maintained through an automated source export. Open issues
and pull requests here; accepted changes are included in a subsequent export.
