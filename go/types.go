package sfvoice

import (
	"encoding/json"
	"fmt"
)

// ── enums ─────────────────────────────────────────────────────────────────

type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusIndexing TaskStatus = "indexing"
	TaskStatusReady    TaskStatus = "ready"
	TaskStatusFailed   TaskStatus = "failed"
)

func (s TaskStatus) IsTerminal() bool {
	return s == TaskStatusReady || s == TaskStatusFailed
}

// ── pagination ────────────────────────────────────────────────────────────

type PageInfo struct {
	Total         int64  `json:"total"`
	Page          int    `json:"page"`
	Limit         int    `json:"limit"`
	NextPageToken string `json:"next_page_token,omitempty"`
}

// ── asset ─────────────────────────────────────────────────────────────────

type Asset struct {
	AssetID         string            `json:"asset_id"`
	ID              string            `json:"-"` // deprecated: use AssetID
	Filename        string            `json:"filename,omitempty"`
	ProjectID       string            `json:"project_id,omitempty"`
	AssetClass      string            `json:"asset_class,omitempty"`
	MediaType       string            `json:"media_type"`
	SourceType      string            `json:"source_type"`
	Types           []string          `json:"types"`
	Status          TaskStatus        `json:"status"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	MetadataValues  map[string]any    `json:"-"`
	DurationMs      *int64            `json:"duration_ms,omitempty"`
	ThumbnailURL    string            `json:"thumbnail_url,omitempty"`
	ThumbnailStatus string            `json:"thumbnail_status,omitempty"`
	CreatedAt       string            `json:"created_at"`
	UpdatedAt       string            `json:"updated_at"`
}

func (asset *Asset) UnmarshalJSON(data []byte) error {
	type assetAlias Asset
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	metadata := fields["metadata"]
	delete(fields, "metadata")
	withoutMetadata, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	var decoded assetAlias
	if err := json.Unmarshal(withoutMetadata, &decoded); err != nil {
		return err
	}
	*asset = Asset(decoded)
	asset.ID = asset.AssetID
	if len(metadata) > 0 && string(metadata) != "null" {
		if err := json.Unmarshal(metadata, &asset.MetadataValues); err != nil {
			return err
		}
		asset.Metadata = make(map[string]string)
		for key, value := range asset.MetadataValues {
			if text, ok := value.(string); ok {
				asset.Metadata[key] = text
			}
		}
	}
	return nil
}

type AssetListResponse struct {
	Items    []Asset  `json:"items"`
	PageInfo PageInfo `json:"page_info"`
}

// ── ingest ────────────────────────────────────────────────────────────────

type IngestRequest struct {
	Source         string            `json:"source"` // "url" | "s3"
	AssetID        string            `json:"asset_id"`
	AssetClass     string            `json:"asset_class,omitempty"`
	ProjectID      string            `json:"project_id,omitempty"`
	URL            string            `json:"url,omitempty"`
	S3Key          string            `json:"s3_key,omitempty"`
	MediaKind      string            `json:"media_kind,omitempty"` // "video" | "audio"
	MediaType      string            `json:"-"`                    // deprecated: use MediaKind
	Metadata       map[string]string `json:"metadata,omitempty"`
	MetadataValues map[string]any    `json:"-"`
	Prefix         string            `json:"prefix,omitempty"`
	TranscriptOnly bool              `json:"transcript_only,omitempty"`
}

func (request IngestRequest) MarshalJSON() ([]byte, error) {
	type ingestRequestAlias IngestRequest
	payload := struct {
		ingestRequestAlias
		MediaKind string `json:"media_kind,omitempty"`
	}{
		ingestRequestAlias: ingestRequestAlias(request),
		MediaKind:          request.MediaKind,
	}
	if payload.MediaKind == "" {
		payload.MediaKind = request.MediaType
	}
	if payload.AssetID == "" {
		if request.URL != "" {
			payload.AssetID = compatibilityAssetID(request.URL)
		} else {
			payload.AssetID = compatibilityAssetID(request.S3Key)
		}
	}
	encoded, err := json.Marshal(payload)
	if err != nil || request.MetadataValues == nil {
		return encoded, err
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	fields["metadata"] = request.MetadataValues
	return json.Marshal(fields)
}

func compatibilityAssetID(value string) string {
	hash := uint64(0xcbf29ce484222325)
	for _, valueByte := range []byte(value) {
		hash ^= uint64(valueByte)
		hash *= 0x100000001b3
	}
	return fmt.Sprintf("legacy-%016x", hash)
}

type IngestResponse struct {
	AssetID string     `json:"asset_id"`
	TaskID  string     `json:"task_id"`
	Status  TaskStatus `json:"status"`
}

// ── tasks ─────────────────────────────────────────────────────────────────

type Task struct {
	TaskID      string     `json:"task_id"`
	AssetID     string     `json:"asset_id"`
	AssetClass  string     `json:"asset_class,omitempty"`
	Types       []string   `json:"types"`
	Status      TaskStatus `json:"status"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   string     `json:"created_at"`
	CompletedAt string     `json:"completed_at,omitempty"`
}

