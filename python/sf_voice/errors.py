"""
error types for the sf-voice media sdk.
"""

from __future__ import annotations

from typing import Any, Dict, Optional


class SfVoiceMediaError(Exception):
    """raised on any non-2xx response from the api."""

    def __init__(self, code: str, message: str, status: int) -> None:
        """store the api error code, message, and http status."""
        super().__init__(message)
        self.code = code
        self.message = message
        self.status = status

    @classmethod
    def from_response(
        cls, status: int, body: Optional[Dict[str, Any]]
    ) -> "SfVoiceMediaError":
        """build an sdk error from the standard api error envelope."""
        if isinstance(body, dict) and isinstance(body.get("error"), dict):
            error = body["error"]
            return cls(
                code=error.get("code", "provider_unavailable"),
                message=error.get("message", f"request failed with status {status}"),
                status=status,
            )

        return cls(
            code="provider_unavailable",
            message=f"request failed with status {status}",
            status=status,
        )

    def __repr__(self) -> str:
        """return a detailed representation for debugging."""
        return (
            f"SfVoiceMediaError(status={self.status}, "
            f"code={self.code!r}, message={self.message!r})"
        )


class SfVoiceMediaRequestTimeoutError(TimeoutError):
    """raised when one http request exceeds the client timeout."""

    def __init__(self, timeout_ms: int) -> None:
        """store the request timeout in milliseconds."""
        super().__init__(f"request timed out after {timeout_ms}ms")
        self.timeout_ms = timeout_ms


class SfVoiceMediaPollTimeoutError(TimeoutError):
    """raised when poll_task reaches its total timeout."""

    def __init__(self, task_id: str, timeout_ms: int) -> None:
        """store the task id and total polling timeout."""
        super().__init__(f"task {task_id} did not complete within {timeout_ms}ms")
        self.task_id = task_id
        self.timeout_ms = timeout_ms


class SfVoiceMediaAlertCallbackError(RuntimeError):
    """raised when an alert callback fails."""
