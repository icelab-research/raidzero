import {
  SfVoiceMediaError,
  SfVoiceMediaPollTimeoutError,
  SfVoiceMediaRequestTimeoutError,
} from "./errors.js";
import type {
  AlertHandle,
  AlertOptions,
  ApiErrorBody,
  Asset,
  AssetListResponse,
  CreateMonitorRequest,
  IngestRequest,
  IngestResponse,
  ListAssetClassesParams,
  ListAssetClassesResponse,
  ListAssetsParams,
  ListMonitorEventsParams,
  Monitor,
  MonitorEvent,
  MonitorEventListResponse,
  MonitorListResponse,
  PollTaskOptions,
  PrefixListResponse,
  SearchRequest,
  SearchResponse,
  Task,
  UpdateMonitorRequest,
} from "./types.js";

/** sleep utility used by pollTask */
const sleep = (ms: number): Promise<void> =>
  new Promise((resolve) => setTimeout(resolve, ms));

/**
 * Build a URL query string from an object by encoding defined values.
 *
 * @param params - Object whose string/number/boolean values will be URL-encoded; properties with `undefined` are omitted
 * @returns A query string beginning with `?` containing the encoded key/value pairs, or an empty string if no defined parameters exist
 */
function toQueryString(
  params: Record<string, string | number | boolean | undefined>,
): string {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined) as [
    string,
    string | number | boolean,
  ][];
  if (entries.length === 0) return "";
  return (
    "?" +
    new URLSearchParams(
      entries.map(([k, v]) => [k, String(v)] as [string, string]),
    )
  );
}

function toUploadBlob(
  file: Blob | ArrayBuffer | Uint8Array,
  contentType?: string,
): Blob {
  if (file instanceof Blob) return file;

  const bytes =
    file instanceof ArrayBuffer
      ? file
      : (() => {
          const copy = new Uint8Array(file.byteLength);
          copy.set(file);
          return copy.buffer;
        })();

  return contentType !== undefined
    ? new Blob([bytes], { type: contentType })
    : new Blob([bytes]);
}

const DEFAULT_BASE_URL = "https://api.sf-voice.com";

// reads SF_VOICE_BASE_URL from the environment when available (node / edge runtimes).
// falls back to the production URL so browser bundles work without any config.
function resolveBaseUrl(explicit?: string): string {
  if (explicit) return explicit;
  if (typeof process !== "undefined" && process.env?.SF_VOICE_BASE_URL) {
    return process.env.SF_VOICE_BASE_URL;
  }
  return DEFAULT_BASE_URL;
}

export type SfVoiceMediaOptions = {
  /** base URL of the media API. defaults to SF_VOICE_BASE_URL env var, then "https://api.sf-voice.com". */
  baseUrl?: string;
  /** API key sent as a bearer token */
  apiKey: string;
  /** per-request fetch timeout in ms; defaults to 30 000 */
  timeoutMs?: number;
};

/**
 * SDK client for the sf-voice media API.
 *
 * @example
 * ```ts
 * const client = new SfVoiceMedia({
 *   apiKey: process.env.SF_VOICE_API_KEY!,
 * });
 * ```
 */
export class SfVoiceMedia {
  private readonly baseUrl: string;
  private readonly headers: Record<string, string>;
  private readonly timeoutMs: number;

  constructor({ baseUrl, apiKey, timeoutMs = 30_000 }: SfVoiceMediaOptions) {
    // strip trailing slash so every path concat is predictable
    this.baseUrl = resolveBaseUrl(baseUrl).replace(/\/$/, "");
    this.headers = {
      Authorization: `Bearer ${apiKey}`,
    };
    this.timeoutMs = timeoutMs;
  }

  // ─── internal request helper ───────────────────────────────────────────────

