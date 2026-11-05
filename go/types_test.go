package sfvoice

import (
	"encoding/json"
	"testing"
)

func TestIngestRequestMarshalJSONPreservesLegacyFields(t *testing.T) {
	request := IngestRequest{
		Source:    "url",
		URL:       "https://example.com/video.mp4",
		MediaType: "video",
	}

	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal ingest request: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode ingest request: %v", err)
	}

	if decoded["asset_id"] != compatibilityAssetID(request.URL) {
		t.Fatalf("asset_id = %v, want compatibility id", decoded["asset_id"])
	}
	if decoded["media_kind"] != request.MediaType {
		t.Fatalf("media_kind = %v, want %q", decoded["media_kind"], request.MediaType)
	}
	if _, exists := decoded["media_type"]; exists {
		t.Fatal("legacy media_type must not be sent on the wire")
	}
}

func TestAssetUnmarshalPreservesArbitraryMetadata(t *testing.T) {
	data := []byte(`{
		"asset_id":"asset-1",
		"media_type":"audio",
		"source_type":"upload",
		"types":[],
		"status":"ready",
		"metadata":{"title":"call","score":0.9,"nested":{"source":"crm"}},
		"created_at":"2026-11-05T00:00:00Z",
		"updated_at":"2026-11-05T00:00:00Z"
	}`)

	var asset Asset
	if err := json.Unmarshal(data, &asset); err != nil {
		t.Fatalf("decode asset: %v", err)
	}
	if asset.Metadata["title"] != "call" {
		t.Fatalf("legacy title = %q", asset.Metadata["title"])
	}
	if asset.MetadataValues["score"] != 0.9 {
		t.Fatalf("raw score = %v", asset.MetadataValues["score"])
	}
}
