defmodule SfVoiceMedia.Types do
  @moduledoc """
  typespecs for all request and response shapes in the sf-voice media API.

  these are documentation-only types — Elixir maps are used at runtime.

  - **keys** are always atoms in both directions. req decodes JSON with
    `keys: :atoms`; the SDK stringifies outgoing atom keys before encoding.
  - **enum values** (status, media_type, source, match_type) are atoms in
    request maps (e.g. `%{source: :url}`) but arrive as strings in response
    maps (e.g. `%{status: "ready"}`), because JSON values are not converted.
  """

  # ── shared ───────────────────────────────────────────────────────────────────

  @typedoc "ingestion source — where the media file comes from"
  @type source :: :url | :s3

  @typedoc "media kind"
  @type media_type :: :video | :audio

  @typedoc "lifecycle state of a task or asset"
  @type task_status :: :pending | :indexing | :ready | :failed

  @typedoc "what kind of match a search result represents"
  @type match_type :: :visual | :conversation | :text_in_video

  # ── pagination ───────────────────────────────────────────────────────────────

  @typedoc """
  pagination envelope returned by list and search endpoints.

      %{total: integer, page: integer, limit: integer, next_page_token: String.t() | nil}
  """
  @type page_info :: %{
          total: non_neg_integer(),
          page: pos_integer(),
          limit: pos_integer(),
          next_page_token: String.t() | nil
        }

  # ── asset ────────────────────────────────────────────────────────────────────

  @typedoc "a single asset stored in the library"
  @type asset :: %{
          asset_id: String.t(),
          filename: String.t() | nil,
          project_id: String.t() | nil,
          asset_class: String.t() | nil,
          # "video" | "audio"
          media_type: String.t(),
          # "url" | "s3"
          source_type: String.t(),
          types: [String.t()],
          # "pending" | "indexing" | "ready" | "failed"
          status: String.t(),
          metadata: map() | nil,
          duration_ms: non_neg_integer() | nil,
          thumbnail_url: String.t() | nil,
          thumbnail_status: String.t() | nil,
          created_at: String.t(),
          updated_at: String.t()
        }

  # ── ingest ───────────────────────────────────────────────────────────────────

  @typedoc """
  request body for `POST /api/v1/ingest`.

  either `:url` (with `:url` key) or `:s3` (with `:s3_key` key).
  `:media_kind`, `:prefix`, `:transcript_only`, and `:metadata` are optional.
  """
  @type ingest_request :: %{
          required(:source) => source(),
          required(:asset_id) => String.t(),
          optional(:asset_class) => String.t(),
          optional(:project_id) => String.t(),
          optional(:url) => String.t(),
          optional(:s3_key) => String.t(),
          optional(:media_kind) => media_type(),
          optional(:prefix) => String.t(),
          optional(:transcript_only) => boolean(),
          optional(:metadata) => map()
        }

  @typedoc "202 response body from `POST /api/v1/ingest`"
  @type ingest_response :: %{
          asset_id: String.t(),
          task_id: String.t(),
          # "pending"
          status: String.t()
        }

  # ── tasks ────────────────────────────────────────────────────────────────────

  @typedoc "response from `GET /api/v1/tasks/:task_id`"
  @type task :: %{
          task_id: String.t(),
          asset_id: String.t(),
          # "pending" | "indexing" | "ready" | "failed"
          status: String.t(),
          error: String.t() | nil,
          created_at: String.t(),
          completed_at: String.t() | nil
        }

  # ── list assets ──────────────────────────────────────────────────────────────

  @typedoc "optional query params for `GET /api/v1/assets`"
  @type list_assets_params :: %{
          optional(:page) => pos_integer(),
          optional(:limit) => pos_integer()
        }

  @typedoc "response from `GET /api/v1/assets`"
  @type asset_list_response :: %{
          items: [asset()],
          page_info: page_info()
        }

  # ── search ───────────────────────────────────────────────────────────────────

  @typedoc "request body for `POST /api/v1/search`"
  @type search_request :: %{
          required(:query) => String.t(),
          optional(:types) => [match_type()],
          optional(:asset_ids) => [String.t()],
          optional(:asset_class) => String.t(),
          optional(:project_id) => String.t(),
          optional(:prefix) => String.t(),
          optional(:threshold) => float(),
          optional(:page) => pos_integer(),
          optional(:limit) => pos_integer()
        }

  @typedoc "a single match returned by the search endpoint"
  @type search_result :: %{
          document_id: String.t(),
          asset_id: String.t() | nil,
          asset_class: String.t() | nil,
          twelvelabs_media_id: String.t() | nil,
          score: float(),
          start_ms: non_neg_integer(),
          end_ms: non_neg_integer(),
          # "visual" | "conversation" | "text_in_video"
          match_type: String.t(),
          text: String.t() | nil,
          thumbnail_url: String.t() | nil
        }

  @typedoc "response from `POST /api/v1/search`"
  @type search_response :: %{
          search_query_id: String.t(),
          results: [search_result()],
          page_info: page_info()
        }

  @type prefix_item :: %{prefix: String.t(), owned_by_caller: boolean()}
  @type prefix_list_response :: %{items: [prefix_item()]}

  # ── monitors ─────────────────────────────────────────────────────────────────

  @typedoc "a monitor that watches for matching content across ingested media"
  @type monitor :: %{
          required(:id) => String.t(),
          required(:slug) => String.t(),
          required(:text) => String.t(),
          required(:threshold) => float(),
          required(:enabled) => boolean(),
          required(:created_at) => String.t(),
          required(:updated_at) => String.t(),
          optional(:project_id) => String.t(),
          optional(:asset_class) => String.t()
        }

  @typedoc "paginated list of monitors from `GET /api/v1/monitors`"
  @type monitor_list_response :: %{
          required(:items) => [monitor()],
          required(:total) => non_neg_integer()
        }

  @typedoc "request body for `POST /api/v1/monitors`"
  @type create_monitor_request :: %{
          required(:text) => String.t(),
          optional(:slug) => String.t(),
          optional(:project_id) => String.t(),
          optional(:asset_class) => String.t(),
          optional(:threshold) => float()
        }

  @typedoc "request body for `PATCH /api/v1/monitors/:id`"
  @type update_monitor_request :: %{
          optional(:text) => String.t(),
          optional(:threshold) => float(),
          optional(:enabled) => boolean(),
          optional(:asset_class) => String.t()
        }

  @typedoc "a single event fired when a monitor matches (or is evaluated against) a document"
  @type monitor_event :: %{
          required(:id) => String.t(),
          required(:monitor_id) => String.t(),
          required(:document_id) => String.t(),
          required(:matched) => boolean(),
          required(:webhook_sent) => boolean(),
          required(:created_at) => String.t(),
          optional(:asset_id) => String.t(),
          optional(:score) => float(),
          optional(:match_detail) => map()
        }

  @typedoc "paginated list of monitor events from `GET /api/v1/monitors/:id/events`"
  @type monitor_event_list_response :: %{
          required(:items) => [monitor_event()],
          required(:total) => non_neg_integer()
        }

  # ── poll_task opts ───────────────────────────────────────────────────────────

  @typedoc """
  options for `SfVoiceMedia.poll_task!/3`.

  - `:interval_ms` — milliseconds between polls (default 1_500)
  - `:timeout_ms`  — max total wait time in ms before raising (default 120_000)
  """
  @type poll_opts :: %{
          optional(:interval_ms) => pos_integer(),
          optional(:timeout_ms) => pos_integer()
        }
end
