package com.sfvoice.media

import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.engine.cio.*
import io.ktor.client.plugins.defaultRequest
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.*
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.json.Json
import java.util.concurrent.atomic.AtomicReference

/**
 * coroutine-based client for the sf-voice media API.
 *
 * ```kotlin
 * val client = SfVoiceMediaClient(
 *     apiKey = System.getenv("SF_VOICE_API_KEY"),
 *     baseUrl = "https://api.sf-voice.com",
 * )
 * ```
 *
 * call [close] when done to release the underlying HTTP connection pool.
 */
class SfVoiceMediaClient(
    private val apiKey: String,
    baseUrl: String = "https://api.sf-voice.com",
) : AutoCloseable {

    private val baseUrl = baseUrl.trimEnd('/')

    private val json = Json {
        ignoreUnknownKeys = true
        encodeDefaults = false
    }

    private val http = HttpClient(CIO) {
        defaultRequest {
            bearerAuth(apiKey)
        }
        install(ContentNegotiation) {
            json(json)
        }
    }

    /**
     * Throws an SfVoiceMediaException when the HTTP response status indicates an error.
     *
     * If the response body can be decoded as an `ApiErrorEnvelope`, the exception's `code`
     * and `message` are taken from that envelope; otherwise the exception uses
     * HTTP-derived defaults (`"http_error"` and a status-based message).
     *
     * @throws SfVoiceMediaException when `status.isSuccess()` is false.
     */

    private suspend fun HttpResponse.throwIfError() {
        if (status.isSuccess()) return
        val body = runCatching { bodyAsText() }.getOrNull()
        val envelope = body?.let {
            runCatching { json.decodeFromString<ApiErrorEnvelope>(it) }.getOrNull()
        }
        throw SfVoiceMediaException(
            code = envelope?.error?.code ?: "http_error",
            message = envelope?.error?.message ?: "request failed with status ${status.value}",
            status = status.value,
        )
    }

    // ── public API ─────────────────────────────────────────────────────────

    /**
     * Submit a media file for ingestion.
     *
     * Initiates ingestion and returns the created task information immediately.
     *
     * @param request Details of the media to ingest and any ingestion options.
     * @return An IngestResponse containing the created task's ID and associated metadata.
     */
    suspend fun ingest(request: IngestRequest): IngestResponse {
        val response = http.post("$baseUrl/api/v1/ingest") {
            contentType(ContentType.Application.Json)
            setBody(request.normalized())
        }
        response.throwIfError()
        return response.body()
    }

    /**
     * Fetches the current state of an ingestion task.
     *
     * @param taskId The ID of the task to retrieve.
     * @return The requested Task containing its current status and metadata.
     */
    suspend fun getTask(taskId: String): Task {
        val response = http.get("$baseUrl/api/v1/tasks/${taskId.encodeURLPathPart()}") {
        }
        response.throwIfError()
        return response.body()
    }

    /**
     * Lists assets in the library with pagination.
     *
     * @param page Page number starting at 1.
     * @param limit Number of items to return per page.
     * @return An AssetListResponse containing the requested page of assets.
     */
    suspend fun listAssets(page: Int = 1, limit: Int = 20): AssetListResponse {
        val response = http.get("$baseUrl/api/v1/assets") {
            parameter("page", page)
            parameter("limit", limit)
        }
        response.throwIfError()
        return response.body()
    }

    /**
     * Retrieve an asset by its identifier.
     *
     * @param assetId The ID of the asset to fetch.
     * @return The asset with the specified ID.
     */
    suspend fun getAsset(assetId: String): Asset {
        val response = http.get("$baseUrl/api/v1/assets/${assetId.encodeURLPathPart()}") {
        }
        response.throwIfError()
        return response.body()
    }

    /**
     * Soft-delete the asset identified by [assetId], retaining the record but excluding it from list results.
     *
     * Completes successfully when the server responds with HTTP 204.
     *
     * @param assetId The identifier of the asset to soft-delete.
     * @throws SfVoiceMediaException if the HTTP response indicates an error.
     */
    suspend fun deleteAsset(assetId: String) {
        val response = http.delete("$baseUrl/api/v1/assets/${assetId.encodeURLPathPart()}") {
        }
        response.throwIfError()
    }

    /**
     * Performs a semantic search over indexed media.
     *
     * @param request The search query and any filters or pagination parameters to apply.
     * @return A SearchResponse containing matching results and associated metadata.
     */
    suspend fun search(request: SearchRequest): SearchResponse {
        val response = http.post("$baseUrl/api/v1/search") {
            contentType(ContentType.Application.Json)
            setBody(request)
        }
        response.throwIfError()
        return response.body()
    }

    /** List native transcript prefixes visible to this API key. */
    suspend fun listPrefixes(): PrefixListResponse {
        val response = http.get("$baseUrl/api/v1/prefixes")
        response.throwIfError()
        return response.body()
    }

    /**
     * Waits until the specified task reaches a terminal status and returns its final state.
     *
     * @param taskId The ID of the task to poll.
     * @param intervalMs Milliseconds between polls.
     * @param timeoutMs Maximum total wait time in milliseconds before giving up.
     * @return The final `Task` whose `status` is terminal.
     * @throws SfVoiceMediaPollTimeoutException if the timeout elapses before the task becomes terminal.
     */
    suspend fun pollTask(
        taskId: String,
        intervalMs: Long = 1_500L,
        timeoutMs: Long = 120_000L,
    ): Task {
        val deadline = System.currentTimeMillis() + timeoutMs

        while (true) {
            val task = getTask(taskId)
            if (task.status.isTerminal) return task

            val remaining = deadline - System.currentTimeMillis()
            if (remaining <= 0) throw SfVoiceMediaPollTimeoutException(taskId, timeoutMs)

            delay(minOf(intervalMs, remaining))
        }
    }

    // ── monitors ────────────────────────────────────────────────────────────

    /**
     * Creates a new monitor that watches for content matching the given text.
     *
     * @param request The monitor definition including search text and optional filters.
     * @return The created Monitor.
     */
    suspend fun createMonitor(request: CreateMonitorRequest): Monitor {
        val response = http.post("$baseUrl/api/v1/monitors") {
            contentType(ContentType.Application.Json)
            setBody(request)
        }
        response.throwIfError()
        return response.body()
    }

    /**
     * Lists all monitors owned by this API key.
     *
     * @return A MonitorListResponse containing the monitors and total count.
     */
    suspend fun listMonitors(): MonitorListResponse {
        val response = http.get("$baseUrl/api/v1/monitors") {
        }
        response.throwIfError()
        return response.body()
    }

    /**
     * Retrieves a single monitor by its identifier.
     *
     * @param monitorId The ID of the monitor to fetch.
     * @return The requested Monitor.
     */
    suspend fun getMonitor(monitorId: String): Monitor {
        val response = http.get("$baseUrl/api/v1/monitors/${monitorId.encodeURLPathPart()}") {
        }
        response.throwIfError()
        return response.body()
    }

    /**
     * Partially updates an existing monitor.
     *
     * Only fields present in [request] are modified; omitted fields remain unchanged.
     *
     * @param monitorId The ID of the monitor to update.
     * @param request The fields to patch.
     * @return The updated Monitor.
     */
    suspend fun updateMonitor(monitorId: String, request: UpdateMonitorRequest): Monitor {
        val response = http.patch("$baseUrl/api/v1/monitors/${monitorId.encodeURLPathPart()}") {
            contentType(ContentType.Application.Json)
            setBody(request)
        }
        response.throwIfError()
        return response.body()
    }

    /**
     * Deletes a monitor by its identifier.
     *
     * @param monitorId The ID of the monitor to delete.
     */
    suspend fun deleteMonitor(monitorId: String) {
        val response = http.delete("$baseUrl/api/v1/monitors/${monitorId.encodeURLPathPart()}") {
        }
        response.throwIfError()
    }

    /**
     * Lists events for a monitor with optional filtering and pagination.
     *
     * @param monitorId The ID of the monitor whose events to list.
     * @param matchedOnly When true, only return events where matched is true.
     * @param limit Maximum number of events to return.
     * @param offset Number of events to skip for pagination.
     * @return A MonitorEventListResponse containing events and total count.
     */
    suspend fun listMonitorEvents(
        monitorId: String,
        matchedOnly: Boolean = false,
        limit: Int = 50,
        offset: Int = 0,
    ): MonitorEventListResponse {
        val response = http.get("$baseUrl/api/v1/monitors/${monitorId.encodeURLPathPart()}/events") {
            parameter("matched_only", matchedOnly)
            parameter("limit", limit)
            parameter("offset", offset)
        }
        response.throwIfError()
        return response.body()
    }

    /**
     * Creates a monitor and polls it for matching events in the background.
     *
     * Returns an [AlertHandle] that can be used to stop polling and clean up the monitor.
     * The [callback] is invoked for each new matched event as it appears.
     *
     * ```kotlin
     * val handle = client.alert("breaking news about AI") { event ->
     *     println("match: score=${event.score}")
     * }
     * // ... later
     * handle.stop()
     * ```
     *
     * @param text The search text for the monitor.
     * @param callback Invoked for each new matched MonitorEvent.
     * @param slug Optional slug for the monitor.
     * @param projectId Optional project ID filter.
     * @param assetClass Optional asset class filter.
     * @param threshold Optional similarity threshold.
     * @param intervalMs Milliseconds between polls.
     * @return An AlertHandle to control the background polling.
     */
    suspend fun alert(
        text: String,
        callback: (MonitorEvent) -> Unit,
        slug: String? = null,
        projectId: String? = null,
        assetClass: String? = null,
        threshold: Float? = null,
        intervalMs: Long = 5_000L,
    ): AlertHandle {
        val monitor = createMonitor(
            CreateMonitorRequest(
                text = text,
                slug = slug,
                projectId = projectId,
                assetClass = assetClass,
                threshold = threshold,
            )
        )

        val failure = AtomicReference<Throwable?>()
        val job = CoroutineScope(SupervisorJob() + Dispatchers.IO).launch {
            var checkpoint: String? = null
            var consecutiveFailures = 0
            while (isActive) {
                delay(intervalMs)
                try {
                    val batch = collectNewMonitorEvents(monitor.id, checkpoint)
                    consecutiveFailures = 0
                    for (event in batch.events) {
                        ensureActive()
                        try {
                            callback(event)
                        } catch (error: Throwable) {
                            throw AlertCallbackException(error)
                        }
                    }
                    checkpoint = batch.checkpoint ?: checkpoint
                } catch (error: CancellationException) {
                    throw error
                } catch (error: Throwable) {
                    consecutiveFailures++
                    val retryable = error !is AlertCallbackException &&
                        (error !is SfVoiceMediaException || error.status == 429 || error.status >= 500)
                    if (!retryable || consecutiveFailures >= 3) {
                        failure.set(error)
                        return@launch
                    }
                }
            }
        }

        return AlertHandle(
            monitorId = monitor.id,
            job = job,
            client = this,
            failure = failure,
        )
    }

    private data class NewMonitorEvents(
        val events: List<MonitorEvent>,
        val checkpoint: String?,
    )

    private suspend fun collectNewMonitorEvents(
        monitorId: String,
        checkpoint: String?,
    ): NewMonitorEvents {
        val limit = 100
        var offset = 0
        var nextCheckpoint: String? = null
        val pending = mutableListOf<MonitorEvent>()
        val seen = mutableSetOf<String>()

        repeat(1_000) {
            val response = listMonitorEvents(monitorId, matchedOnly = true, limit = limit, offset = offset)
            nextCheckpoint = nextCheckpoint ?: response.items.firstOrNull()?.id
            for (event in response.items) {
                if (event.id == checkpoint) {
                    return NewMonitorEvents(pending.asReversed(), nextCheckpoint)
                }
                if (seen.add(event.id)) pending.add(event)
            }
            offset += response.items.size
            if (response.items.size < limit || offset.toLong() >= response.total) {
                return NewMonitorEvents(pending.asReversed(), nextCheckpoint)
            }
        }

        throw IllegalStateException("alert event pagination exceeded 1000 pages")
    }

    /**
     * Releases resources held by this client.
     *
     * Closes the underlying HTTP client, rendering this instance unusable for further requests.
     */
    override fun close() {
        http.close()
    }
}

private class AlertCallbackException(cause: Throwable) :
    RuntimeException("alert callback failed", cause)

/**
 * Handle returned by [SfVoiceMediaClient.alert] to control background polling.
 *
 * Call [stop] to cancel the polling coroutine and delete the underlying monitor.
 */
class AlertHandle(
    val monitorId: String,
    private val job: Job,
    private val client: SfVoiceMediaClient,
    private val failure: AtomicReference<Throwable?>,
) {
    private val stopMutex = Mutex()
    private var stopped = false
    private var stopError: Throwable? = null

    /**
     * Cancels background polling, deletes the monitor, and surfaces polling failures.
     */
    suspend fun stop() {
        stopMutex.withLock {
            if (stopped) {
                stopError?.let { throw it }
                return
            }

            stopped = true
            try {
                job.cancelAndJoin()
                client.deleteMonitor(monitorId)
                stopError = failure.get()
            } catch (error: Throwable) {
                stopError = error
            }
            stopError?.let { throw it }
        }
    }
}
