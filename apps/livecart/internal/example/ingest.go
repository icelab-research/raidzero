// ingest runs concurrent URL ingestion and blocks until all tasks are ready.
package example

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	sfvoice "github.com/icelab-research/raidzero/go"
)

// IngestResult holds the outcome of one ingested URL.
type IngestResult struct {
	URL     string
	AssetID string
	TaskID  string
	Elapsed time.Duration
	Err     error
}

// IngestAll concurrently ingests the provided URLs and returns their per-URL results.
// If concurrency is less than 1 it is treated as 1.
// metadata is attached to every IngestRequest for tagging and later filtering; pass nil to omit.
// The returned slice has the same length and order as urls.
func IngestAll(ctx context.Context, client *sfvoice.Client, urls []string, concurrency int, metadata map[string]any) []IngestResult {
	if concurrency < 1 {
		concurrency = 1
	}

	sem := make(chan struct{}, concurrency)
	results := make([]IngestResult, len(urls))
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, u string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = ingestOne(ctx, client, u, metadata)
		}(i, url)
	}

	wg.Wait()
	return results
}

// ingestOne ingests a single URL and polls until the task reaches a terminal state
// or the 5-minute timeout elapses.
func ingestOne(ctx context.Context, client *sfvoice.Client, url string, metadata map[string]any) IngestResult {
	t0 := time.Now()

	// URL ingestion — for S3 use: sfvoice.IngestRequest{Source: "s3", S3Key: "bucket/key.mp4"}
	resp, err := client.Ingest(ctx, sfvoice.IngestRequest{
		Source:         "url",
		AssetID:        assetIDFromURL(url),
		URL:            url,
		MetadataValues: metadata,
	})
	if err != nil {
		// typed check: *sfvoice.Error carries a machine-readable Code and HTTP Status,
		// which is distinct from a transport-level failure (DNS, timeout, etc.)
		var apiErr *sfvoice.Error
		if errors.As(err, &apiErr) {
			return IngestResult{URL: url, Err: fmt.Errorf("ingest [%s]: %s", apiErr.Code, apiErr.Message)}
		}
		return IngestResult{URL: url, Err: fmt.Errorf("ingest: %w", err)}
	}

	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	task, err := client.PollTask(pollCtx, resp.TaskID, 1500, 300_000)
	if err != nil {
		// *sfvoice.PollTimeoutError means the asset is still indexing — not a hard failure.
		// the asset_id is valid and the task will eventually complete.
		var timeoutErr *sfvoice.PollTimeoutError
		if errors.As(err, &timeoutErr) {
			return IngestResult{
				URL: url, AssetID: resp.AssetID, TaskID: resp.TaskID,
				Err: fmt.Errorf("task %s still indexing after %dms", timeoutErr.TaskID, timeoutErr.TimeoutMs),
			}
		}
		return IngestResult{URL: url, AssetID: resp.AssetID, TaskID: resp.TaskID, Err: fmt.Errorf("poll: %w", err)}
	}
	if task.Status == sfvoice.TaskStatusFailed {
		return IngestResult{URL: url, AssetID: resp.AssetID, TaskID: resp.TaskID, Err: fmt.Errorf("indexing failed: %s", task.Error)}
	}

	return IngestResult{
		URL:     url,
		AssetID: resp.AssetID,
		TaskID:  resp.TaskID,
		Elapsed: time.Since(t0),
	}
}

func assetIDFromURL(url string) string {
	digest := sha256.Sum256([]byte(url))
	return fmt.Sprintf("go-example-%x", digest[:8])
}
