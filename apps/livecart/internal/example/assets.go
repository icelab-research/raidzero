// assets demonstrates ListAssets, GetAsset, and DeleteAsset.
package example

import (
	"context"
	"errors"
	"fmt"

	sfvoice "github.com/icelab-research/raidzero/go"
)

// ManageAssets demonstrates the three asset management methods:
//   - ListAssets: browse indexed assets with pagination
//   - GetAsset: fetch full detail for one asset, with typed 404 handling
//   - DeleteAsset: soft-delete assets by ID (pass nil deleteIDs to skip)
func ManageAssets(ctx context.Context, client *sfvoice.Client, spotAssetID string, deleteIDs []string) error {
	fmt.Println("── asset list (page 1, limit 5) ──────────────────────────────")
	list, err := client.ListAssets(ctx, 1, 5)
	if err != nil {
		return fmt.Errorf("list assets: %w", err)
	}
	fmt.Printf("  total=%d  page=%d  limit=%d\n",
		list.PageInfo.Total, list.PageInfo.Page, list.PageInfo.Limit)
	for _, a := range list.Items {
		dur := "–"
		if a.DurationMs != nil {
			dur = fmt.Sprintf("%.1fs", float64(*a.DurationMs)/1000)
		}
		fmt.Printf("  • %-36s  %-10s  %s\n", a.AssetID, a.Status, dur)
	}

	if spotAssetID != "" {
		fmt.Println("\n── get asset ─────────────────────────────────────────────────")
		asset, err := client.GetAsset(ctx, spotAssetID)
		if err != nil {
			// typed check: a 404 is a known, recoverable state (asset was deleted or
			// never existed). anything else (auth, network) is an unexpected failure.
			var apiErr *sfvoice.Error
			if errors.As(err, &apiErr) && apiErr.Status == 404 {
				fmt.Printf("  asset %s not found (already deleted?)\n", spotAssetID)
			} else {
				return fmt.Errorf("get asset: %w", err)
			}
		} else {
			fmt.Printf("  id          %s\n", asset.AssetID)
			fmt.Printf("  media_type  %s\n", asset.MediaType)
			fmt.Printf("  source      %s\n", asset.SourceType)
			fmt.Printf("  status      %s\n", asset.Status)
			if asset.DurationMs != nil {
				fmt.Printf("  duration    %.1fs\n", float64(*asset.DurationMs)/1000)
			}
			for k, v := range asset.Metadata {
				fmt.Printf("  meta.%-10s %s\n", k, v)
			}
		}
	}

	if len(deleteIDs) > 0 {
		fmt.Println("\n── delete assets ─────────────────────────────────────────────")
		for _, id := range deleteIDs {
			if err := client.DeleteAsset(ctx, id); err != nil {
				var apiErr *sfvoice.Error
				if errors.As(err, &apiErr) {
					fmt.Printf("  ✗ %s: [HTTP %d] %s\n", id, apiErr.Status, apiErr.Message)
				} else {
					fmt.Printf("  ✗ %s: %v\n", id, err)
				}
			} else {
				fmt.Printf("  ✓ deleted %s\n", id)
			}
		}
	}

	return nil
}
