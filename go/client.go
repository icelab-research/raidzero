// Package sfvoice provides a client for the sf-voice media API.
//
// Quick start:
//
//	client := sfvoice.New("https://api.sf-voice.com", os.Getenv("SF_VOICE_API_KEY"))
//
//	resp, err := client.Ingest(ctx, sfvoice.IngestRequest{
//	    Source: "url",
//	    URL:    "https://example.com/call.mp4",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.TaskID)
package sfvoice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Client is the sf-voice media API client.
// It is safe for concurrent use; create one and reuse it.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New creates a client.
//
// baseURL should not have a trailing slash (e.g. "https://api.sf-voice.com").
// apiKey is sent as a bearer token on every request.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewWithHTTPClient creates a client using a custom *http.Client.
// Useful for setting custom timeouts, transports, or test doubles.
func NewWithHTTPClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, httpClient: httpClient}
}

// ── internal helpers ──────────────────────────────────────────────────────

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("sfvoice: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("sfvoice: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sfvoice: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("sfvoice: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var env apiErrorEnvelope
		code, message := "http_error", fmt.Sprintf("request failed with status %d", resp.StatusCode)
		if json.Unmarshal(respBody, &env) == nil && env.Error.Code != "" {
			code = env.Error.Code
			message = env.Error.Message
		}
		return &Error{Code: code, Message: message, Status: resp.StatusCode}
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("sfvoice: decode response: %w", err)
		}
	}
	return nil
}

// ── public API ────────────────────────────────────────────────────────────

// Ingest submits a media file for ingestion from a URL or S3 key.
// It returns immediately with a TaskID you can poll with GetTask or PollTask.
func (c *Client) Ingest(ctx context.Context, req IngestRequest) (IngestResponse, error) {
	var out IngestResponse
	return out, c.do(ctx, http.MethodPost, "/api/v1/ingest", req, &out)
}

// GetTask fetches the current state of an ingestion task.
func (c *Client) GetTask(ctx context.Context, taskID string) (Task, error) {
	var out Task
	return out, c.do(ctx, http.MethodGet, "/api/v1/tasks/"+url.PathEscape(taskID), nil, &out)
}

// ListAssets returns a paginated list of assets. Pass 0 for page or limit to use server defaults.
func (c *Client) ListAssets(ctx context.Context, page, limit int) (AssetListResponse, error) {
	path := "/api/v1/assets"
	if page > 0 || limit > 0 {
		q := url.Values{}
		if page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if limit > 0 {
			q.Set("limit", strconv.Itoa(limit))
		}
		path += "?" + q.Encode()
	}
	var out AssetListResponse
	return out, c.do(ctx, http.MethodGet, path, nil, &out)
}

// GetAsset fetches a single asset by ID.
func (c *Client) GetAsset(ctx context.Context, assetID string) (Asset, error) {
	var out Asset
	return out, c.do(ctx, http.MethodGet, "/api/v1/assets/"+url.PathEscape(assetID), nil, &out)
}

// DeleteAsset soft-deletes an asset. The backend retains the record but
// excludes it from list results. Returns nil on HTTP 204.
func (c *Client) DeleteAsset(ctx context.Context, assetID string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/assets/"+url.PathEscape(assetID), nil, nil)
}

// Search runs a semantic search across indexed media.
func (c *Client) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	var out SearchResponse
	return out, c.do(ctx, http.MethodPost, "/api/v1/search", req, &out)
}

// ListPrefixes lists native transcript prefixes visible to this API key.
func (c *Client) ListPrefixes(ctx context.Context) (PrefixListResponse, error) {
	var out PrefixListResponse
	return out, c.do(ctx, http.MethodGet, "/api/v1/prefixes", nil, &out)
}

// PollTask polls GetTask at intervalMs intervals until the task reaches a terminal
// state (ready or failed), then returns the final Task.
//
// Returns *PollTimeoutError if timeoutMs elapses before a terminal state is reached.
func (c *Client) PollTask(ctx context.Context, taskID string, intervalMs, timeoutMs int64) (Task, error) {
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	interval := time.Duration(intervalMs) * time.Millisecond

	for {
		task, err := c.GetTask(ctx, taskID)
		if err != nil {
			return Task{}, err
		}
		if task.Status.IsTerminal() {
			return task, nil
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return Task{}, &PollTimeoutError{TaskID: taskID, TimeoutMs: timeoutMs}
		}

		sleep := interval
		if sleep > remaining {
			sleep = remaining
		}

		select {
		case <-ctx.Done():
			return Task{}, ctx.Err()
		case <-time.After(sleep):
		}
	}
}

// ── monitors ─────────────────────────────────────────────────────────────

// CreateMonitor creates a new monitor.
func (c *Client) CreateMonitor(ctx context.Context, req CreateMonitorRequest) (Monitor, error) {
	var out Monitor
	return out, c.do(ctx, http.MethodPost, "/api/v1/monitors", req, &out)
}

// ListMonitors returns all monitors for the current API key.
func (c *Client) ListMonitors(ctx context.Context) (MonitorListResponse, error) {
	var out MonitorListResponse
	return out, c.do(ctx, http.MethodGet, "/api/v1/monitors", nil, &out)
}

// GetMonitor fetches a single monitor by ID.
func (c *Client) GetMonitor(ctx context.Context, monitorID string) (Monitor, error) {
	var out Monitor
	return out, c.do(ctx, http.MethodGet, "/api/v1/monitors/"+url.PathEscape(monitorID), nil, &out)
}

