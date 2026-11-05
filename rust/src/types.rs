//! all public types that mirror the sf-voice media API's wire shapes.
//! serde rename_all = "snake_case" is used throughout since the API
//! returns snake_case JSON and Rust convention is snake_case anyway,
//! so no per-field renames are needed.

use serde::{Deserialize, Serialize};

// ─── enums ───────────────────────────────────────────────────────────────────

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum MediaType {
    Video,
    Audio,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum SourceType {
    Url,
    S3,
    File,
    Upload,
    Youtube,
    Twitter,
}

pub type AssetSourceType = SourceType;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum TaskStatus {
    Pending,
    Indexing,
    Ready,
    Failed,
}

impl TaskStatus {
    /// Indicates whether the task is in a terminal state.
    ///
    /// A terminal state is a non-progressing final status for a task.
    ///
    /// # Returns
    ///
    /// `true` if the status is `Ready` or `Failed`, `false` otherwise.
    ///
    /// # Examples
    ///
    /// ```
    /// use sf_voice_media::types::TaskStatus;
    ///
    /// let ready = TaskStatus::Ready;
    /// assert!(ready.is_terminal());
    ///
    /// let pending = TaskStatus::Pending;
    /// assert!(!pending.is_terminal());
    /// ```
    pub fn is_terminal(&self) -> bool {
        matches!(self, TaskStatus::Ready | TaskStatus::Failed)
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum SearchType {
    Visual,
    Conversation,
    TextInVideo,
    Video,
    Audio,
    Transcript,
}

// ─── shared ──────────────────────────────────────────────────────────────────

/// optional metadata attached to an asset at ingest time.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct MediaMetadata {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub title: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
}

/// pagination envelope returned by list and search endpoints.
#[derive(Debug, Clone, Deserialize)]
pub struct PageInfo {
    pub total: u64,
    pub page: u32,
    pub limit: u32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_page_token: Option<String>,
}

// ─── assets ──────────────────────────────────────────────────────────────────

/// a single asset in the library.
#[derive(Debug, Clone)]
pub struct Asset {
    pub id: String,
    pub filename: Option<String>,
    pub project_id: Option<String>,
    pub asset_class: Option<String>,
    pub media_type: MediaType,
    pub source_type: AssetSourceType,
    pub types: Vec<SearchType>,
    pub status: TaskStatus,
    pub metadata: Option<MediaMetadata>,
    pub metadata_raw: Option<serde_json::Value>,
    pub duration_ms: Option<u64>,
    pub thumbnail_url: Option<String>,
    pub thumbnail_status: Option<String>,
    pub created_at: String,
    pub updated_at: String,
}

impl Asset {
    /// returns the current wire-contract asset identifier.
    pub fn asset_id(&self) -> &str {
        &self.id
    }

    /// returns the complete metadata object without dropping unknown values.
    pub fn metadata_value(&self) -> Option<&serde_json::Value> {
        self.metadata_raw.as_ref()
    }
}

#[derive(Deserialize)]
struct AssetWire {
    asset_id: String,
    filename: Option<String>,
    project_id: Option<String>,
    asset_class: Option<String>,
    media_type: MediaType,
    source_type: AssetSourceType,
    types: Vec<SearchType>,
    status: TaskStatus,
    metadata: Option<serde_json::Value>,
    duration_ms: Option<u64>,
    thumbnail_url: Option<String>,
    thumbnail_status: Option<String>,
    created_at: String,
    updated_at: String,
}

impl<'de> Deserialize<'de> for Asset {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let wire = AssetWire::deserialize(deserializer)?;
        let metadata = wire.metadata.as_ref().and_then(|value| {
            let object = value.as_object()?;
            Some(MediaMetadata {
                title: object
                    .get("title")
                    .and_then(|item| item.as_str())
                    .map(str::to_owned),
                tags: object.get("tags").and_then(|item| {
                    item.as_array().map(|items| {
                        items
                            .iter()
                            .filter_map(|item| item.as_str().map(str::to_owned))
                            .collect()
                    })
                }),
            })
        });

        Ok(Self {
            id: wire.asset_id,
            filename: wire.filename,
            project_id: wire.project_id,
            asset_class: wire.asset_class,
            media_type: wire.media_type,
            source_type: wire.source_type,
            types: wire.types,
            status: wire.status,
            metadata,
            metadata_raw: wire.metadata,
            duration_ms: wire.duration_ms,
            thumbnail_url: wire.thumbnail_url,
            thumbnail_status: wire.thumbnail_status,
            created_at: wire.created_at,
            updated_at: wire.updated_at,
        })
    }
}

/// response envelope for `GET /api/v1/assets`.
#[derive(Debug, Clone, Deserialize)]
pub struct AssetListResponse {
    pub items: Vec<Asset>,
    pub page_info: PageInfo,
}

// ─── ingest ──────────────────────────────────────────────────────────────────

/// source-specific body for `POST /api/v1/ingest`.
/// use `IngestRequest::from_url` or `IngestRequest::from_s3` then chain
/// `.media_kind()` / `.metadata()` to fill optional fields.
#[derive(Debug, Clone, Serialize)]
pub struct IngestRequest {
    pub source: SourceType,
    pub asset_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub asset_class: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub project_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub s3_key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[serde(rename = "media_kind")]
    pub media_type: Option<MediaType>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<MediaMetadata>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub prefix: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub transcript_only: Option<bool>,
}

impl IngestRequest {
    /// Create an `IngestRequest` targeting a public or pre-signed URL.
    ///
    /// The returned request has `source` set to `SourceType::Url`, `url` set to the provided value,
    /// and `s3_key`, `media_type`, and `metadata` left unset. a stable bounded
    /// compatibility asset id is derived from the source URL.
    ///
    /// # Examples
    ///
    /// ```
    /// use sf_voice_media::types::{IngestRequest, SourceType};
    ///
    /// let req = IngestRequest::from_url("https://example.com/media.mp4");
    /// assert_eq!(req.url.as_deref(), Some("https://example.com/media.mp4"));
    /// assert_eq!(req.source, SourceType::Url);
    /// ```
    pub fn from_url(url: impl Into<String>) -> Self {
        let url = url.into();
        Self::from_url_with_asset_id(compatibility_asset_id(&url), url)
    }

    /// creates a URL ingest request with an explicit customer asset id.
    pub fn from_url_with_asset_id(asset_id: impl Into<String>, url: impl Into<String>) -> Self {
        Self {
            source: SourceType::Url,
            asset_id: asset_id.into(),
            asset_class: None,
            project_id: None,
            url: Some(url.into()),
            s3_key: None,
            media_type: None,
            metadata: None,
            prefix: None,
            transcript_only: None,
        }
    }

    /// Creates an `IngestRequest` configured to ingest from an S3 object key.
    ///
    /// A stable bounded compatibility asset id is derived from the object key;
    /// other optional fields are left unset.
    ///
    /// # Examples
    ///
    /// ```
    /// use sf_voice_media::types::{IngestRequest, SourceType};
    ///
    /// let req = IngestRequest::from_s3("path/to/object.mp4");
    /// assert_eq!(req.source, SourceType::S3);
    /// assert_eq!(req.s3_key.as_deref(), Some("path/to/object.mp4"));
    /// ```
    pub fn from_s3(s3_key: impl Into<String>) -> Self {
        let s3_key = s3_key.into();
        Self::from_s3_with_asset_id(compatibility_asset_id(&s3_key), s3_key)
    }

    /// creates an S3 ingest request with an explicit customer asset id.
    pub fn from_s3_with_asset_id(asset_id: impl Into<String>, s3_key: impl Into<String>) -> Self {
        Self {
            source: SourceType::S3,
            asset_id: asset_id.into(),
            asset_class: None,
            project_id: None,
            url: None,
            s3_key: Some(s3_key.into()),
            media_type: None,
            metadata: None,
            prefix: None,
            transcript_only: None,
        }
    }

    /// Set the media type hint for the ingest request.
    ///
    /// When set, the request will include this media type so the server can treat the provided source as video or audio.
    ///
    /// # Examples
    ///
    /// ```
    /// use sf_voice_media::types::{IngestRequest, MediaType};
    ///
    /// let req = IngestRequest::from_url("https://example.com/video.mp4").media_kind(MediaType::Video);
    /// assert_eq!(req.media_type.unwrap(), MediaType::Video);
    /// ```
    pub fn media_kind(mut self, media_kind: MediaType) -> Self {
        self.media_type = Some(media_kind);
        self
    }

    /// deprecated compatibility alias for [`Self::media_kind`].
    pub fn media_type(mut self, media_type: MediaType) -> Self {
        self.media_type = Some(media_type);
        self
    }

    /// Scope the asset to a project UUID.
    pub fn project_id(mut self, project_id: impl Into<String>) -> Self {
        self.project_id = Some(project_id.into());
        self
    }

    /// Route this asset through a native transcript prefix.
    pub fn transcript_prefix(mut self, prefix: impl Into<String>) -> Self {
        self.prefix = Some(prefix.into());
        self.transcript_only = Some(true);
        self
    }

    /// Sets the media metadata on the ingest request.
    ///
    /// # Examples
    ///
    /// ```
    /// use sf_voice_media::types::{IngestRequest, MediaMetadata};
    ///
    /// let req = IngestRequest::from_url("https://example.com/video.mp4")
    ///     .metadata(MediaMetadata { title: Some("Example".into()), tags: None });
    /// assert!(req.metadata.is_some());
    /// ```
    pub fn metadata(mut self, metadata: MediaMetadata) -> Self {
        self.metadata = Some(metadata);
        self
    }
}

/// response body for a successful `POST /api/v1/ingest` (HTTP 202).
#[derive(Debug, Clone, Deserialize)]
pub struct IngestResponse {
    pub asset_id: String,
    pub task_id: String,
    /// always "pending" at ingest time.
    pub status: String,
}

// ─── tasks ───────────────────────────────────────────────────────────────────

/// response body for `GET /api/v1/tasks/:task_id`.
#[derive(Debug, Clone, Deserialize)]
pub struct Task {
    pub task_id: String,
    pub asset_id: String,
    pub asset_class: Option<String>,
    pub types: Vec<SearchType>,
    pub status: TaskStatus,
    pub error: Option<String>,
    pub created_at: String,
    pub completed_at: Option<String>,
}

// ─── search ──────────────────────────────────────────────────────────────────

/// request body for `POST /api/v1/search`.
/// use `SearchRequest::new(query)` then chain optional setters.
#[derive(Debug, Clone, Serialize)]
pub struct SearchRequest {
    pub query: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub types: Option<Vec<SearchType>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub asset_ids: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub asset_class: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub project_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub prefix: Option<String>,
    /// similarity threshold between 0.0 and 1.0. defaults to 0.5 server-side.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub threshold: Option<f32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub page: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub limit: Option<u32>,
}

impl SearchRequest {
    /// Constructs a SearchRequest with the provided query and all optional fields unset.
    ///
    /// # Examples
    ///
    /// ```
    /// use sf_voice_media::types::SearchRequest;
    ///
    /// let req = SearchRequest::new("find cats");
    /// assert_eq!(req.query, "find cats");
    /// assert!(req.types.is_none());
    /// assert!(req.asset_ids.is_none());
    /// assert!(req.threshold.is_none());
    /// assert!(req.page.is_none());
    /// assert!(req.limit.is_none());
    /// ```
    pub fn new(query: impl Into<String>) -> Self {
        Self {
            query: query.into(),
            types: None,
            asset_ids: None,
            asset_class: None,
            project_id: None,
            prefix: None,
            threshold: None,
            page: None,
            limit: None,
        }
    }

    /// Set which search modalities the request should match.
    ///
    /// This limits the search to the provided `SearchType` variants.
    ///
    /// # Examples
    ///
    /// ```
    /// use sf_voice_media::types::{SearchRequest, SearchType};
    ///
    /// let req = SearchRequest::new("query").types(vec![SearchType::Visual, SearchType::TextInVideo]);
    /// assert!(req.types.is_some());
    /// ```
    pub fn types(mut self, types: Vec<SearchType>) -> Self {
        self.types = Some(types);
        self
    }

    /// Restricts the search to the provided asset IDs.
    ///
    /// Sets the request's `asset_ids` field to the supplied list of identifiers (each element
    /// is converted to a `String`) and returns the updated request for chaining.
    ///
    /// # Examples
    ///
    /// ```
    /// use sf_voice_media::types::SearchRequest;
    ///
    /// let req = SearchRequest::new("find this")
    ///     .asset_ids(vec!["asset-1", "asset-2"]);
    /// assert!(req.asset_ids.is_some());
    /// ```
    pub fn asset_ids(mut self, ids: Vec<impl Into<String>>) -> Self {
        self.asset_ids = Some(ids.into_iter().map(Into::into).collect());
        self
    }

    /// Search a native transcript prefix.
    pub fn prefix(mut self, prefix: impl Into<String>) -> Self {
        self.prefix = Some(prefix.into());
        self
    }

    /// Sets the similarity threshold used to filter search results.
    ///
    /// Values should be in the range 0.0 to 1.0; only results with scores greater than or equal to
    /// the threshold will be considered.
    ///
    /// # Examples
    ///
    /// ```
    /// use sf_voice_media::types::SearchRequest;
    ///
    /// let req = SearchRequest::new("example query").threshold(0.75);
    /// assert_eq!(req.threshold, Some(0.75));
    /// ```
    pub fn threshold(mut self, threshold: f32) -> Self {
        self.threshold = Some(threshold);
        self
    }

    /// Set the 1-based page number for the search request.
    ///
    /// Pages are 1-based; pass `1` for the first page.
    ///
    /// # Examples
    ///
    /// ```
    /// use sf_voice_media::types::SearchRequest;
    ///
    /// let req = SearchRequest::new("query").page(2);
    /// ```
    pub fn page(mut self, page: u32) -> Self {
        self.page = Some(page);
        self
    }

    /// Sets the maximum number of results returned per page.
    ///
    /// Returns the updated `SearchRequest` with its `limit` set to the given value.
    ///
    /// # Examples
    ///
    /// ```
    /// use sf_voice_media::types::SearchRequest;
    ///
    /// let req = SearchRequest::new("cats").limit(25);
    /// assert_eq!(req.limit, Some(25));
    /// ```
    pub fn limit(mut self, limit: u32) -> Self {
        self.limit = Some(limit);
        self
    }
}

/// a single search hit.
#[derive(Debug, Clone, Deserialize)]
pub struct SearchResult {
    pub document_id: String,
    #[serde(default, deserialize_with = "deserialize_nullable_string")]
    pub asset_id: String,
    pub asset_class: Option<String>,
    pub twelvelabs_media_id: Option<String>,
    /// similarity score, higher is better.
    pub score: f32,
    /// start of the matching segment in milliseconds.
    pub start_ms: u64,
    /// end of the matching segment in milliseconds.
    pub end_ms: u64,
    pub match_type: SearchType,
    pub text: Option<String>,
    pub thumbnail_url: Option<String>,
}

fn compatibility_asset_id(value: &str) -> String {
    let hash = value
        .as_bytes()
        .iter()
        .fold(0xcbf29ce484222325_u64, |hash, byte| {
            (hash ^ u64::from(*byte)).wrapping_mul(0x100000001b3)
        });
    format!("legacy-{hash:016x}")
}

fn deserialize_nullable_string<'de, D>(deserializer: D) -> Result<String, D::Error>
where
    D: serde::Deserializer<'de>,
{
    Ok(Option::<String>::deserialize(deserializer)?.unwrap_or_default())
}

/// response body for `POST /api/v1/search`.
#[derive(Debug, Clone, Deserialize)]
pub struct SearchResponse {
    pub search_query_id: String,
    pub results: Vec<SearchResult>,
    pub page_info: PageInfo,
}

/// a native transcript prefix visible to the current api key.
#[derive(Debug, Clone, Deserialize)]
pub struct Prefix {
    pub prefix: String,
    pub owned_by_caller: bool,
}

/// response envelope for `GET /api/v1/prefixes`.
#[derive(Debug, Clone, Deserialize)]
pub struct PrefixListResponse {
    pub items: Vec<Prefix>,
}

// ─── monitors ───────────────────────────────────────────────────────────────

/// request body for `POST /api/v1/monitors`.
/// use `CreateMonitorRequest::new(text)` then chain optional setters.
#[derive(Debug, Clone, Serialize)]
pub struct CreateMonitorRequest {
    pub text: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub slug: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub project_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub asset_class: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub threshold: Option<f32>,
}

impl CreateMonitorRequest {
    /// create a monitor request with the given text and all optional fields unset.
    pub fn new(text: impl Into<String>) -> Self {
        Self {
            text: text.into(),
            slug: None,
            project_id: None,
            asset_class: None,
            threshold: None,
        }
    }

    pub fn slug(mut self, slug: impl Into<String>) -> Self {
        self.slug = Some(slug.into());
        self
    }

    pub fn project_id(mut self, project_id: impl Into<String>) -> Self {
        self.project_id = Some(project_id.into());
        self
    }

    pub fn asset_class(mut self, asset_class: impl Into<String>) -> Self {
        self.asset_class = Some(asset_class.into());
        self
    }

    pub fn threshold(mut self, threshold: f32) -> Self {
        self.threshold = Some(threshold);
        self
    }
}

/// request body for `PATCH /api/v1/monitors/:id`.
#[derive(Debug, Clone, Default, Serialize)]
pub struct UpdateMonitorRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub text: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub threshold: Option<f32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub enabled: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub asset_class: Option<String>,
}

