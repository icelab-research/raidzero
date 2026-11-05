package signoz

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"
)

// goCPUSampleRate is the rate runtime/pprof samples the CPU profile at (Hz).
// Go's profiler is fixed at 100 Hz; Pyroscope needs it to scale samples.
const goCPUSampleRate = 100

// runProfiler captures back-to-back CPU profiles of length ProfileInterval and
// uploads each one to the Pyroscope-compatible ingest endpoint. Only one CPU
// profile can be active process-wide, so this loop must own runtime/pprof's
// CPU profiler for the lifetime of the agent.
func (a *Agent) runProfiler(ctx context.Context) {
	for {
		var buf bytes.Buffer
		if err := pprof.StartCPUProfile(&buf); err != nil {
			// Most likely another part of the process already holds the CPU
			// profiler. Back off a window and retry.
			a.logf("start cpu profile: %v", err)
			if !sleep(ctx, a.cfg.ProfileInterval) {
				return
			}
			continue
		}

		from := time.Now()
		completed := sleep(ctx, a.cfg.ProfileInterval)
		pprof.StopCPUProfile()
		until := time.Now()

		if !completed {
			// ctx was cancelled mid-window; drop the partial profile and exit.
			return
		}

		if err := a.uploadProfile(ctx, buf.Bytes(), from, until); err != nil {
			a.logf("profile upload: %v", err)
		}
	}
}

// uploadProfile pushes one gzipped pprof CPU profile to <ProfilesEndpoint>/ingest
// using the Pyroscope pprof ingest protocol: multipart body with a "profile"
// field and metadata carried as query parameters.
func (a *Agent) uploadProfile(ctx context.Context, profile []byte, from, until time.Time) error {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("profile", "profile.pprof")
	if err != nil {
		return fmt.Errorf("multipart: %w", err)
	}
	if _, err := part.Write(profile); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}
	if err := mw.Close(); err != nil {
		return fmt.Errorf("close multipart: %w", err)
	}

	q := url.Values{}
	q.Set("name", a.pyroscopeName())
	q.Set("from", strconv.FormatInt(from.Unix(), 10))
	q.Set("until", strconv.FormatInt(until.Unix(), 10))
	q.Set("format", "pprof")
	q.Set("sampleRate", strconv.Itoa(goCPUSampleRate))
	q.Set("spyName", "gospy")

	endpoint := strings.TrimRight(a.cfg.ProfilesEndpoint, "/") + profilePath + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if a.cfg.ProfilesAuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.cfg.ProfilesAuthToken)
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

// pyroscopeName builds the Pyroscope application name, encoding Config.Labels
// as the {k=v,...} tag suffix Pyroscope expects, e.g. "svc.cpu{env=prod}".
func (a *Agent) pyroscopeName() string {
	name := a.cfg.ServiceName + ".cpu"
	if len(a.cfg.Labels) == 0 {
		return name
	}
	keys := make([]string, 0, len(a.cfg.Labels))
	for k := range a.cfg.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+a.cfg.Labels[k])
	}
	return name + "{" + strings.Join(pairs, ",") + "}"
}