// UpdateMonitor patches an existing monitor.
func (c *Client) UpdateMonitor(ctx context.Context, monitorID string, req UpdateMonitorRequest) (Monitor, error) {
	var out Monitor
	return out, c.do(ctx, http.MethodPatch, "/api/v1/monitors/"+url.PathEscape(monitorID), req, &out)
}

// DeleteMonitor deletes a monitor by ID. Returns nil on success.
func (c *Client) DeleteMonitor(ctx context.Context, monitorID string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/monitors/"+url.PathEscape(monitorID), nil, nil)
}

// ListMonitorEvents returns a paginated list of events for a monitor.
// Set matchedOnly to true to only return matched events.
func (c *Client) ListMonitorEvents(ctx context.Context, monitorID string, matchedOnly bool, limit, offset int) (MonitorEventListResponse, error) {
	q := url.Values{}
	if matchedOnly {
		q.Set("matched_only", "true")
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	path := "/api/v1/monitors/" + url.PathEscape(monitorID) + "/events"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out MonitorEventListResponse
	return out, c.do(ctx, http.MethodGet, path, nil, &out)
}

// ── alert (high-level convenience) ───────────────────────────────────────

// AlertHandle is returned by Alert and lets the caller stop polling
// and clean up the underlying monitor.
type AlertHandle struct {
	MonitorID     string
	cancel        context.CancelFunc
	done          chan struct{}
	client        *Client
	backgroundErr chan error
	stopOnce      sync.Once
	stopErr       error
}

// Stop cancels the polling goroutine, waits for it to exit, and
// deletes the monitor that was created by Alert and returns cleanup failures.
func (h *AlertHandle) Stop() error {
	h.stopOnce.Do(func() {
		h.cancel()
		<-h.done
		var pollErr error
		select {
		case pollErr = <-h.backgroundErr:
		default:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.stopErr = errors.Join(pollErr, h.client.DeleteMonitor(ctx, h.MonitorID))
	})
	return h.stopErr
}

// Alert creates a monitor for text and polls for matched events,
// calling callback on each new match. The polling goroutine runs until
// the returned AlertHandle is stopped or the context is cancelled.
func (c *Client) Alert(ctx context.Context, text string, callback func(MonitorEvent), opts AlertOptions) (*AlertHandle, error) {
	req := CreateMonitorRequest{
		Text:       text,
		Slug:       opts.Slug,
		ProjectID:  opts.ProjectID,
		AssetClass: opts.AssetClass,
		Threshold:  opts.Threshold,
	}

	mon, err := c.CreateMonitor(ctx, req)
	if err != nil {
		return nil, err
	}

	intervalMs := opts.IntervalMs
	if intervalMs <= 0 {
		intervalMs = 5000
	}
	interval := time.Duration(intervalMs) * time.Millisecond

	childCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	backgroundErr := make(chan error, 1)

	handle := &AlertHandle{
		MonitorID:     mon.ID,
		cancel:        cancel,
		done:          done,
		client:        c,
		backgroundErr: backgroundErr,
	}

	go func() {
		defer close(done)
		var checkpoint string
		consecutiveErrors := 0

		for {
			pending, nextCheckpoint, err := c.collectNewMonitorEvents(childCtx, mon.ID, checkpoint)
			if err == nil {
				consecutiveErrors = 0
				for index := len(pending) - 1; index >= 0; index-- {
					select {
					case <-childCtx.Done():
						return
					default:
					}
					if callbackErr := callAlertCallback(callback, pending[index]); callbackErr != nil {
						backgroundErr <- callbackErr
						return
					}
				}
				if nextCheckpoint != "" {
					checkpoint = nextCheckpoint
				}
			} else if childCtx.Err() != nil {
				return
			} else {
				consecutiveErrors++
				if !isRetryableAlertError(err) || consecutiveErrors >= 3 {
					backgroundErr <- err
					return
				}
			}

			select {
			case <-childCtx.Done():
				return
			case <-time.After(interval):
			}
		}
	}()

	return handle, nil
}

func (c *Client) collectNewMonitorEvents(ctx context.Context, monitorID, checkpoint string) ([]MonitorEvent, string, error) {
	const limit = 100
	var pending []MonitorEvent
	seen := make(map[string]struct{})
	nextCheckpoint := ""

	for offset, page := 0, 0; page < 1000; page++ {
		response, err := c.ListMonitorEvents(ctx, monitorID, true, limit, offset)
		if err != nil {
			return nil, "", err
		}
		if nextCheckpoint == "" && len(response.Items) > 0 {
			nextCheckpoint = response.Items[0].ID
		}

		for _, event := range response.Items {
			if event.ID == checkpoint {
				return pending, nextCheckpoint, nil
			}
			if _, exists := seen[event.ID]; !exists {
				seen[event.ID] = struct{}{}
				pending = append(pending, event)
			}
		}

		offset += len(response.Items)
		if len(response.Items) < limit || int64(offset) >= response.Total {
			return pending, nextCheckpoint, nil
		}
	}

	return nil, "", fmt.Errorf("alert event pagination exceeded 1000 pages")
}

func isRetryableAlertError(err error) bool {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		return true
	}
	return apiErr.Status == http.StatusTooManyRequests || apiErr.Status >= http.StatusInternalServerError
}

func callAlertCallback(callback func(MonitorEvent), event MonitorEvent) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("alert callback panicked: %v", recovered)
		}
	}()
	callback(event)
	return nil
}
