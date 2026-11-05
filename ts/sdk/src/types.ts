/**
 * all public types for the @sf-voice/media SDK.
 * mirrors the shapes returned by the sf-voice media API.
 */

// ─── shared ──────────────────────────────────────────────────────────────────

export type MediaType = "video" | "audio";
export type SourceType = "url" | "s3" | "file";
export type AssetSourceType = SourceType | "upload" | "youtube" | "twitter";
export type TaskStatus = "pending" | "indexing" | "ready" | "failed";
export type MediaSearchType = "video" | "audio" | "transcript";
export type SearchMatchType =
  MediaSearchType | "visual" | "conversation" | "text_in_video";

export type MediaMetadata = Record<string, string | number | boolean>;
export type IngestFile = Blob | ArrayBuffer | Uint8Array;

/** a single asset in the library */
export type Asset = {
  asset_id: string;
  filename?: string;
  project_id?: string;
  asset_class?: string;
  media_type: MediaType;
  source_type: AssetSourceType;
  types: MediaSearchType[];
  status: TaskStatus;
  metadata?: MediaMetadata;
  duration_ms?: number;
  thumbnail_url?: string;
  thumbnail_status?: "pending" | "ready" | "not_applicable";
  created_at: string;
  updated_at: string;
};

// ─── pagination ──────────────────────────────────────────────────────────────

export type PageInfo = {
  total: number;
  page: number;
  limit: number;
  next_page_token?: string;
};

// ─── ingest ──────────────────────────────────────────────────────────────────

export type IngestBase = {
  /** customer-provided unique id for correlating this asset */
  asset_id: string;
  /** logical group for this asset. the backend maps this to a provider index. */
  asset_class?: string;
  media_kind?: MediaType;
  /** @deprecated use media_kind */
  media_type?: MediaType;
  project_id?: string;
  metadata?: MediaMetadata;
  /** real-world asset timestamp. include an offset when known, e.g. 2026-06-05T14:30:00-07:00. */
  recorded_at?: string;
  /** weak client file modified time in unix milliseconds. */
  file_last_modified_ms?: number;
  types?: MediaSearchType[];
  /**
   * knowledge-bank namespace. when set, routes the asset through the native
   * transcript pipeline (whisper + local embeddings) instead of TwelveLabs.
   * must be 1–128 chars, letters/numbers/hyphens/underscores only.
   * all assets under a prefix share a qdrant collection scoped to your org.
   */
  prefix?: string;
  /**
   * when true, skips TwelveLabs entirely and uses the local whisper + embedding
   * pipeline. requires `prefix` to be set.
   */
  transcript_only?: boolean;
};

export type IngestRequest = IngestBase &
  (
    | {
        source: "url";
        url: string;
      }
    | {
        source: "s3";
        s3_key: string;
      }
    | {
        source: "file";
        file: IngestFile;
        filename: string;
        content_type?: string;
      }
  );

export type IngestResponse = {
  asset_id: string;
  task_id: string;
  status: "pending";
};

// ─── tasks ───────────────────────────────────────────────────────────────────

export type Task = {
  task_id: string;
  asset_id: string;
  asset_class?: string;
  types: MediaSearchType[];
  status: TaskStatus;
  error?: string;
  created_at: string;
  completed_at?: string;
};

// ─── assets ──────────────────────────────────────────────────────────────────

export type ListAssetsParams = {
  page?: number;
  /** max 50 */
  limit?: number;
};

export type AssetListResponse = {
  items: Asset[];
  page_info: PageInfo;
};

// ─── asset classes ───────────────────────────────────────────────────────────

export type ListAssetClassesParams = {
  /** project slug; omit to return classes across every project in the org. */
  project?: string;
};

export type ListAssetClassesResponse = {
  /** sorted alphabetically; case-sensitive match to stored asset_class. */
  asset_classes: string[];
};

// ─── search ──────────────────────────────────────────────────────────────────

export type SearchRequest = {
  query: string;
  types?: MediaSearchType[];
  asset_ids?: string[];
  project_id?: string;
  asset_class?: string;
  /** set this to "all" to intentionally search every asset. */
  scope?: "all";
  /** minimum match score from 0.0 to 1.0. higher values return fewer, more confident results. default 0.5. */
  threshold?: number;
  page?: number;
  /** max 50 */
  limit?: number;
  /**
   * when set, searches the native qdrant collection for this prefix instead
   * of TwelveLabs. results include a `text` field with the matched transcript
   * segment. requires assets ingested with the same prefix.
   */
  prefix?: string;
};

export type SearchResultText = SearchResult & {
  /** transcript segment text. only present on native prefix searches. */
  text: string;
};

export type SearchResult = {
  document_id: string;
  asset_id?: string;
  asset_class?: string;
  twelvelabs_media_id?: string;
  score: number;
  start_ms: number;
  end_ms: number;
  match_type: SearchMatchType;
  text?: string;
  thumbnail_url?: string;
};

export type SearchResponse = {
  search_query_id: string;
  results: SearchResult[];
  page_info: PageInfo;
};

// ─── prefixes ────────────────────────────────────────────────────────────────

export type Prefix = {
  /** the prefix name */
  prefix: string;
  /** true when this prefix was created by the calling api key */
  owned_by_caller: boolean;
};

export type PrefixListResponse = {
  items: Prefix[];
};

// ─── poll ────────────────────────────────────────────────────────────────────

export type PollTaskOptions = {
  /** how long to wait between polls, in ms. default 1500 */
  intervalMs?: number;
  /** max total wait time in ms. default 120_000 (2 min) */
  timeoutMs?: number;
};

// ─── monitors ───────────────────────────────────────────────────────────────

export type Monitor = {
  id: string;
  slug: string;
  text: string;
  project_id?: string;
  asset_class?: string;
  threshold: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
};

export type MonitorListResponse = {
  items: Monitor[];
  total: number;
};

export type CreateMonitorRequest = {
  text: string;
  slug?: string;
  project_id?: string;
  asset_class?: string;
  /** minimum match score 0.0–1.0; default 0.7 server-side */
  threshold?: number;
};

export type UpdateMonitorRequest = {
  text?: string;
  threshold?: number;
  enabled?: boolean;
  asset_class?: string;
};

export type MonitorEvent = {
  id: string;
  monitor_id: string;
  document_id: string;
  asset_id?: string;
  matched: boolean;
  score?: number;
  webhook_sent: boolean;
  match_detail?: Record<string, unknown>;
  created_at: string;
};

export type MonitorEventListResponse = {
  items: MonitorEvent[];
  total: number;
};

export type ListMonitorEventsParams = {
  matched_only?: boolean;
  /** max 100 */
  limit?: number;
  offset?: number;
};

export type AlertOptions = {
  slug?: string;
  project_id?: string;
  asset_class?: string;
  /** minimum match score 0.0–1.0; default 0.7 */
  threshold?: number;
  /** ms between event polls; default 5000 */
  interval_ms?: number;
};

export type AlertHandle = {
  monitor_id: string;
  stop: () => Promise<void>;
};

// ─── error ───────────────────────────────────────────────────────────────────

/** all known API error codes */
export type ApiErrorCode =
  | "bucket_not_connected"
  | "s3_access_denied"
  | "s3_key_not_found"
  | "unsupported_format"
  | "file_too_large"
  | "provider_unavailable"
  | "unauthorized"
  | "not_found"
  | "rate_limited"
  // catch-all for future codes the server may return
  | (string & {});

export type ApiErrorBody = {
  error: {
    code: ApiErrorCode;
    message: string;
  };
};
