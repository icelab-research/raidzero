package signoz

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// verifies that the production ratio calculation is bounded and scaled.
func TestUtilizationClampAndRatio(t *testing.T) {
	// 1s of CPU over a 2s window on a 2-core box → 0.25.
	got := utilizationRatio(1, 2, 2)
	if got != 0.25 {
		t.Fatalf("ratio = %v, want 0.25", got)
	}
	if r := utilizationRatio(10, 1, 1); r != 1 {
		t.Fatalf("ratio should clamp to 1, got %v", r)
	}
	if r := utilizationRatio(-1, 1, 1); r != 0 {
		t.Fatalf("ratio should clamp to 0, got %v", r)
	}
	if r := utilizationRatio(1, 0, 1); r != 0 {
		t.Fatalf("ratio should be 0 for an invalid wall interval, got %v", r)
	}
}

// verifies cgroup parsing and process-capacity bounds.
func TestCPUCapacity(t *testing.T) {
	quota, ok := parseCPUMax("50000 100000")
	if !ok || quota != 0.5 {
		t.Fatalf("parseCPUMax = %v, %v; want 0.5, true", quota, ok)
	}
	if _, ok := parseCPUMax("max 100000"); ok {
		t.Fatal("unlimited cpu.max should not produce a quota")
	}
	if _, ok := parseCPUQuota("-1", "100000"); ok {
		t.Fatal("unlimited cgroup v1 quota should not produce a limit")
	}
	if got := normalizedCPUCapacity(8, 4, 1.5, true); got != 1.5 {
		t.Fatalf("normalizedCPUCapacity = %v, want 1.5", got)
	}
	if got := normalizedCPUCapacity(8, 4, 0, false); got != 4 {
		t.Fatalf("normalizedCPUCapacity = %v, want 4", got)
	}
}

// verifies that labels are encoded in a stable order.
func TestPyroscopeNameEncoding(t *testing.T) {
	a := &Agent{cfg: Config{ServiceName: "svc", Labels: map[string]string{"env": "prod", "az": "a"}}}
	if got, want := a.pyroscopeName(), "svc.cpu{az=a,env=prod}"; got != want {
		t.Fatalf("pyroscopeName = %q, want %q (labels must be sorted)", got, want)
	}

	b := &Agent{cfg: Config{ServiceName: "svc"}}
	if got, want := b.pyroscopeName(), "svc.cpu"; got != want {
		t.Fatalf("pyroscopeName = %q, want %q", got, want)
	}
}

// verifies the minimal OTLP JSON representation used by the exporter.
func TestOTLPMetricsJSONShape(t *testing.T) {
	util := 0.42
	count := "7"
	payload := otlpRequest{ResourceMetrics: []otlpResourceMetrics{{
		Resource: otlpResource{Attributes: []otlpKV{stringAttr("service.name", "svc")}},
		ScopeMetrics: []otlpScopeMetrics{{
			Scope: otlpScope{Name: "n", Version: "0.1.0"},
			Metrics: []otlpMetric{
				{Name: "process.cpu.utilization", Unit: "1", Gauge: &otlpGauge{
					DataPoints: []otlpNumberDataPoint{{TimeUnixNano: "1", AsDouble: &util}}}},
				{Name: "process.runtime.go.goroutines", Gauge: &otlpGauge{
					DataPoints: []otlpNumberDataPoint{{TimeUnixNano: "1", AsInt: &count}}}},
			},
		}},
	}}}

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)
	for _, want := range []string{
		`"resourceMetrics"`, `"scopeMetrics"`, `"process.cpu.utilization"`,
		`"asDouble":0.42`, `"asInt":"7"`, `"timeUnixNano":"1"`,
		`"stringValue":"svc"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("OTLP JSON missing %s\nfull: %s", want, out)
		}
	}
	// int64 fields must be JSON strings, not numbers.
	if strings.Contains(out, `"timeUnixNano":1`) {
		t.Error("timeUnixNano must be encoded as a string per proto3 JSON mapping")
	}
}

// verifies that malformed endpoints fail before any goroutine starts.
func TestStartRejectsInvalidEndpoints(t *testing.T) {
	testCases := []struct {
		name     string
		endpoint string
	}{
		{name: "relative", endpoint: "collector:4318"},
		{name: "unsupported scheme", endpoint: "ftp://collector.example.com"},
		{name: "missing host", endpoint: "https:///metrics"},
		{name: "query", endpoint: "https://collector.example.com?token=value"},
		{name: "fragment", endpoint: "https://collector.example.com#metrics"},
		{name: "empty query", endpoint: "https://collector.example.com?"},
		{name: "empty fragment", endpoint: "https://collector.example.com#"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Start(Config{ServiceName: "svc", MetricsEndpoint: testCase.endpoint})
			if err == nil {
				t.Fatalf("Start accepted invalid endpoint %q", testCase.endpoint)
			}
		})
	}
}

// verifies that caller mutations cannot race with the running agent.
func TestStartClonesCallerMaps(t *testing.T) {
	headers := map[string]string{"authorization": "original"}
	labels := map[string]string{"env": "production"}
	agent, err := Start(Config{
		ServiceName:     "svc",
		MetricsEndpoint: "https://collector.example.com",
		MetricsHeaders:  headers,
		MetricsInterval: time.Hour,
		Labels:          labels,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer agent.Stop()

	headers["authorization"] = "changed"
	labels["env"] = "staging"
	if got := agent.cfg.MetricsHeaders["authorization"]; got != "original" {
		t.Fatalf("stored header changed with caller map: %q", got)
	}
	if got := agent.cfg.Labels["env"]; got != "production" {
		t.Fatalf("stored label changed with caller map: %q", got)
	}
}

// verifies that non-CPU metrics remain available when CPU sampling is disabled.
func TestExportMetricsWithoutCPU(t *testing.T) {
	var requestBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var err error
		requestBody, err = io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		response.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	agent := &Agent{
		cfg:    Config{ServiceName: "svc", MetricsEndpoint: server.URL},
		client: server.Client(),
	}
	if err := agent.exportMetrics(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	body := string(requestBody)
	if !strings.Contains(body, `"process.runtime.go.goroutines"`) {
		t.Fatalf("goroutine metric missing from request: %s", body)
	}
	if strings.Contains(body, `"process.cpu.utilization"`) {
		t.Fatalf("CPU metric should be omitted when sampling is unavailable: %s", body)
	}
}
