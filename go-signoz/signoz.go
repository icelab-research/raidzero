// Package signoz is a zero-dependency Go SDK that ships CPU profiles and
// CPU-utilization metrics to SigNoz so you can alert on (and flame-graph)
// high CPU usage.
//
// It runs two independent background loops:
//
//   - Profiler — captures continuous runtime/pprof CPU profiles and pushes
//     them to a Pyroscope-compatible /ingest endpoint (Pyroscope itself, or
//     SigNoz's profiling ingest). This answers "where is the CPU going?".
//   - Metrics — samples process CPU utilization and exports it over OTLP/HTTP
//     as the gauge process.cpu.utilization. This is the timeseries a SigNoz
//     alert rule watches; when it crosses your threshold SigNoz fires the
//     configured Slack notification channel.
//
// The SDK itself never talks to Slack. The high-CPU → Slack message is a
// SigNoz alert rule + Slack notification channel (see README.md). A profile
// (flame graph) cannot be threshold-alerted on; the metric is what makes the
// Slack alert possible, which is why both loops exist.
//
// Quick start:
//
//	agent, err := signoz.Start(signoz.Config{
//	    ServiceName:      "sf-voice-api",
//	    MetricsEndpoint:  "https://ingest.us.signoz.cloud:443",
//	    MetricsHeaders:   map[string]string{"signoz-ingestion-key": os.Getenv("SIGNOZ_KEY")},
//	    ProfilesEndpoint: "http://localhost:4040", // Pyroscope / SigNoz profiling ingest
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer agent.Stop()
package signoz

import (
	"context"
	"fmt"
	"log"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Defaults applied by Start when the corresponding Config field is zero.
const (
	// default metrics interval balances alert latency with export overhead.
	DefaultMetricsInterval = 15 * time.Second
	// default profile interval keeps uploads small while preserving useful samples.
	DefaultProfileInterval = 10 * time.Second
	defaultHTTPTimeout     = 30 * time.Second

	// metricsPath is appended to MetricsEndpoint for OTLP/HTTP metric export.
	metricsPath = "/v1/metrics"
	// profilePath is appended to ProfilesEndpoint for Pyroscope ingest.
	profilePath = "/ingest"
)

// Config configures an Agent. ServiceName is required; at least one of
// MetricsEndpoint or ProfilesEndpoint must be set or Start returns an error.
type Config struct {
	// ServiceName identifies this process in SigNoz (sets the OTLP resource
	// attribute service.name and the Pyroscope application name). Required.
	ServiceName string

	// MetricsEndpoint is the OTLP/HTTP base URL of your SigNoz collector,
	// e.g. "https://ingest.us.signoz.cloud:443" or "http://localhost:4318".
	// Metrics are POSTed as OTLP JSON to MetricsEndpoint + "/v1/metrics".
	// Leave empty to disable the metrics loop.
	MetricsEndpoint string
	// MetricsHeaders are extra headers sent with every metric request, e.g.
	// {"signoz-ingestion-key": "<key>"} for SigNoz Cloud.
	MetricsHeaders map[string]string
	// MetricsInterval is the sample + export period. Defaults to 15s.
	MetricsInterval time.Duration

	// ProfilesEndpoint is the Pyroscope-compatible ingest base URL, e.g.
	// "http://localhost:4040". Profiles are POSTed to ProfilesEndpoint +
	// "/ingest". Leave empty to disable the profiling loop.
	ProfilesEndpoint string
	// ProfilesAuthToken, if set, is sent as "Authorization: Bearer <token>".
	ProfilesAuthToken string
	// ProfileInterval is the length of each CPU profile window before it is
	// uploaded. Defaults to 10s.
	ProfileInterval time.Duration

	// Labels are extra key/value pairs attached to both metric data points
	// (as OTLP attributes) and profiles (as Pyroscope labels), e.g.
	// {"env": "prod", "region": "sfo"}.
	Labels map[string]string

	// HTTPClient is used for all uploads. Defaults to a client with a 30s
	// timeout.
	HTTPClient *http.Client
	// Logger receives non-fatal errors (failed uploads, sampling errors).
	// Defaults to log.Default().
	Logger *log.Logger
}

// Agent owns the running background loops. Create it with Start and shut it
// down with Stop. It is safe to call Stop exactly once.
type Agent struct {
	cfg    Config
	client *http.Client
	logger *log.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Start validates cfg, applies defaults, and launches the enabled background
// loops. It returns immediately; the loops run until Stop is called.
func Start(cfg Config) (*Agent, error) {
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("signoz: ServiceName is required")
	}
	if cfg.MetricsEndpoint == "" && cfg.ProfilesEndpoint == "" {
		return nil, fmt.Errorf("signoz: set MetricsEndpoint and/or ProfilesEndpoint")
	}
	if err := validateEndpoint("MetricsEndpoint", cfg.MetricsEndpoint); err != nil {
		return nil, err
	}
	if err := validateEndpoint("ProfilesEndpoint", cfg.ProfilesEndpoint); err != nil {
		return nil, err
	}
	if cfg.MetricsInterval <= 0 {
		cfg.MetricsInterval = DefaultMetricsInterval
	}
	if cfg.ProfileInterval <= 0 {
		cfg.ProfileInterval = DefaultProfileInterval
	}
	cfg.MetricsHeaders = maps.Clone(cfg.MetricsHeaders)
	cfg.Labels = maps.Clone(cfg.Labels)

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())
	a := &Agent{cfg: cfg, client: client, logger: logger, cancel: cancel}

	if cfg.MetricsEndpoint != "" {
		a.wg.Add(1)
		go func() { defer a.wg.Done(); a.runMetrics(ctx) }()
	}
	if cfg.ProfilesEndpoint != "" {
		a.wg.Add(1)
		go func() { defer a.wg.Done(); a.runProfiler(ctx) }()
	}
	return a, nil
}

// Stop signals the background loops to exit and blocks until they have. It is
// safe to defer immediately after Start.
func (a *Agent) Stop() {
	a.cancel()
	a.wg.Wait()
}

// logf keeps the package prefix consistent for background errors.
func (a *Agent) logf(format string, args ...any) {
	a.logger.Printf("signoz: "+format, args...)
}

// validateEndpoint rejects malformed base URLs before background work starts.
func validateEndpoint(name, rawURL string) error {
	if rawURL == "" {
		return nil
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("signoz: %s is invalid: %w", name, err)
	}
	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return fmt.Errorf("signoz: %s must be an absolute http or https URL", name)
	}
	if strings.ContainsAny(rawURL, "?#") {
		return fmt.Errorf("signoz: %s must not contain a query or fragment", name)
	}
	return nil
}

// sleep blocks for d or until ctx is cancelled. It returns true if the full
// duration elapsed and false if ctx was cancelled first.
func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