/// a single monitor resource.
#[derive(Debug, Clone, Deserialize)]
pub struct Monitor {
    pub id: String,
    pub slug: String,
    pub text: String,
    #[serde(default)]
    pub project_id: Option<String>,
    #[serde(default)]
    pub asset_class: Option<String>,
    pub threshold: f32,
    pub enabled: bool,
    pub created_at: String,
    pub updated_at: String,
}

/// response envelope for `GET /api/v1/monitors`.
#[derive(Debug, Clone, Deserialize)]
pub struct MonitorListResponse {
    pub items: Vec<Monitor>,
    pub total: u64,
}

/// a single monitor event (match evaluation result).
#[derive(Debug, Clone, Deserialize)]
pub struct MonitorEvent {
    pub id: String,
    pub monitor_id: String,
    pub document_id: String,
    #[serde(default)]
    pub asset_id: Option<String>,
    pub matched: bool,
    #[serde(default)]
    pub score: Option<f32>,
    pub webhook_sent: bool,
    #[serde(default)]
    pub match_detail: Option<serde_json::Value>,
    pub created_at: String,
}

/// response envelope for `GET /api/v1/monitors/:id/events`.
#[derive(Debug, Clone, Deserialize)]
pub struct MonitorEventListResponse {
    pub items: Vec<MonitorEvent>,
    pub total: u64,
}

/// options for the high-level `alert()` convenience method.
#[derive(Debug, Clone, Default)]
pub struct AlertOptions {
    pub slug: Option<String>,
    pub project_id: Option<String>,
    pub asset_class: Option<String>,
    pub threshold: Option<f32>,
    pub interval_ms: Option<u64>,
}

#[cfg(test)]
mod tests {
    use super::Asset;

    #[test]
    fn asset_accepts_arbitrary_metadata() {
        let asset: Asset = serde_json::from_value(serde_json::json!({
            "asset_id": "asset-1",
            "media_type": "audio",
            "source_type": "upload",
            "types": [],
            "status": "ready",
            "metadata": {
                "title": "call",
                "score": 0.9,
                "nested": { "source": "crm" }
            },
            "created_at": "2026-11-05T00:00:00Z",
            "updated_at": "2026-11-05T00:00:00Z"
        }))
        .expect("asset metadata should decode");

        assert_eq!(
            asset
                .metadata
                .as_ref()
                .and_then(|metadata| metadata.title.as_deref()),
            Some("call")
        );
        assert_eq!(
            asset
                .metadata_value()
                .and_then(|value| value["score"].as_f64()),
            Some(0.9)
        );
    }
}
