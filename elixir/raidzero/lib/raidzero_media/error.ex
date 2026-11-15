defmodule RaidzeroMedia.Error do
  @moduledoc """
  structured error returned by every SDK function on failure.

  raised as an exception by `RaidzeroMedia.poll_task/3` on timeout or task failure;
  returned inside `{:error, %RaidzeroMedia.Error{}}` tuples by all other functions.

  fields:
  - `code`    — machine-readable error code string from the API (e.g. `"not_found"`)
  - `message` — human-readable description
  - `status`  — http status code, or `nil` when the error is client-side (e.g. poll timeout)
  """

  defexception [:code, :message, :status]

  @type t :: %__MODULE__{
          code: String.t(),
          message: String.t(),
          status: non_neg_integer() | nil
        }

  @impl true
  @doc """
  Format the exception into a human-readable string.

  Formats an `%RaidzeroMedia.Error{}` as `"[<code>] <message> (HTTP <status>)"`.
  """
  @spec message(t()) :: String.t()
  def message(%__MODULE__{code: code, message: msg, status: status}) do
    "[#{code}] #{msg} (HTTP #{status})"
  end

  # ── constructors ─────────────────────────────────────────────────────────────

  @doc """
  Builds a RaidzeroMedia.Error from the API error envelope.

  Parses a response body with the shape `%{"error" => %{"code" => code, "message" => message}}`
  and returns an `%RaidzeroMedia.Error{}` populated with `code`, `message`, and the given HTTP `status`.
  If the body does not match that shape, the function falls back to a generic HTTP error.
  `message: "request failed with status <status>"`, and `status` set to the given status.

  ## Parameters

    - status: HTTP status code returned by the request.
    - body: Parsed response body (map) or `nil`.

  """
  @spec from_response(non_neg_integer(), map() | nil) :: t()
  def from_response(status, %{"error" => %{"code" => code, "message" => msg}}) do
    %__MODULE__{code: code, message: msg, status: status}
  end

  def from_response(status, _body) do
    %__MODULE__{
      code: "http_error",
      message: "request failed with status #{status}",
      status: status
    }
  end

  @doc """
  Create a client-side timeout error for a polling task.

  The returned `%RaidzeroMedia.Error{}` has `code: "poll_timeout"`, a `message` that includes the `task_id` and `timeout_ms`, and `status: nil`.
  """
  @spec poll_timeout(String.t(), non_neg_integer()) :: t()
  def poll_timeout(task_id, timeout_ms) do
    %__MODULE__{
      code: "poll_timeout",
      message: "task #{task_id} did not complete within #{timeout_ms}ms",
      status: nil
    }
  end
end
