package signoz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ── CPU sampling ───────────────────────────────────────────────────────────

// cpuSampler turns the process's cumulative CPU time into a utilization ratio.
type cpuSampler struct {
	lastCPU  time.Duration
	lastWall time.Time
	numCPU   float64
}

// newCPUSampler captures a valid baseline before utilization is calculated.
func newCPUSampler() (*cpuSampler, error) {
	cpu, err := processCPUTime()
	if err != nil {
		return nil, err
	}
	return &cpuSampler{
		lastCPU:  cpu,
		lastWall: time.Now(),
		numCPU:   availableCPUCapacity(),
	}, nil
}

// utilization returns CPU utilization in [0,1] since the previous call, where
// 1.0 means every available core was fully saturated. This matches the
// OpenTelemetry semantics for process.cpu.utilization: ΔCPU / (Δwall · nCPU).
func (s *cpuSampler) utilization() (float64, error) {
	cpu, err := processCPUTime()
	if err != nil {
		return 0, err
	}
	now := time.Now()
	dCPU := (cpu - s.lastCPU).Seconds()
	dWall := now.Sub(s.lastWall).Seconds()
	s.lastCPU, s.lastWall = cpu, now

	return utilizationRatio(dCPU, dWall, s.numCPU), nil
}

// utilizationRatio converts CPU and wall deltas into a bounded ratio.
func utilizationRatio(cpuSeconds, wallSeconds, numCPU float64) float64 {
	if wallSeconds <= 0 || numCPU <= 0 {
		return 0
	}
	utilization := cpuSeconds / (wallSeconds * numCPU)
	if utilization < 0 {
		return 0
	}
	if utilization > 1 {
		return 1
	}
	return utilization
}

// ── metrics loop ───────────────────────────────────────────────────────────

// runMetrics exports available metrics until the agent is stopped.
func (a *Agent) runMetrics(ctx context.Context) {
	sampler, sampleErr := newCPUSampler()
	cpuSamplingEnabled := sampleErr == nil
	if sampleErr != nil {
		a.logf("cpu sampling disabled: %v", sampleErr)
	}
	ticker := time.NewTicker(a.cfg.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var utilization *float64
			if cpuSamplingEnabled {
				value, err := sampler.utilization()
				if err != nil {
					a.logf("cpu sampling disabled: %v", err)
					cpuSamplingEnabled = false
				} else {
					utilization = &value
				}
			}
			if err := a.exportMetrics(ctx, utilization); err != nil {
				a.logf("metrics export: %v", err)
			}
		}
	}
}

// exportMetrics POSTs the current CPU utilization (plus goroutine count) to the
// SigNoz OTLP/HTTP endpoint as an OTLP JSON payload.
func (a *Agent) exportMetrics(ctx context.Context, utilization *float64) error {
	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	attrs := a.metricAttributes()
	goroutines := strconv.Itoa(runtime.NumGoroutine())
	metrics := []otlpMetric{{
		Name:        "process.runtime.go.goroutines",
		Unit:        "{goroutine}",
		Description: "Number of goroutines currently running.",
		Gauge: &otlpGauge{DataPoints: []otlpNumberDataPoint{{
			Attributes:   attrs,
			TimeUnixNano: now,
			AsInt:        &goroutines,
		}}},
	}}
	if utilization != nil {
		metrics = append([]otlpMetric{{
			Name:        "process.cpu.utilization",
			Unit:        "1",
			Description: "Process CPU utilization in [0,1], 1.0 = all cores saturated.",
			Gauge: &otlpGauge{DataPoints: []otlpNumberDataPoint{{
				Attributes:   attrs,
				TimeUnixNano: now,
				AsDouble:     utilization,
			}}},
		}}, metrics...)
	}

	payload := otlpRequest{ResourceMetrics: []otlpResourceMetrics{{
		Resource: otlpResource{Attributes: a.resourceAttributes()},
		ScopeMetrics: []otlpScopeMetrics{{
			Scope:   otlpScope{Name: "github.com/icelab-research/raidzero/go-signoz", Version: "0.1.0"},
			Metrics: metrics,
		}},
	}}}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal otlp: %w", err)
	}

	url := strings.TrimRight(a.cfg.MetricsEndpoint, "/") + metricsPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range a.cfg.MetricsHeaders {
		req.Header.Set(k, v)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return nil
}

// resourceAttributes returns the OTLP resource attributes (service identity).
func (a *Agent) resourceAttributes() []otlpKV {
	attrs := []otlpKV{stringAttr("service.name", a.cfg.ServiceName)}
	if host, err := os.Hostname(); err == nil && host != "" {
		attrs = append(attrs, stringAttr("host.name", host))
	}
	return attrs
}

// metricAttributes returns the per-data-point attributes from Config.Labels,
// sorted for deterministic output.
func (a *Agent) metricAttributes() []otlpKV {
	if len(a.cfg.Labels) == 0 {
		return nil
	}
	keys := make([]string, 0, len(a.cfg.Labels))
	for k := range a.cfg.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	attrs := make([]otlpKV, 0, len(keys))
	for _, k := range keys {
		attrs = append(attrs, stringAttr(k, a.cfg.Labels[k]))
	}
	return attrs
}

// ── OTLP JSON wire types ───────────────────────────────────────────────────
//
// Minimal subset of the OTLP metrics JSON schema (proto3 JSON mapping): int64
// fields such as timeUnixNano and asInt are encoded as strings.

type otlpRequest struct {
	ResourceMetrics []otlpResourceMetrics `json:"resourceMetrics"`
}

type otlpResourceMetrics struct {
	Resource     otlpResource       `json:"resource"`
	ScopeMetrics []otlpScopeMetrics `json:"scopeMetrics"`
}

type otlpResource struct {
	Attributes []otlpKV `json:"attributes,omitempty"`
}

type otlpScopeMetrics struct {
	Scope   otlpScope    `json:"scope"`
	Metrics []otlpMetric `json:"metrics"`
}

type otlpScope struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type otlpMetric struct {
	Name        string     `json:"name"`
	Unit        string     `json:"unit,omitempty"`
	Description string     `json:"description,omitempty"`
	Gauge       *otlpGauge `json:"gauge,omitempty"`
}

type otlpGauge struct {
	DataPoints []otlpNumberDataPoint `json:"dataPoints"`
}

type otlpNumberDataPoint struct {
	Attributes   []otlpKV `json:"attributes,omitempty"`
	TimeUnixNano string   `json:"timeUnixNano"`
	AsDouble     *float64 `json:"asDouble,omitempty"`
	AsInt        *string  `json:"asInt,omitempty"`
}

type otlpKV struct {
	Key   string   `json:"key"`
	Value otlpAnyV `json:"value"`
}

type otlpAnyV struct {
	StringValue string `json:"stringValue"`
}

// stringAttr builds the OTLP representation of a string attribute.
func stringAttr(k, v string) otlpKV {
	return otlpKV{Key: k, Value: otlpAnyV{StringValue: v}}
}
