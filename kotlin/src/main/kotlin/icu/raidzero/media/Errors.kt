package icu.raidzero.media

/** thrown by every client method on a non-2xx API response. */
class RaidzeroMediaException(
    /** machine-readable error code from the API. */
    val code: String,
    message: String,
    /** HTTP status code. */
    val status: Int,
) : Exception(message)

/** thrown by [RaidzeroMediaClient.pollTask] when the timeout elapses. */
class RaidzeroMediaPollTimeoutException(
    val taskId: String,
    timeoutMs: Long,
) : Exception("task $taskId did not complete within ${timeoutMs}ms")