// ── search ────────────────────────────────────────────────────────────────

type SearchRequest struct {
	Query      string   `json:"query"`
	Types      []string `json:"types,omitempty"` // "visual" | "conversation" | "text_in_video"
	AssetIDs   []string `json:"asset_ids,omitempty"`
	AssetClass string   `json:"asset_class,omitempty"`
	ProjectID  string   `json:"project_id,omitempty"`
	Prefix     string   `json:"prefix,omitempty"`
	Threshold  float64  `json:"threshold,omitempty"`
	Page       int      `json:"page,omitempty"`
	Limit      int      `json:"limit,omitempty"`
}

type SearchResult struct {
	DocumentID        string  `json:"document_id"`
	AssetID           string  `json:"asset_id,omitempty"`
	AssetClass        string  `json:"asset_class,omitempty"`
	TwelveLabsMediaID string  `json:"twelvelabs_media_id,omitempty"`
	Score             float64 `json:"score"`
	StartMs           int64   `json:"start_ms"`
	EndMs             int64   `json:"end_ms"`
	MatchType         string  `json:"match_type"`
	Text              string  `json:"text,omitempty"`
	ThumbnailURL      string  `json:"thumbnail_url,omitempty"`
}

type SearchResponse struct {
	SearchQueryID string         `json:"search_query_id"`
	Results       []SearchResult `json:"results"`
	PageInfo      PageInfo       `json:"page_info"`
}

type Prefix struct {
	Prefix        string `json:"prefix"`
	OwnedByCaller bool   `json:"owned_by_caller"`
}

type PrefixListResponse struct {
	Items []Prefix `json:"items"`
}

// ── monitor ──────────────────────────────────────────────────────────────

type Monitor struct {
	ID         string  `json:"id"`
	Slug       string  `json:"slug"`
	Text       string  `json:"text"`
	ProjectID  string  `json:"project_id,omitempty"`
	AssetClass string  `json:"asset_class,omitempty"`
	Threshold  float64 `json:"threshold"`
	Enabled    bool    `json:"enabled"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type MonitorListResponse struct {
	Items []Monitor `json:"items"`
	Total int64     `json:"total"`
}

type CreateMonitorRequest struct {
	Text       string   `json:"text"`
	Slug       string   `json:"slug,omitempty"`
	ProjectID  string   `json:"project_id,omitempty"`
	AssetClass string   `json:"asset_class,omitempty"`
	Threshold  *float64 `json:"threshold,omitempty"`
}

type UpdateMonitorRequest struct {
	Text       *string  `json:"text,omitempty"`
	AssetClass *string  `json:"asset_class,omitempty"`
	Threshold  *float64 `json:"threshold,omitempty"`
	Enabled    *bool    `json:"enabled,omitempty"`
}

type MonitorEvent struct {
	ID          string          `json:"id"`
	MonitorID   string          `json:"monitor_id"`
	DocumentID  string          `json:"document_id"`
	AssetID     string          `json:"asset_id,omitempty"`
	Matched     bool            `json:"matched"`
	Score       *float64        `json:"score,omitempty"`
	WebhookSent bool            `json:"webhook_sent"`
	MatchDetail json.RawMessage `json:"match_detail,omitempty"`
	CreatedAt   string          `json:"created_at"`
}

type MonitorEventListResponse struct {
	Items []MonitorEvent `json:"items"`
	Total int64          `json:"total"`
}

// AlertOptions configures the Alert convenience method.
type AlertOptions struct {
	Slug       string   `json:"slug,omitempty"`
	ProjectID  string   `json:"project_id,omitempty"`
	AssetClass string   `json:"asset_class,omitempty"`
	Threshold  *float64 `json:"threshold,omitempty"`
	// IntervalMs is the polling interval in milliseconds. Defaults to 5000 if zero.
	IntervalMs int64 `json:"-"`
}

// ── internal ──────────────────────────────────────────────────────────────

type apiErrorEnvelope struct {
	Error apiErrorBody `json:"error"`
}

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
