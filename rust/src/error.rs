//! error types for the sf-voice-media SDK.
//! all non-2xx API responses surface as `SfVoiceMediaError::Api`.
//! network-level failures wrap as `SfVoiceMediaError::Http`.

use thiserror::Error;

/// the raw error envelope the API returns on non-2xx responses.
#[derive(Debug, serde::Deserialize)]
struct ApiErrorEnvelope {
    error: ApiErrorBody,
}

#[derive(Debug, serde::Deserialize)]
struct ApiErrorBody {
    code: String,
    message: String,
}

/// every fallible SDK operation returns `Result<T, SfVoiceMediaError>`.
#[derive(Debug, Error)]
pub enum SfVoiceMediaError {
    /// the API returned a non-2xx status with a structured error body.
    #[error("api error {status} [{code}]: {message}")]
    Api {
        /// machine-readable error code from the API (e.g. "not_found").
        code: String,
        /// human-readable description.
        message: String,
        /// HTTP status code.
        status: u16,
    },

    /// the underlying HTTP transport failed (dns, timeout, tls, etc.).
    #[error("http error: {0}")]
    Http(#[from] reqwest::Error),

    /// the background alert polling task failed unexpectedly.
    #[error("background task failed: {0}")]
    BackgroundTask(#[from] tokio::task::JoinError),

    /// alert event pagination exceeded the safety bound.
    #[error("alert event pagination exceeded 1000 pages")]
    AlertPaginationOverflow,

    /// both alert polling and monitor cleanup failed.
    #[error("alert polling failed ({polling}); monitor cleanup also failed ({cleanup})")]
    AlertCleanup {
        /// polling or callback task error.
        polling: String,
        /// monitor deletion error.
        cleanup: String,
    },

    /// `poll_task` exhausted its timeout before the task reached a terminal state.
    #[error("task {task_id} did not complete within the timeout")]
    PollTimeout {
        /// the task ID that was being polled.
        task_id: String,
    },
}

impl SfVoiceMediaError {
    /// Constructs an `SfVoiceMediaError::Api` from an HTTP status code and raw response body.
    ///
    /// Attempts to interpret `body` as the API's structured error envelope; if parsing fails or the
    /// body is empty/malformed, produces a generic `Api` error whose `message` is the body decoded
    /// lossy as UTF-8 (or `"request failed with status {status}"` when that result is empty).
    ///
    /// # Examples
    ///
    /// ```ignore
    /// let env = br#"{"error":{"code":"bad_request","message":"invalid input"}}"#;
    /// let err = SfVoiceMediaError::from_response(400, env);
    /// match err {
    ///     SfVoiceMediaError::Api { code, message, status } => {
    ///         assert_eq!(code, "bad_request");
    ///         assert_eq!(message, "invalid input");
    ///         assert_eq!(status, 400);
    ///     }
    ///     _ => panic!("expected Api variant"),
    /// }
    ///
    /// let plain = b"plain text error";
    /// let err2 = SfVoiceMediaError::from_response(500, plain);
    /// match err2 {
    ///     SfVoiceMediaError::Api { code, message, status } => {
    ///         assert_eq!(code, "http_error");
    ///         assert_eq!(message, "plain text error");
    ///         assert_eq!(status, 500);
    ///     }
    ///     _ => panic!("expected Api variant"),
    /// }
    /// ```
    pub(crate) fn from_response(status: u16, body: &[u8]) -> Self {
        if let Ok(env) = serde_json::from_slice::<ApiErrorEnvelope>(body) {
            return Self::Api {
                code: env.error.code,
                message: env.error.message,
                status,
            };
        }

        // body was not the expected envelope — synthesize a generic error
        let message = String::from_utf8_lossy(body).into_owned();
        Self::Api {
            code: "http_error".into(),
            message: if message.is_empty() {
                format!("request failed with status {status}")
            } else {
                message
            },
            status,
        }
    }
}
