# go-signoz — MEMORY

## [2026-05-31] — Created sf-voice-signoz-go as a separate, zero-dep module (superseded)

**What was decided at the time:** New top-level module
`github.com/sf-voice/sf-voice-signoz-go`
(dir `go-signoz/`), independent from `go/` (`sf-voice-media-go`). It runs two
background loops: continuous `runtime/pprof` CPU profiling pushed to a
Pyroscope-compatible `/ingest`, and a `process.cpu.utilization` gauge exported
over OTLP/HTTP JSON to SigNoz. Slack alerting is **SigNoz config** (alert rule +
Slack channel), not SDK code.

**Why:**
- Separate module keeps any observability concerns out of the media client's
  `go.mod` (the user chose "separate module").
- Stdlib-only (hand-rolled OTLP JSON + Pyroscope multipart push) keeps the build
  verifiable here without `go get`, and matches the media client's no-deps style.
- Both signals on purpose: a flame graph can't be threshold-alerted, so the
  metric is what the "Slack on high CPU" alert fires on; the profile explains
  *where* the CPU went.

**What was rejected:**
- OTel Go SDK for metrics — would add third-party deps; hand-rolled OTLP JSON
  avoids them and the OTLP/HTTP receiver accepts `application/json`.
- Metrics-only and local-Slack-webhook approaches — user chose continuous
  profiling; local Slack would duplicate alert logic outside SigNoz.

**Open risk:** SigNoz's Pyroscope-compatible profiling ingest is not a stable,
documented endpoint (SigNoz#5641). Profiler is verified against Pyroscope's
protocol; the metric→Slack path is the fully SigNoz-supported part.

## [2026-11-05] — Use the monorepo-native Go module path

**What was decided:** The module path is
`github.com/icelab-research/raidzero/go-signoz`, published from matching
`go-signoz/vX.Y.Z` tags in this repository.

**Why:** The previous `github.com/sf-voice/sf-voice-signoz-go` repository does
not exist, so imports and proxy indexing could not work. The monorepo path and
subdirectory tag now describe the same source.

**What was rejected:** Keeping the unavailable module path in anticipation of a
future repository or vanity import.
