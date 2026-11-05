// sf-voice media SDK demo — concurrent ingest + search + asset management.
//
// usage:
//
//	go run ./cmd/example -urls "https://a.mp4,https://b.mp3" -query "product launch"
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/icelab-research/raidzero/apps/livecart/config"
	"github.com/icelab-research/raidzero/apps/livecart/internal/example"
	sfvoice "github.com/icelab-research/raidzero/go"
)

// main parses command-line flags, loads configuration, ingests one or more media URLs
// concurrently, runs a search query, then demonstrates asset management.
//
// Flags:
//
//	-urls         comma-separated media URLs to ingest (falls back to SAMPLE_MEDIA_URL)
//	-query        required search query to run after ingest
//	-types        optional comma-separated match types: visual, conversation, text_in_video
//	-threshold    minimum search score 0–1; 0 uses the server default
//	-page         search result page, 1-based (default 1)
//	-concurrency  max concurrent ingests (default 4)
//	-delete       soft-delete all freshly-ingested assets after the demo
//
// Exit codes:
//
//	1 on configuration load failure, ingest failure, or asset management error
//	2 when required inputs (urls or query) are missing
func main() {
	urlsFlag := flag.String("urls", "", "comma-separated media URLs to ingest")
	queryFlag := flag.String("query", "", "search query to run after ingest")
	typesFlag := flag.String("types", "", "comma-separated match types: visual,conversation,text_in_video")
	thresholdFlag := flag.Float64("threshold", 0, "minimum search score 0–1 (0 = server default)")
	pageFlag := flag.Int("page", 1, "search result page (1-based)")
	concFlag := flag.Int("concurrency", 4, "max concurrent ingests")
	deleteFlag := flag.Bool("delete", false, "soft-delete ingested assets after the demo")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	rawURLs := strings.TrimSpace(*urlsFlag)
	if rawURLs == "" {
		rawURLs = cfg.SampleMediaURL
	}
	if rawURLs == "" {
		fmt.Fprintln(os.Stderr, "error: provide -urls or set SAMPLE_MEDIA_URL")
		os.Exit(2)
	}

	query := strings.TrimSpace(*queryFlag)
	if query == "" {
		fmt.Fprintln(os.Stderr, "error: -query is required")
		os.Exit(2)
	}

	urls := strings.Split(rawURLs, ",")
	for i, u := range urls {
		urls[i] = strings.TrimSpace(u)
	}

	var types []string
	if *typesFlag != "" {
		for _, t := range strings.Split(*typesFlag, ",") {
			types = append(types, strings.TrimSpace(t))
		}
	}

	// NewWithHTTPClient lets you inject a custom transport for retries or proxies.
	// For most cases sfvoice.New(baseURL, apiKey) is sufficient.
	_ = sfvoice.NewWithHTTPClient // documented; available when you need it
	client := sfvoice.New(cfg.BaseURL, cfg.APIKey)
	ctx := context.Background()

	// metadata is attached to every ingest request — useful for attribution and filtering later.
	metadata := map[string]any{"demo": "go-example"}

	fmt.Printf("ingesting %d URL(s) with concurrency=%d …\n", len(urls), *concFlag)
	results := example.IngestAll(ctx, client, urls, *concFlag, metadata)

	var assetIDs []string
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", r.URL, r.Err)
		} else {
			fmt.Printf("  ✓ asset_id=%-30s  %.1fs  %s\n", r.AssetID, r.Elapsed.Seconds(), r.URL)
			assetIDs = append(assetIDs, r.AssetID)
		}
	}

	if len(assetIDs) == 0 {
		fmt.Fprintln(os.Stderr, "no assets ingested successfully")
		os.Exit(1)
	}

	fmt.Println()
	if err := example.Search(ctx, client, query, assetIDs, types, *thresholdFlag, *pageFlag); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Println()
	var deleteIDs []string
	if *deleteFlag {
		deleteIDs = assetIDs
	}
	// spot-fetch the first successfully ingested asset to demonstrate GetAsset
	if err := example.ManageAssets(ctx, client, assetIDs[0], deleteIDs); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
