// search runs a query and prints formatted results.
package example

import (
	"context"
	"fmt"
	"strings"

	sfvoice "github.com/icelab-research/raidzero/go"
)

// Search queries the sfvoice API and prints a formatted result list to stdout.
// assetIDs and types are optional filters.
// threshold (0–1) filters by minimum score; pass 0 to use the server default.
// page selects the result page (1-based).
func Search(ctx context.Context, client *sfvoice.Client, query string, assetIDs []string, types []string, threshold float64, page int) error {
	req := sfvoice.SearchRequest{
		Query:     query,
		Threshold: threshold,
		Page:      page,
	}
	if len(assetIDs) > 0 {
		req.AssetIDs = assetIDs
	}
	if len(types) > 0 {
		req.Types = types
	}

	resp, err := client.Search(ctx, req)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	fmt.Printf("\U0001f50d results for %q  threshold=%.2f  page=%d  (%d total):\n",
		query, threshold, resp.PageInfo.Page, resp.PageInfo.Total)
	if len(resp.Results) == 0 {
		fmt.Println("  (no results)")
		return nil
	}

	for i, r := range resp.Results {
		start := msToTime(uint64(r.StartMs))
		end := msToTime(uint64(r.EndMs))
		matchType := strings.ReplaceAll(r.MatchType, "_", " ")
		line := fmt.Sprintf("  %d. score=%.2f  %s–%s  %-20s  asset=%s",
			i+1, r.Score, start, end, matchType, r.AssetID)
		if r.ThumbnailURL != "" {
			line += "  thumbnail=" + r.ThumbnailURL
		}
		fmt.Println(line)
	}

	if resp.PageInfo.NextPageToken != "" {
		fmt.Printf("  … more results (next_page_token=%s)\n", resp.PageInfo.NextPageToken)
	}

	return nil
}

// msToTime converts a duration given in milliseconds to a string in "M:SS" format.
func msToTime(ms uint64) string {
	s := ms / 1000
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
