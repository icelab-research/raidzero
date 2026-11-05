package com.sfvoice.media;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.sfvoice.media.models.*;

import java.io.IOException;
import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.function.Consumer;

/**
 * blocking HTTP client for the sf-voice media API.
 *
 * <pre>{@code
 * SfVoiceMediaClient client = new SfVoiceMediaClient.Builder()
 *     .apiKey(System.getenv("SF_VOICE_API_KEY"))
 *     .baseUrl("https://api.sf-voice.com")
 *     .build();
 * }</pre>
 */
public class SfVoiceMediaClient {

    private final String baseUrl;
    private final String apiKey;
    private final HttpClient http;
    private final ObjectMapper json;

    /**
     * Constructs an SfVoiceMediaClient from the provided Builder.
     *
     * Normalizes the builder's baseUrl by removing trailing slashes, copies the API key,
     * creates an HttpClient with a 10-second connect timeout, and initializes an
     * ObjectMapper configured to ignore unknown JSON properties.
     *
     * @param builder builder containing configuration values (baseUrl and apiKey)
     */
    private SfVoiceMediaClient(Builder builder) {
        this.baseUrl = builder.baseUrl.replaceAll("/+$", "");
        this.apiKey  = builder.apiKey;
        this.http    = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(10))
            .build();
        this.json = new ObjectMapper()
            .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    }

    /**
     * Create an HttpRequest.Builder targeting the client's API path with standard headers and timeout.
     *
     * @param path the request path appended to the client's baseUrl (baseUrl has no trailing slash); should start with a leading '/'
     * @return an HttpRequest.Builder whose URI is baseUrl + path, with bearer authentication and a JSON content type set and a 30-second request timeout
     */

    private HttpRequest.Builder baseRequest(String path) {
        return HttpRequest.newBuilder()
            .uri(URI.create(baseUrl + path))
            .header("Authorization", "Bearer " + apiKey)
            .header("Content-Type", "application/json")
            .timeout(Duration.ofSeconds(30));
    }

    private static String pathSegment(String value) {
        return URLEncoder.encode(value, StandardCharsets.UTF_8).replace("+", "%20");
    }

    /**
     * Send the given HTTP request, validate the response status, and deserialize the response body.
     *
     * @param request      the HTTP request to send (must include URI, headers and method)
     * @param responseType the class to deserialize a successful JSON response into
     * @return the deserialized response object, or `null` if the server returned 204 No Content
     * @throws SfVoiceMediaException if the response status is not in the 200–299 range; the exception's code and message
     *                               will reflect parsed `error.code` and `error.message` from the response body when present
     * @throws IOException          if an I/O error occurs sending the request or deserializing the response
     * @throws InterruptedException if the thread is interrupted while waiting for the HTTP response
     */
    private <T> T execute(HttpRequest request, Class<T> responseType)
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpResponse<String> response = http.send(request, HttpResponse.BodyHandlers.ofString());

        if (response.statusCode() == 204 || responseType == Void.class) {
            return null;
        }

        if (response.statusCode() < 200 || response.statusCode() >= 300) {
            String code    = "http_error";
            String message = "request failed with status " + response.statusCode();
            try {
                var node = json.readTree(response.body());
                var err  = node.path("error");
                if (!err.isMissingNode()) {
                    code    = err.path("code").asText(code);
                    message = err.path("message").asText(message);
                }
            } catch (Exception ignored) {}
            throw new SfVoiceMediaException(code, message, response.statusCode());
        }

        return json.readValue(response.body(), responseType);
    }

    /**
     * Serialize an object to its JSON representation.
     *
     * @param body the object to serialize
     * @return the JSON string representation of the given object
     * @throws RuntimeException if serialization fails
     */
    private String toJson(Object body) {
        try {
            return json.writeValueAsString(body);
        } catch (IOException e) {
            throw new RuntimeException("failed to serialise request body", e);
        }
    }

    // ── public API ────────────────────────────────────────────────────────

    /**
     * Submit a media file for ingestion and receive a task to track its processing.
     *
     * @param request the ingestion request payload containing the media reference and options
     * @return an IngestResponse containing the created task ID and related ingestion metadata
     * @throws SfVoiceMediaException if the API responds with an error status
     * @throws IOException if an I/O error occurs sending the request or reading the response
     * @throws InterruptedException if the operation is interrupted while waiting for the HTTP response
     */
    public IngestResponse ingest(IngestRequest request)
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpRequest req = baseRequest("/api/v1/ingest")
            .POST(HttpRequest.BodyPublishers.ofString(toJson(request)))
            .build();
        return execute(req, IngestResponse.class);
    }

    /**
     * Fetches the current state of an ingestion task.
     *
     * @param taskId the identifier of the task to retrieve
     * @return the Task representing the task's current state
     * @throws SfVoiceMediaException if the API returns an error (non-2xx) — contains error code, message, and HTTP status
     * @throws IOException if a network or JSON I/O error occurs
     * @throws InterruptedException if the calling thread is interrupted while sending the request
     */
    public Task getTask(String taskId)
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpRequest req = baseRequest("/api/v1/tasks/" + pathSegment(taskId))
            .GET()
            .build();
        return execute(req, Task.class);
    }

    /**
     * Retrieve a paginated list of assets.
     *
     * @param page  page number to retrieve (1-based)
     * @param limit maximum number of assets to return on the page
     * @return the assets and pagination metadata for the requested page
     * @throws SfVoiceMediaException if the API returns an error response
     * @throws IOException if an I/O error occurs while sending the request
     * @throws InterruptedException if the thread is interrupted while waiting for the response
     */
    public AssetListResponse listAssets(int page, int limit)
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpRequest req = baseRequest("/api/v1/assets?page=" + page + "&limit=" + limit)
            .GET()
            .build();
        return execute(req, AssetListResponse.class);
    }

    /**
     * Retrieve a page of assets using the server's default pagination (page=1, limit=20).
     *
     * @return the AssetListResponse containing assets for the default page and limit
     * @throws SfVoiceMediaException if the API responds with an error status
     * @throws IOException if an I/O error occurs while sending the request
     * @throws InterruptedException if the operation is interrupted while waiting for a response
     */
    public AssetListResponse listAssets()
            throws SfVoiceMediaException, IOException, InterruptedException {
        return listAssets(1, 20);
    }

    /**
     * Retrieve an asset by its ID.
     *
     * @param assetId the asset identifier
     * @return the Asset corresponding to the given assetId
     * @throws SfVoiceMediaException if the API returns a non-2xx HTTP status
     * @throws IOException if an I/O error occurs while sending the request or reading the response
     * @throws InterruptedException if the thread is interrupted while waiting for the HTTP response
     */
    public Asset getAsset(String assetId)
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpRequest req = baseRequest("/api/v1/assets/" + pathSegment(assetId))
            .GET()
            .build();
        return execute(req, Asset.class);
    }

    /**
     * Soft-delete the asset with the given ID; completes when the server acknowledges deletion.
     *
     * The backend retains the record but excludes it from list results. The call completes
     * when the server returns HTTP 204.
     *
     * @param assetId the ID of the asset to soft-delete
     * @throws SfVoiceMediaException if the server responds with an error status
     * @throws IOException if an I/O error occurs sending the request or reading the response
     * @throws InterruptedException if the calling thread is interrupted while sending the request
     */
    public void deleteAsset(String assetId)
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpRequest req = baseRequest("/api/v1/assets/" + pathSegment(assetId))
            .DELETE()
            .build();
        execute(req, Void.class);
    }

    /**
     * Performs a semantic search over indexed media using the provided search parameters.
     *
     * @param request the search request describing the query and any search options
     * @return the search results and associated metadata
     * @throws SfVoiceMediaException if the API responds with an error status or error payload
     * @throws IOException if a network or I/O error occurs while sending the request or reading the response
     * @throws InterruptedException if the thread is interrupted while sending the HTTP request
     */
    public SearchResponse search(SearchRequest request)
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpRequest req = baseRequest("/api/v1/search")
            .POST(HttpRequest.BodyPublishers.ofString(toJson(request)))
            .build();
        return execute(req, SearchResponse.class);
    }

    /** List native transcript prefixes visible to this API key. */
    public PrefixListResponse listPrefixes()
            throws SfVoiceMediaException, IOException, InterruptedException {
        HttpRequest req = baseRequest("/api/v1/prefixes")
            .GET()
            .build();
        return execute(req, PrefixListResponse.class);
    }

    // ── monitors ──────────────────────────────────────────────────────────

    /**
     * Create a new monitor.
     *
     * @param request the monitor creation payload
     * @return the created Monitor
     * @throws SfVoiceMediaException if the API responds with an error status
     * @throws IOException if an I/O error occurs sending the request or reading the response
     * @throws InterruptedException if the operation is interrupted while waiting for the HTTP response
     */
    public Monitor createMonitor(CreateMonitorRequest request)
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpRequest req = baseRequest("/api/v1/monitors")
            .POST(HttpRequest.BodyPublishers.ofString(toJson(request)))
            .build();
        return execute(req, Monitor.class);
    }

    /**
     * List all monitors.
     *
     * @return a MonitorListResponse containing the monitors and total count
     * @throws SfVoiceMediaException if the API responds with an error status
     * @throws IOException if an I/O error occurs sending the request or reading the response
     * @throws InterruptedException if the operation is interrupted while waiting for the HTTP response
     */
    public MonitorListResponse listMonitors()
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpRequest req = baseRequest("/api/v1/monitors")
            .GET()
            .build();
        return execute(req, MonitorListResponse.class);
    }

    /**
     * Retrieve a monitor by its ID.
     *
     * @param monitorId the monitor identifier
     * @return the Monitor corresponding to the given monitorId
     * @throws SfVoiceMediaException if the API returns a non-2xx HTTP status
     * @throws IOException if an I/O error occurs sending the request or reading the response
     * @throws InterruptedException if the thread is interrupted while waiting for the HTTP response
     */
    public Monitor getMonitor(String monitorId)
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpRequest req = baseRequest("/api/v1/monitors/" + pathSegment(monitorId))
            .GET()
            .build();
        return execute(req, Monitor.class);
    }

    /**
     * Partially update an existing monitor.
     *
     * @param monitorId the monitor identifier
     * @param request   the fields to update (null fields are omitted from the request body)
     * @return the updated Monitor
     * @throws SfVoiceMediaException if the API responds with an error status
     * @throws IOException if an I/O error occurs sending the request or reading the response
     * @throws InterruptedException if the operation is interrupted while waiting for the HTTP response
     */
    public Monitor updateMonitor(String monitorId, UpdateMonitorRequest request)
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpRequest req = baseRequest("/api/v1/monitors/" + pathSegment(monitorId))
            .method("PATCH", HttpRequest.BodyPublishers.ofString(toJson(request)))
            .build();
        return execute(req, Monitor.class);
    }

    /**
     * Delete a monitor by its ID.
     *
     * @param monitorId the monitor identifier
     * @throws SfVoiceMediaException if the API responds with an error status
     * @throws IOException if an I/O error occurs sending the request or reading the response
     * @throws InterruptedException if the thread is interrupted while waiting for the HTTP response
     */
    public void deleteMonitor(String monitorId)
            throws SfVoiceMediaException, IOException, InterruptedException {

        HttpRequest req = baseRequest("/api/v1/monitors/" + pathSegment(monitorId))
            .DELETE()
            .build();
        execute(req, Void.class);
    }

    /**
     * List events for a monitor with filtering and pagination.
     *
     * @param monitorId   the monitor identifier
     * @param matchedOnly when true, only return events where matched is true
     * @param limit       maximum number of events to return
     * @param offset      number of events to skip
     * @return a MonitorEventListResponse containing the events and total count
     * @throws SfVoiceMediaException if the API responds with an error status
     * @throws IOException if an I/O error occurs sending the request or reading the response
     * @throws InterruptedException if the thread is interrupted while waiting for the HTTP response
     */
    public MonitorEventListResponse listMonitorEvents(
            String monitorId, boolean matchedOnly, int limit, int offset)
            throws SfVoiceMediaException, IOException, InterruptedException {

        String path = "/api/v1/monitors/" + pathSegment(monitorId) + "/events"
            + "?matched_only=" + matchedOnly
            + "&limit=" + limit
            + "&offset=" + offset;
        HttpRequest req = baseRequest(path)
            .GET()
            .build();
        return execute(req, MonitorEventListResponse.class);
    }

    // ── alert convenience ────────────────────────────────────────────────

    /**
     * Create a monitor and begin polling its events in a background thread.
     * The callback fires for each new matched event. Returns an AlertHandle
     * that can stop polling and clean up the monitor.
     *
     * @param text     the monitor text (what to watch for)
     * @param callback invoked on the polling thread for each new matched event
     * @param opts     optional alert configuration (slug, projectId, assetClass, threshold, intervalMs)
     * @return an AlertHandle to control and stop the background poll
     * @throws SfVoiceMediaException if creating the monitor fails
     * @throws IOException if an I/O error occurs creating the monitor
     * @throws InterruptedException if the thread is interrupted while creating the monitor
     */
    public AlertHandle alert(String text, Consumer<MonitorEvent> callback, AlertOptions opts)
            throws SfVoiceMediaException, IOException, InterruptedException {

        CreateMonitorRequest.Builder builder = CreateMonitorRequest.text(text);
        if (opts != null) {
            if (opts.slug != null)       builder.slug(opts.slug);
            if (opts.projectId != null)  builder.projectId(opts.projectId);
            if (opts.assetClass != null) builder.assetClass(opts.assetClass);
            if (opts.threshold != null)  builder.threshold(opts.threshold);
        }

        Monitor monitor = createMonitor(builder.build());
        long interval = (opts != null && opts.intervalMs > 0) ? opts.intervalMs : 5000;

        AlertHandle handle = new AlertHandle(monitor.getId(), this);
        Thread poller = new Thread(() -> {
            String checkpoint = null;
            int consecutiveFailures = 0;
            while (!handle.stopped) {
                try {
                    NewMonitorEvents batch = collectNewMonitorEvents(handle.monitorId, checkpoint);
                    consecutiveFailures = 0;
                    for (MonitorEvent event : batch.events) {
                        if (handle.stopped) break;
                        try {
                            callback.accept(event);
                        } catch (RuntimeException callbackError) {
                            handle.failure = new IllegalStateException(
                                "alert callback failed", callbackError);
                            return;
                        }
                    }
                    checkpoint = batch.checkpoint != null ? batch.checkpoint : checkpoint;
                    Thread.sleep(interval);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    break;
                } catch (Exception e) {
                    if (handle.stopped) break;
                    consecutiveFailures++;
                    boolean retryable = !(e instanceof SfVoiceMediaException) ||
                        ((SfVoiceMediaException) e).getStatus() == 429 ||
                        ((SfVoiceMediaException) e).getStatus() >= 500;
                    if (!retryable || consecutiveFailures >= 3) {
                        handle.failure = new IllegalStateException("alert polling failed", e);
                        break;
                    }
                    try { Thread.sleep(interval); } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        break;
                    }
                }
            }
        }, "sf-voice-alert-" + monitor.getId());
        poller.setDaemon(true);
        poller.start();

        handle.thread = poller;
        return handle;
    }

    private NewMonitorEvents collectNewMonitorEvents(String monitorId, String checkpoint)
            throws SfVoiceMediaException, IOException, InterruptedException {
        final int limit = 100;
        int offset = 0;
        String nextCheckpoint = null;
        List<MonitorEvent> pending = new ArrayList<>();
        Set<String> seen = new HashSet<>();

        while (true) {
            MonitorEventListResponse response = listMonitorEvents(monitorId, true, limit, offset);
            if (nextCheckpoint == null && !response.getItems().isEmpty()) {
                nextCheckpoint = response.getItems().get(0).getId();
            }
            for (MonitorEvent event : response.getItems()) {
                if (event.getId().equals(checkpoint)) {
                    Collections.reverse(pending);
                    return new NewMonitorEvents(pending, nextCheckpoint);
                }
                if (seen.add(event.getId())) pending.add(event);
            }
            offset += response.getItems().size();
            if (response.getItems().size() < limit || offset >= response.getTotal()) {
                Collections.reverse(pending);
                return new NewMonitorEvents(pending, nextCheckpoint);
            }
        }
    }

    private static class NewMonitorEvents {
        private final List<MonitorEvent> events;
        private final String checkpoint;

        private NewMonitorEvents(List<MonitorEvent> events, String checkpoint) {
            this.events = events;
            this.checkpoint = checkpoint;
        }
    }

    /**
     * Options for {@link #alert(String, Consumer, AlertOptions)}.
     */
    public static class AlertOptions {
        public String slug;
        public String projectId;
        public String assetClass;
        public Float threshold;
        /** poll interval in milliseconds, defaults to 5000 */
        public long intervalMs = 5000;
    }

    /**
     * Handle returned by {@link #alert(String, Consumer, AlertOptions)} to
     * control the background polling thread and clean up the monitor.
     */
    public static class AlertHandle {
        private final String monitorId;
        private final SfVoiceMediaClient client;
        private volatile boolean stopped = false;
        private volatile RuntimeException failure;
        private Thread thread;

        AlertHandle(String monitorId, SfVoiceMediaClient client) {
            this.monitorId = monitorId;
            this.client    = client;
        }

        public String getMonitorId() { return monitorId; }

        /**
         * Stop polling and delete the monitor. Safe to call multiple times.
         */
        public synchronized void stop() {
            if (stopped) {
                if (failure != null) throw failure;
                return;
            }
            stopped = true;
            if (thread != null) {
                thread.interrupt();
                try { thread.join(); } catch (InterruptedException ignored) {
                    Thread.currentThread().interrupt();
                }
            }
            try {
                client.deleteMonitor(monitorId);
            } catch (Exception e) {
                failure = new IllegalStateException("failed to delete alert monitor", e);
            }
            if (failure != null) throw failure;
        }
    }

    // ── poll task ─────────────────────────────────────────────────────────

    /**
     * Waits until the specified task reaches a terminal state.
     *
     * Polls the task state at up to `intervalMs` millisecond intervals until the task is terminal
     * or `timeoutMs` milliseconds have elapsed.
     *
     * @param taskId     the identifier of the task to poll
     * @param intervalMs poll interval in milliseconds between successive checks
     * @param timeoutMs  maximum total time in milliseconds to wait before giving up
     * @return the final {@link Task} when it reaches a terminal state
     * @throws SfVoiceMediaException if the task does not reach a terminal state before the timeout
     * @throws IOException if fetching the task state fails due to I/O or response parsing errors
     * @throws InterruptedException if the thread is interrupted while sleeping between polls
     */
    public Task pollTask(String taskId, long intervalMs, long timeoutMs)
            throws SfVoiceMediaException, IOException, InterruptedException {

        long deadline = System.currentTimeMillis() + timeoutMs;

        while (true) {
            Task task = getTask(taskId);
            if (task.isTerminal()) return task;

            long remaining = deadline - System.currentTimeMillis();
            if (remaining <= 0) {
                throw new SfVoiceMediaException(
                    "poll_timeout",
                    "task " + taskId + " did not complete within " + timeoutMs + "ms",
                    0
                );
            }
            Thread.sleep(Math.min(intervalMs, remaining));
        }
    }

    // ── builder ───────────────────────────────────────────────────────────

    public static class Builder {
        private String apiKey;
        private String baseUrl = "https://api.sf-voice.com";

        /**
 * Sets the API key used for bearer authentication.
 *
 * @param apiKey the API key value; must be non-null and not blank when building the client
 * @return this Builder instance
 */
public Builder apiKey(String apiKey)   { this.apiKey = apiKey; return this; }
        /**
 * Sets the API base URL the client will use.
 *
 * The URL should be a full URL including scheme (for example, "https://api.sf-voice.com"); trailing slashes will be normalized by the client constructor.
 *
 * @param baseUrl the base URL for API requests
 * @return this builder
 */
public Builder baseUrl(String baseUrl) { this.baseUrl = baseUrl; return this; }

        /**
         * Builds an SfVoiceMediaClient configured with the values in this Builder.
         *
         * @return a new SfVoiceMediaClient configured with this Builder
         * @throws IllegalStateException if the required `apiKey` is null or blank
         */
        public SfVoiceMediaClient build() {
            if (apiKey == null || apiKey.isBlank()) {
                throw new IllegalStateException("apiKey must be set");
            }
            return new SfVoiceMediaClient(this);
        }
    }
}