  /**
   * executes a fetch and unwraps the JSON body.
   * throws SfVoiceMediaError on any non-2xx status.
   *
   * returns `null` for 204 No Content responses.
   */
  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    initBody?: BodyInit,
    headers: Record<string, string> = {},
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeoutMs);
    const init: RequestInit = {
      method,
      headers: {
        ...this.headers,
        ...headers,
      },
      signal: controller.signal,
    };

    if (initBody !== undefined) {
      init.body = initBody;
    } else if (body !== undefined) {
      init.headers = {
        ...this.headers,
        "Content-Type": "application/json",
        ...headers,
      };
      init.body = JSON.stringify(body);
    }

    let res: Response;
    try {
      res = await fetch(url, init);
    } catch (err) {
      if (err instanceof Error && err.name === "AbortError") {
        throw new SfVoiceMediaRequestTimeoutError(this.timeoutMs);
      }
      throw err;
    } finally {
      clearTimeout(timer);
    }

    // 204 — no body to parse
    if (res.status === 204) {
      return null as T;
    }

    // attempt to parse JSON regardless of status so we get the error payload
    let json: unknown;
    try {
      json = await res.json();
    } catch {
      // non-JSON body on an error status
      throw new SfVoiceMediaError(
        "provider_unavailable",
        `unexpected non-JSON response from server (HTTP ${res.status})`,
        res.status,
      );
    }

    if (!res.ok) {
      const errBody = json as ApiErrorBody;
      throw new SfVoiceMediaError(
        errBody?.error?.code ?? "provider_unavailable",
        errBody?.error?.message ?? `request failed with status ${res.status}`,
        res.status,
      );
    }

    return json as T;
  }

  // ─── public methods ────────────────────────────────────────────────────────

  /**
   * submit a media file for ingestion from a URL, S3 key, or file upload.
   * returns immediately with a task_id you can poll with `getTask` or `pollTask`.
   *
   * @example
   * ```ts
   * const { task_id } = await client.ingest({
   *   source: "url",
   *   asset_id: "customer-video-123",
   *   asset_class: "customer_456",
   *   url: "https://example.com/clip.mp4",
   *   metadata: { title: "demo clip" },
   *   types: ["video", "transcript"],
   * });
   * ```
   */
  async ingest(req: IngestRequest): Promise<IngestResponse> {
    if (req.source === "file") {
      const form = new FormData();
      const file = toUploadBlob(req.file, req.content_type);

      form.append("source", req.source);
      form.append("asset_id", req.asset_id);
      form.append("file", file, req.filename);
      if (req.asset_class !== undefined) {
        form.append("asset_class", req.asset_class);
      }
      if (req.project_id !== undefined) {
        form.append("project_id", req.project_id);
      }
      const mediaKind = req.media_kind ?? req.media_type;
      if (mediaKind !== undefined) form.append("media_kind", mediaKind);
      if (req.content_type !== undefined) {
        form.append("content_type", req.content_type);
      }
      if (req.metadata !== undefined) {
        form.append("metadata", JSON.stringify(req.metadata));
      }
      if (req.recorded_at !== undefined) {
        form.append("recorded_at", req.recorded_at);
      }
      if (req.file_last_modified_ms !== undefined) {
        form.append("file_last_modified_ms", String(req.file_last_modified_ms));
      }
      if (req.types !== undefined)
        form.append("types", JSON.stringify(req.types));
      if (req.prefix !== undefined) form.append("prefix", req.prefix);
      if (req.transcript_only !== undefined)
        form.append("transcript_only", String(req.transcript_only));

      return this.request<IngestResponse>(
        "POST",
        "/api/v1/ingest",
        undefined,
        form,
      );
    }

    const { media_type: legacyMediaType, ...body } = req;
    return this.request<IngestResponse>("POST", "/api/v1/ingest", {
      ...body,
      media_kind: req.media_kind ?? legacyMediaType,
    });
  }

  /**
   * fetch the current state of an ingestion task.
   *
   * @example
   * ```ts
   * const task = await client.getTask("task_abc123");
   * console.log(task.status); // "pending" | "indexing" | "ready" | "failed"
   * ```
   */
  async getTask(taskId: string): Promise<Task> {
    return this.request<Task>(
      "GET",
      `/api/v1/tasks/${encodeURIComponent(taskId)}`,
    );
  }

  /**
   * list all assets in the library, paginated.
   *
   * @example
   * ```ts
   * const { items, page_info } = await client.listAssets({ page: 1, limit: 20 });
   * ```
   */
  async listAssets(params: ListAssetsParams = {}): Promise<AssetListResponse> {
    const qs = toQueryString(params);
    return this.request<AssetListResponse>("GET", `/api/v1/assets${qs}`);
  }

  /**
   * list distinct `asset_class` values the org has used. powers
   * autocomplete UX in dashboards / cli tooling so customers don't
   * mistype an existing class.
   *
   * @example
   * ```ts
   * const { asset_classes } = await client.listAssetClasses({ project: "default" });
   * ```
   */
  async listAssetClasses(
    params: ListAssetClassesParams = {},
  ): Promise<ListAssetClassesResponse> {
    const qs = toQueryString(params);
    return this.request<ListAssetClassesResponse>(
      "GET",
      `/api/v1/asset-classes${qs}`,
    );
  }

  /**
   * list native-pipeline prefixes the org has used.
   * `owned_by_caller` is true when the prefix was created by this API key.
   *
   * @example
   * ```ts
   * const { items } = await client.listPrefixes();
   * ```
   */
  async listPrefixes(): Promise<PrefixListResponse> {
    return this.request<PrefixListResponse>("GET", "/api/v1/prefixes");
  }

  /**
   * fetch a single asset by asset_id.
   *
   * @example
   * ```ts
   * const asset = await client.getAsset("customer-video-123");
   * ```
   */
  async getAsset(assetId: string): Promise<Asset> {
    return this.request<Asset>(
      "GET",
      `/api/v1/assets/${encodeURIComponent(assetId)}`,
    );
  }

  /**
   * delete an asset. the backend performs a soft delete — the record is
   * retained but excluded from list results.
   * resolves void on success (HTTP 204).
   *
   * @example
   * ```ts
   * await client.deleteAsset("customer-video-123");
   * ```
   */
  async deleteAsset(assetId: string): Promise<void> {
    return this.request<void>(
      "DELETE",
      `/api/v1/assets/${encodeURIComponent(assetId)}`,
    );
  }

  /**
   * run a semantic search across indexed media.
   *
   * @example
   * ```ts
   * const { results } = await client.search({
   *   query: "someone mentions the product roadmap",
   *   asset_class: "customer_456",
   *   types: ["transcript"],
   *   threshold: 0.7,
   * });
   * ```
   */
  async search(req: SearchRequest): Promise<SearchResponse> {
    return this.request<SearchResponse>("POST", "/api/v1/search", req);
  }

  /**
   * polls `getTask` at a fixed interval until the task reaches
   * `"ready"` or `"failed"`, then returns the final Task object.
   *
   * throws `SfVoiceMediaPollTimeoutError` if `timeoutMs` is exceeded
   * before a terminal state is reached.
   *
   * @example
   * ```ts
   * const { task_id } = await client.ingest({ source: "url", url: "..." });
   * const task = await client.pollTask(task_id, { intervalMs: 2000, timeoutMs: 60_000 });
   * if (task.status === "failed") throw new Error(task.error);
   * console.log("ready:", task.asset_id);
   * ```
   */
  async pollTask(
    taskId: string,
    { intervalMs = 1500, timeoutMs = 120_000 }: PollTaskOptions = {},
  ): Promise<Task> {
    const deadline = Date.now() + timeoutMs;

    while (true) {
      const task = await this.getTask(taskId);

      if (task.status === "ready" || task.status === "failed") {
        return task;
      }

      if (Date.now() + intervalMs > deadline) {
        throw new SfVoiceMediaPollTimeoutError(taskId, timeoutMs);
      }

      await sleep(intervalMs);
    }
  }

  // ─── monitors ──────────────────────────────────────────────────────────────

  async createMonitor(req: CreateMonitorRequest): Promise<Monitor> {
    return this.request<Monitor>("POST", "/api/v1/monitors", req);
  }

  async listMonitors(): Promise<MonitorListResponse> {
    return this.request<MonitorListResponse>("GET", "/api/v1/monitors");
  }

  async getMonitor(monitorId: string): Promise<Monitor> {
    return this.request<Monitor>(
      "GET",
      `/api/v1/monitors/${encodeURIComponent(monitorId)}`,
    );
  }

  async updateMonitor(
    monitorId: string,
    req: UpdateMonitorRequest,
  ): Promise<Monitor> {
    return this.request<Monitor>(
      "PATCH",
      `/api/v1/monitors/${encodeURIComponent(monitorId)}`,
      req,
    );
  }

  async deleteMonitor(monitorId: string): Promise<void> {
    return this.request<void>(
      "DELETE",
      `/api/v1/monitors/${encodeURIComponent(monitorId)}`,
    );
  }

  async listMonitorEvents(
    monitorId: string,
    params: ListMonitorEventsParams = {},
  ): Promise<MonitorEventListResponse> {
    const qs = toQueryString(
      params as Record<string, string | number | boolean | undefined>,
    );
    return this.request<MonitorEventListResponse>(
      "GET",
      `/api/v1/monitors/${encodeURIComponent(monitorId)}/events${qs}`,
    );
  }

  /**
   * create a monitor for the given search query and poll for matched events
   * in the background. the callback fires once per new matched event.
   *
   * call `handle.stop()` to stop polling and delete the monitor.
   *
   * @example
   * ```ts
   * const handle = await client.alert("someone mentions pricing", (event) => {
   *   console.log("match:", event.asset_id, event.score);
   * });
   *
   * // later…
   * await handle.stop();
   * ```
   */
  async alert(
    text: string,
    callback: (event: MonitorEvent) => void,
    opts: AlertOptions = {},
  ): Promise<AlertHandle> {
    const req: CreateMonitorRequest = { text };
    if (opts.slug !== undefined) req.slug = opts.slug;
    if (opts.project_id !== undefined) req.project_id = opts.project_id;
    if (opts.asset_class !== undefined) req.asset_class = opts.asset_class;
    if (opts.threshold !== undefined) req.threshold = opts.threshold;

    const monitor = await this.createMonitor(req);

    const intervalMs = opts.interval_ms ?? 5_000;
    const pageLimit = 100;
    let checkpoint: string | undefined;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let activePoll: Promise<void> | null = null;
    let pollingError: unknown;
    let consecutiveFailures = 0;
    let stopPromise: Promise<void> | null = null;

    const schedulePoll = () => {
      timer = setTimeout(() => {
        activePoll = poll();
      }, intervalMs);
    };

    const poll = async () => {
      if (stopped) return;
      try {
        let offset = 0;
        let nextCheckpoint: string | undefined;
        const seenThisPoll = new Set<string>();
        const pendingEvents: MonitorEvent[] = [];
        let reachedCheckpoint = false;
        let pages = 0;

        while (!stopped && !reachedCheckpoint) {
          if (pages++ >= 1_000) {
            throw new Error("alert event pagination exceeded 1000 pages");
          }
          const { items, total } = await this.listMonitorEvents(monitor.id, {
            matched_only: true,
            limit: pageLimit,
            offset,
          });
          nextCheckpoint ??= items[0]?.id;

          for (const event of items) {
            if (event.id === checkpoint) {
              reachedCheckpoint = true;
              break;
            }
            if (!seenThisPoll.has(event.id)) {
              seenThisPoll.add(event.id);
              pendingEvents.push(event);
            }
          }

          offset += items.length;
          if (items.length < pageLimit || offset >= total) break;
        }

        for (const event of pendingEvents.reverse()) {
          if (stopped) return;
          try {
            callback(event);
          } catch (error) {
            pollingError = error;
            stopped = true;
            return;
          }
        }

        checkpoint = nextCheckpoint ?? checkpoint;
        consecutiveFailures = 0;
      } catch (error) {
        consecutiveFailures += 1;
        const retryable =
          !(error instanceof SfVoiceMediaError) ||
          error.status === 429 ||
          error.status >= 500;
        if (!retryable || consecutiveFailures >= 3) {
          pollingError = error;
          stopped = true;
          return;
        }
      }
      if (!stopped) {
        schedulePoll();
      }
    };

    schedulePoll();

    return {
      monitor_id: monitor.id,
      stop: () => {
        stopPromise ??= (async () => {
          stopped = true;
          if (timer !== null) clearTimeout(timer);
          await activePoll;
          await this.deleteMonitor(monitor.id);
          if (pollingError !== undefined) throw pollingError;
        })();
        return stopPromise;
      },
    };
  }
}
