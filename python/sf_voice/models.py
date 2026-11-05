"""
public request and response types for the sf-voice media api.
the shapes mirror the typescript sdk contract.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import (
    Any,
    Awaitable,
    BinaryIO,
    Callable,
    Dict,
    List,
    Literal,
    Optional,
    TypedDict,
    Union,
    cast,
)


MediaType = Literal["video", "audio"]
SourceType = Literal["url", "s3", "file"]
AssetSourceType = Literal["url", "s3", "file", "upload", "youtube", "twitter"]
TaskStatus = Literal["pending", "indexing", "ready", "failed"]
MediaSearchType = Literal[
    "video",
    "audio",
    "transcript",
    "visual",
    "conversation",
    "text_in_video",
]
MediaMetadataValue = Union[str, int, float, bool]
MediaMetadata = Dict[str, MediaMetadataValue]
IngestFile = Union[bytes, bytearray, memoryview, BinaryIO]
ApiErrorCode = str


class _IngestOptional(TypedDict, total=False):
    asset_class: str
    project_id: str
    media_kind: MediaType
    media_type: MediaType
    metadata: MediaMetadata
    types: List[MediaSearchType]
    prefix: str
    transcript_only: bool


class UrlIngestRequest(_IngestOptional):
    source: Literal["url"]
    asset_id: str
    url: str


class S3IngestRequest(_IngestOptional):
    source: Literal["s3"]
    asset_id: str
    s3_key: str


class _FileIngestOptional(_IngestOptional, total=False):
    content_type: str


class FileIngestRequest(_FileIngestOptional):
    source: Literal["file"]
    asset_id: str
    file: IngestFile
    filename: str


IngestRequest = Union[UrlIngestRequest, S3IngestRequest, FileIngestRequest]


class ListAssetsParams(TypedDict, total=False):
    page: int
    limit: int


class _SearchOptional(TypedDict, total=False):
    types: List[MediaSearchType]
    asset_ids: List[str]
    asset_class: str
    project_id: str
    prefix: str
    scope: Literal["all"]
    threshold: float
    page: int
    limit: int


class SearchRequest(_SearchOptional):
    query: str


class PollTaskOptions(TypedDict, total=False):
    interval_ms: int
    timeout_ms: int


@dataclass
class PageInfo:
    total: int
    page: int
    limit: int
    next_page_token: Optional[str] = None

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "PageInfo":
        """build page metadata from the api response."""
        return cls(
            total=d["total"],
            page=d["page"],
            limit=d["limit"],
            next_page_token=d.get("next_page_token"),
        )


@dataclass
class Asset:
    asset_id: str
    media_type: MediaType
    source_type: AssetSourceType
    types: List[MediaSearchType]
    status: TaskStatus
    created_at: str
    updated_at: str
    asset_class: Optional[str] = None
    filename: Optional[str] = None
    project_id: Optional[str] = None
    metadata: Optional[MediaMetadata] = None
    duration_ms: Optional[int] = None
    thumbnail_url: Optional[str] = None
    thumbnail_status: Optional[str] = None

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "Asset":
        """build an asset from the canonical asset response."""
        return cls(
            asset_id=d["asset_id"],
            asset_class=d.get("asset_class"),
            filename=d.get("filename"),
            project_id=d.get("project_id"),
            media_type=cast(MediaType, d["media_type"]),
            source_type=cast(AssetSourceType, d["source_type"]),
            types=[cast(MediaSearchType, value) for value in d["types"]],
            status=cast(TaskStatus, d["status"]),
            metadata=d.get("metadata"),
            duration_ms=d.get("duration_ms"),
            thumbnail_url=d.get("thumbnail_url"),
            thumbnail_status=d.get("thumbnail_status"),
            created_at=d["created_at"],
            updated_at=d["updated_at"],
        )


@dataclass
class Task:
    task_id: str
    asset_id: str
    types: List[MediaSearchType]
    status: TaskStatus
    created_at: str
    asset_class: Optional[str] = None
    error: Optional[str] = None
    completed_at: Optional[str] = None

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "Task":
        """build a task from the canonical task response."""
        return cls(
            task_id=d["task_id"],
            asset_id=d["asset_id"],
            asset_class=d.get("asset_class"),
            types=[cast(MediaSearchType, value) for value in d["types"]],
            status=cast(TaskStatus, d["status"]),
            error=d.get("error"),
            created_at=d["created_at"],
            completed_at=d.get("completed_at"),
        )

    @property
    def is_terminal(self) -> bool:
        """return true once the task has reached a final state."""
        return self.status in ("ready", "failed")


@dataclass
class IngestResponse:
    asset_id: str
    task_id: str
    status: Literal["pending"]

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "IngestResponse":
        """build an ingest response from the api response."""
        return cls(
            asset_id=d["asset_id"],
            task_id=d["task_id"],
            status=cast(Literal["pending"], d["status"]),
        )


@dataclass
class SearchResult:
    document_id: str
    score: float
    start_ms: int
    end_ms: int
    match_type: MediaSearchType
    asset_id: Optional[str] = None
    asset_class: Optional[str] = None
    twelvelabs_media_id: Optional[str] = None
    text: Optional[str] = None
    thumbnail_url: Optional[str] = None

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "SearchResult":
        """build one search result from the api response."""
        return cls(
            document_id=d["document_id"],
            asset_id=d.get("asset_id"),
            asset_class=d.get("asset_class"),
            twelvelabs_media_id=d.get("twelvelabs_media_id"),
            score=d["score"],
            start_ms=d["start_ms"],
            end_ms=d["end_ms"],
            match_type=cast(MediaSearchType, d["match_type"]),
            text=d.get("text"),
            thumbnail_url=d.get("thumbnail_url"),
        )


@dataclass
class SearchResponse:
    search_query_id: str
    results: List[SearchResult]
    page_info: PageInfo

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "SearchResponse":
        """build a search response from the api response."""
        return cls(
            search_query_id=d["search_query_id"],
            results=[SearchResult.from_dict(item) for item in d.get("results", [])],
            page_info=PageInfo.from_dict(d["page_info"]),
        )


@dataclass
class AssetListResponse:
    items: List[Asset]
    page_info: PageInfo

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "AssetListResponse":
        """build a paginated asset list from the api response."""
        return cls(
            items=[Asset.from_dict(item) for item in d.get("items", [])],
            page_info=PageInfo.from_dict(d["page_info"]),
        )


@dataclass
class Prefix:
    prefix: str
    owned_by_caller: bool

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "Prefix":
        """build a prefix item from the api response."""
        return cls(prefix=d["prefix"], owned_by_caller=d["owned_by_caller"])


@dataclass
class PrefixListResponse:
    items: List[Prefix]

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "PrefixListResponse":
        """build a prefix list from the api response."""
        return cls(items=[Prefix.from_dict(item) for item in d.get("items", [])])


# ---------------------------------------------------------------------------
# monitors
# ---------------------------------------------------------------------------


# text is required; the rest are optional
class _CreateMonitorRequired(TypedDict):
    text: str


class CreateMonitorRequest(_CreateMonitorRequired, total=False):
    slug: str
    project_id: str
    asset_class: str
    threshold: float


class UpdateMonitorRequest(TypedDict, total=False):
    text: str
    threshold: float
    enabled: bool
    asset_class: str


class ListMonitorEventsParams(TypedDict, total=False):
    matched_only: bool
    limit: int
    offset: int


@dataclass
class Monitor:
    id: str
    slug: str
    text: str
    threshold: float
    enabled: bool
    created_at: str
    updated_at: str
    project_id: Optional[str] = None
    asset_class: Optional[str] = None

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "Monitor":
        """build a monitor from the api response."""
        return cls(
            id=d["id"],
            slug=d["slug"],
            text=d["text"],
            threshold=d["threshold"],
            enabled=d["enabled"],
            created_at=d["created_at"],
            updated_at=d["updated_at"],
            project_id=d.get("project_id"),
            asset_class=d.get("asset_class"),
        )


@dataclass
class MonitorEvent:
    id: str
    monitor_id: str
    document_id: str
    matched: bool
    webhook_sent: bool
    created_at: str
    asset_id: Optional[str] = None
    score: Optional[float] = None
    match_detail: Optional[Dict[str, Any]] = None

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "MonitorEvent":
        """build a monitor event from the api response."""
        return cls(
            id=d["id"],
            monitor_id=d["monitor_id"],
            document_id=d["document_id"],
            matched=d["matched"],
            webhook_sent=d["webhook_sent"],
            created_at=d["created_at"],
            asset_id=d.get("asset_id"),
            score=d.get("score"),
            match_detail=d.get("match_detail"),
        )


@dataclass
class MonitorListResponse:
    items: List[Monitor]
    total: int

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "MonitorListResponse":
        """build a paginated monitor list from the api response."""
        return cls(
            items=[Monitor.from_dict(item) for item in d.get("items", [])],
            total=d["total"],
        )


@dataclass
class MonitorEventListResponse:
    items: List[MonitorEvent]
    total: int

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "MonitorEventListResponse":
        """build a paginated monitor event list from the api response."""
        return cls(
            items=[MonitorEvent.from_dict(item) for item in d.get("items", [])],
            total=d["total"],
        )


@dataclass
class AlertHandle:
    """handle returned by the alert() convenience method."""

    monitor_id: str
    _stop: Callable[[], None]

    def stop(self) -> None:
        """stop polling and delete the monitor."""
        self._stop()


@dataclass
class AsyncAlertHandle:
    """handle returned by the asynchronous alert convenience method."""

    monitor_id: str
    _stop: Callable[[], Awaitable[None]]

    async def stop(self) -> None:
        """stop polling and await monitor deletion."""
        await self._stop()
