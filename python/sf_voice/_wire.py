"""
small wire-format helpers shared by the sync and async clients.
"""

from __future__ import annotations

import json
from typing import Any, Dict, Mapping, Tuple
from urllib.parse import quote

import httpx

from .errors import SfVoiceMediaError


def path_segment(value: str) -> str:
    """encode one path segment without allowing slashes through."""
    return quote(value, safe="")


def compact_dict(values: Mapping[str, Any]) -> Dict[str, Any]:
    """copy a mapping while dropping fields set to none."""
    return {key: value for key, value in values.items() if value is not None}


def normalize_ingest_request(request: Mapping[str, Any]) -> Dict[str, Any]:
    """map deprecated ingest names to the current wire contract."""
    normalized = compact_dict(request)
    legacy_media_type = normalized.pop("media_type", None)
    if "media_kind" not in normalized and legacy_media_type is not None:
        normalized["media_kind"] = legacy_media_type
    return normalized


def build_file_ingest_payload(
    request: Mapping[str, Any],
) -> Tuple[Dict[str, str], Dict[str, Tuple[str, Any, str]]]:
    """build the multipart body used for file ingest."""
    request = normalize_ingest_request(request)
    content_type = request.get("content_type") or "application/octet-stream"
    fields: Dict[str, str] = {}

    for key in (
        "source",
        "asset_id",
        "asset_class",
        "project_id",
        "media_kind",
        "content_type",
        "prefix",
    ):
        value = request.get(key)
        if value is not None:
            fields[key] = str(value)

    transcript_only = request.get("transcript_only")
    if transcript_only is not None:
        fields["transcript_only"] = str(bool(transcript_only)).lower()

    for key in ("metadata", "types"):
        value = request.get(key)
        if value is not None:
            fields[key] = json.dumps(value, separators=(",", ":"))

    files = {
        "file": (
            str(request["filename"]),
            _normalize_file_content(request["file"]),
            str(content_type),
        )
    }
    return fields, files


def decode_response(response: httpx.Response) -> Any:
    """decode a response body and raise sdk errors for api failures."""
    if response.status_code == 204:
        return None

    try:
        body = response.json()
    except ValueError as exc:
        raise SfVoiceMediaError(
            "provider_unavailable",
            f"unexpected non-JSON response from server (HTTP {response.status_code})",
            response.status_code,
        ) from exc

    if not response.is_success:
        raise SfVoiceMediaError.from_response(
            response.status_code,
            body if isinstance(body, dict) else None,
        )

    return body


def _normalize_file_content(value: Any) -> Any:
    """convert byte containers httpx does not accept directly."""
    if isinstance(value, memoryview):
        return value.tobytes()
    if isinstance(value, bytearray):
        return bytes(value)
    return value
