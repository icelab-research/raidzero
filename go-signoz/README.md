# sf-voice-signoz-go

Zero-dependency Go SDK that ships **CPU profiles** and **CPU-utilization
metrics** to SigNoz, so SigNoz can flame-graph hot paths and fire a **Slack
alert when CPU is high**.

```go
import signoz "github.com/icelab-research/raidzero/go-signoz"
```

## What it does

One call starts two independent background loops:

- **Metrics:** samples process CPU through `getrusage` and exports OTLP/HTTP
  JSON to the SigNoz collector's `/v1/metrics` endpoint. This answers whether
  CPU usage is high and provides the timeseries used by alerts.
- **Profiler:** continuously captures `runtime/pprof` CPU profiles and pushes
  them to the Pyroscope-compatible `/ingest` endpoint. This produces flame
  graphs that show where CPU time is spent.

> **The SDK never talks to Slack.** A flame graph can't be threshold-alerted, so
> the metric is what makes "Slack on high CPU" possible. The Slack message
> itself is a **SigNoz alert rule + Slack notification channel** (steps below).

## Usage

```go
agent, err := signoz.Start(signoz.Config{
    ServiceName:     "sf-voice-api",
    MetricsEndpoint: "https://ingest.us.signoz.cloud:443",
    MetricsHeaders: map[string]string{
        "signoz-ingestion-key": os.Getenv("SIGNOZ_KEY"),
    },

    ProfilesEndpoint:  "http://localhost:4040",
    ProfilesAuthToken: os.Getenv("PYROSCOPE_TOKEN"),

    Labels: map[string]string{"env": "prod", "region": "sfo"},
})
if err != nil {
    log.Fatal(err)
}
defer agent.Stop()
```

Either endpoint may be omitted to disable that loop; at least one is required.
Defaults: `MetricsInterval` 15s, `ProfileInterval` 10s, 30s HTTP timeout,
`log.Default()` for non-fatal upload errors.

A runnable demo (burns a core so the metric climbs) lives in `./example`.

## Metrics emitted

- `process.cpu.utilization` is a `[0,1]` gauge where `1.0` means the process has
  saturated its available CPU capacity. Its calculation is
  `ΔCPU / (Δwall · available CPUs)`.
- `process.runtime.go.goroutines` reports the live goroutine count as a gauge.

Current SigNoz deployments keep these dotted metric names. Older deployments may
normalize them to `process_cpu_utilization` and `process_runtime_go_goroutines`.

## Wiring "high CPU → Slack" in SigNoz (config, not code)

1. **Point metrics at SigNoz.** Self-host:
   `MetricsEndpoint=http://<collector>:4318`. Cloud: the regional ingest URL +
   `signoz-ingestion-key` header (shown above).
2. **Confirm the metric.** SigNoz → _Metrics_ and search
   `process.cpu.utilization`. On an older deployment, search
   `process_cpu_utilization` instead.
3. **Create the alert.** _Alerts → New Alert → Metrics-based_:
   - Metric `process.cpu.utilization`, aggregation **avg**, grouped by
     `host.name`. Older deployments may use `process_cpu_utilization` and
     `host_name`.
   - Condition: **above `0.85` for 5m** (utilization is `0–1`, so `0.85` = 85%).
4. **Add a Slack channel.** _Settings → Alert Channels → New → Slack_, paste an
   [incoming webhook](https://api.slack.com/messaging/webhooks) URL, pick the
   channel.
5. **Attach** the channel to the alert. SigNoz now posts to Slack whenever CPU
   stays above your threshold.

## Profiling target

The profiler speaks the **Pyroscope pprof ingest** protocol (`POST /ingest` with
a multipart `profile` field; `name`/`from`/`until`/`format=pprof`/`sampleRate`/
`spyName=gospy` as query params). Point `ProfilesEndpoint` at a Pyroscope
server, or at SigNoz's profiling ingest if your deployment exposes one.

> **Heads-up:** as of this writing SigNoz's Pyroscope-compatible profiling
> ingest is not a stable, documented endpoint
> ([SigNoz#5641](https://github.com/SigNoz/signoz/issues/5641)). The profiler
> works today against Pyroscope; the **metric → Slack alert** path is the part
> that runs entirely on SigNoz's supported OTLP + alerting stack. Leave
> `ProfilesEndpoint` empty to run metrics-only.

## CI & releasing

- **CI:** `.github/workflows/go-signoz.yml` runs on PRs and `main` pushes that
  touch `go-signoz/**` — `gofmt` check, `go vet`, build, `go test -race`, and
  windows/darwin cross-compiles.
- **Release:** Go modules publish by **git tag**, not a registry upload. Tag the
  commit `go-signoz/vX.Y.Z` and push it;
  `.github/workflows/publish-go-signoz-sdk.yml` re-verifies (vet/test/build) and
  asks the Go proxy to index the version. You can also run it manually via
  _workflow_dispatch_ by selecting the matching `go-signoz/vX.Y.Z` tag and
  entering `X.Y.Z` as the `version` input.

  ```sh
  git tag go-signoz/v0.1.0
  git push origin go-signoz/v0.1.0
  ```

  > The proxy-index step is best-effort:
  > `go get github.com/icelab-research/raidzero/go-signoz@vX.Y.Z` resolves from
  > the monorepo's `go-signoz/vX.Y.Z` tag.

## Platform support

Process CPU sampling uses `getrusage(RUSAGE_SELF)` (Linux + macOS,
`//go:build unix`). On other platforms the metrics loop logs an
unsupported-platform error once and the profiler loop still runs. The module has
**no third-party dependencies**.
