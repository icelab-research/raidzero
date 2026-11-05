defmodule SfVoiceMedia do
  @moduledoc """
  Elixir SDK for the sf-voice media API.

  build a client with `new/2`, then call the public functions to ingest,
  query, and search media.

  ## quick start

      client = SfVoiceMedia.new("sk-...")

      # 1. ingest a media file — returns immediately with a task id
      {:ok, %{task_id: tid}} =
        SfVoiceMedia.ingest(client, %{source: :url, url: "https://example.com/clip.mp4"})

      # 2. wait for indexing to complete — raises on failure or timeout
      task = SfVoiceMedia.poll_task!(client, tid)

      # 3. search across your indexed media with natural language
      {:ok, %{results: results}} =
        SfVoiceMedia.search(client, %{query: "product roadmap discussion"})

      # results carry timestamps so you can jump to the exact moment
      Enum.each(results, fn r ->
        IO.puts("\#{r.asset_id} at \#{r.start_ms}ms — \#{r.match_type}")
      end)

  all functions return `{:ok, result}` or `{:error, %SfVoiceMedia.Error{}}`.
  `poll_task!/3` raises `SfVoiceMedia.Error` on timeout or task failure.
  """

  alias SfVoiceMedia.{Client, Error}
  alias SfVoiceMedia.Types

  # ── construction ─────────────────────────────────────────────────────────────

  @doc """
  Builds a SfVoiceMedia.Client preconfigured with the given API key and optional settings.

  ## Options

    - `:base_url` — API base URL; defaults to "https://api.sf-voice.com". A trailing `/` is removed.
    - `:http_opts` — keyword list forwarded to Req for every request; defaults to `[]`.

  ## Examples

      client = SfVoiceMedia.new("sk-my-api-key")

      client = SfVoiceMedia.new("sk-my-api-key",
        base_url: "https://staging.api.sf-voice.com",
        http_opts: [receive_timeout: 10_000]
      )
  """
  @spec new(String.t(), keyword()) :: Client.t()
  def new(api_key, opts \\ []) do
    base_url =
      opts
      |> Keyword.get(:base_url, "https://api.sf-voice.com")
      |> String.trim_trailing("/")

    %Client{
      api_key: api_key,
      base_url: base_url,
      http_opts: Keyword.get(opts, :http_opts, [])
    }
  end

  # ── public API ───────────────────────────────────────────────────────────────

  @doc """
  Submit a media file for ingestion from a URL or an S3 key.

  Returns immediately with a task identifier that can be inspected with `get_task/2` or awaited with `poll_task/3`.

  ## Returns

    - `{:ok, response}` — successful response containing at least `task_id` and optionally `asset_id` and other task metadata.
    - `{:error, %SfVoiceMedia.Error{}}` — request failed; contains error details.
  """
  @spec ingest(Client.t(), Types.ingest_request()) ::
          {:ok, Types.ingest_response()} | {:error, Error.t()}
  def ingest(%Client{} = client, request) when is_map(request) do
    post(client, "/api/v1/ingest", request)
  end

  @doc """
  Fetches the current state of an ingestion task.

  Returns `{:ok, task}` where `task` is a map describing the task (includes a `"status"` field), or `{:error, %SfVoiceMedia.Error{}}` on failure.

  ## Examples

      {:ok, %{status: "ready", asset_id: aid}} = SfVoiceMedia.get_task(client, "task_abc123")
  """
  @spec get_task(Client.t(), String.t()) ::
          {:ok, Types.task()} | {:error, Error.t()}
  def get_task(%Client{} = client, task_id) when is_binary(task_id) do
    get(client, "/api/v1/tasks/#{path_segment(task_id)}")
  end

  @doc """
  lists assets in the library, paginated.

  ## examples

      {:ok, %{items: items, page_info: info}} =
        SfVoiceMedia.list_assets(client, %{page: 1, limit: 20})

      # no params — uses server defaults
      {:ok, %{items: items}} = SfVoiceMedia.list_assets(client)
  """
  @spec list_assets(Client.t(), Types.list_assets_params()) ::
          {:ok, Types.asset_list_response()} | {:error, Error.t()}
  def list_assets(%Client{} = client, params \\ %{}) when is_map(params) do
    qs = build_query(params)
    get(client, "/api/v1/assets#{qs}")
  end

  @doc """
  Retrieve a library asset by its ID.

  Returns `{:ok, asset}` when the asset is found, or `{:error, %SfVoiceMedia.Error{}}` on failure.
  """
  @spec get_asset(Client.t(), String.t()) ::
          {:ok, Types.asset()} | {:error, Error.t()}
  def get_asset(%Client{} = client, id) when is_binary(id) do
    get(client, "/api/v1/assets/#{path_segment(id)}")
  end

  @doc """
  Soft-deletes an asset so it is excluded from list results while the backend retains the record.

  Returns `:ok` if the deletion was successful (HTTP 204), `{:error, %SfVoiceMedia.Error{}}` otherwise.

  ## Examples

      :ok = SfVoiceMedia.delete_asset(client, "ast_abc123")
  """
  @spec delete_asset(Client.t(), String.t()) :: :ok | {:error, Error.t()}
  def delete_asset(%Client{} = client, id) when is_binary(id) do
    case request(client, :delete, "/api/v1/assets/#{path_segment(id)}") do
      {:ok, _} -> :ok
      {:error, _} = err -> err
    end
  end

  @doc """
  Run a semantic search over indexed media.

  The `request` map must include a `:query` string and may include optional parameters such as `:types` (list of asset types), `:threshold` (similarity threshold), and `:limit` (maximum results). Returns the API response wrapped in `{:ok, result}` or `{:error, %SfVoiceMedia.Error{}}`.

  ## Examples

      {:ok, %{results: results}} =
        SfVoiceMedia.search(client, %{query: "product roadmap discussion"})

      {:ok, %{results: results}} =
        SfVoiceMedia.search(client, %{
          query: "quarterly targets",
          types: [:conversation],
          threshold: 0.7,
          limit: 10
        })
  """
  @spec search(Client.t(), Types.search_request()) ::
          {:ok, Types.search_response()} | {:error, Error.t()}
  def search(%Client{} = client, request) when is_map(request) do
    post(client, "/api/v1/search", request)
  end

  @doc "List native transcript prefixes visible to this API key."
  @spec list_prefixes(Client.t()) :: {:ok, Types.prefix_list_response()} | {:error, Error.t()}
  def list_prefixes(%Client{} = client) do
    get(client, "/api/v1/prefixes")
  end

  # ── monitors ──────────────────────────────────────────────────────────────────

  @doc """
  Creates a monitor that watches for content matching the given text.

  ## Examples

      {:ok, monitor} =
        SfVoiceMedia.create_monitor(client, %{text: "product launch discussion"})

      {:ok, monitor} =
        SfVoiceMedia.create_monitor(client, %{
          text: "quarterly revenue",
          threshold: 0.8,
          asset_class: "earnings_call"
        })
  """
  @spec create_monitor(Client.t(), Types.create_monitor_request()) ::
          {:ok, Types.monitor()} | {:error, Error.t()}
  def create_monitor(%Client{} = client, request) when is_map(request) do
    post(client, "/api/v1/monitors", request)
  end

  @doc """
  Lists all monitors for the current API key.

  ## Examples

      {:ok, %{items: monitors, total: n}} = SfVoiceMedia.list_monitors(client)
  """
  @spec list_monitors(Client.t()) ::
          {:ok, Types.monitor_list_response()} | {:error, Error.t()}
  def list_monitors(%Client{} = client) do
    get(client, "/api/v1/monitors")
  end

  @doc """
  Fetches a single monitor by ID.

  ## Examples

      {:ok, monitor} = SfVoiceMedia.get_monitor(client, "mon_abc123")
  """
  @spec get_monitor(Client.t(), String.t()) ::
          {:ok, Types.monitor()} | {:error, Error.t()}
  def get_monitor(%Client{} = client, monitor_id) when is_binary(monitor_id) do
    get(client, "/api/v1/monitors/#{path_segment(monitor_id)}")
  end

  @doc """
  Updates a monitor's configuration.

  ## Examples

      {:ok, updated} =
        SfVoiceMedia.update_monitor(client, "mon_abc123", %{threshold: 0.9, enabled: false})
  """
  @spec update_monitor(Client.t(), String.t(), Types.update_monitor_request()) ::
          {:ok, Types.monitor()} | {:error, Error.t()}
  def update_monitor(%Client{} = client, monitor_id, request)
      when is_binary(monitor_id) and is_map(request) do
    patch(client, "/api/v1/monitors/#{path_segment(monitor_id)}", request)
  end

  @doc """
  Deletes a monitor by ID.

  Returns `:ok` on success.

  ## Examples

      :ok = SfVoiceMedia.delete_monitor(client, "mon_abc123")
  """
  @spec delete_monitor(Client.t(), String.t()) :: :ok | {:error, Error.t()}
  def delete_monitor(%Client{} = client, monitor_id) when is_binary(monitor_id) do
    case request(client, :delete, "/api/v1/monitors/#{path_segment(monitor_id)}") do
      {:ok, _} -> :ok
      {:error, _} = err -> err
    end
  end

  @doc """
  Lists events for a monitor, optionally filtering to matched-only.

  ## Options (as map keys)

    - `:matched_only` — when `true`, only return events where the monitor matched
    - `:limit` — max number of events to return
    - `:offset` — pagination offset

  ## Examples

      {:ok, %{items: events, total: n}} =
        SfVoiceMedia.list_monitor_events(client, "mon_abc123")

      {:ok, %{items: events}} =
        SfVoiceMedia.list_monitor_events(client, "mon_abc123", %{matched_only: true, limit: 50})
  """
  @spec list_monitor_events(Client.t(), String.t(), map()) ::
          {:ok, Types.monitor_event_list_response()} | {:error, Error.t()}
  def list_monitor_events(%Client{} = client, monitor_id, params \\ %{})
      when is_binary(monitor_id) and is_map(params) do
    qs = build_query(params)
    get(client, "/api/v1/monitors/#{path_segment(monitor_id)}/events#{qs}")
  end

  @doc """
  Creates a monitor and spawns a background process that polls for matched events,
  invoking `callback` for each new match.

  Returns `{:ok, %{pid: pid, monitor_id: id, ref: ref}}` on success. The spawned
  process monitors its owner and deletes the remote monitor if the owner dies.

  Use `stop_alert/2` to tear down both the polling process and the monitor.

  ## Options

    - `:interval_ms` — polling interval in milliseconds (default: 5_000)
    - any other key is forwarded to `create_monitor/2` (e.g. `:threshold`, `:asset_class`)

  ## Examples

      {:ok, handle} =
        SfVoiceMedia.alert(client, "product launch", fn event ->
          IO.inspect(event, label: "matched")
        end)

      # later…
      SfVoiceMedia.stop_alert(handle, client)
  """
  @spec alert(Client.t(), String.t(), (Types.monitor_event() -> any()), keyword()) ::
          {:ok, %{pid: pid(), monitor_id: String.t()}} | {:error, Error.t()}
  def alert(%Client{} = client, text, callback, opts \\ []) when is_function(callback, 1) do
    {interval_ms, monitor_opts} = Keyword.pop(opts, :interval_ms, 5_000)
    monitor_req = Map.merge(%{text: text}, Map.new(monitor_opts))

    case create_monitor(client, monitor_req) do
      {:ok, monitor} ->
        caller = self()
        ref = make_ref()

        pid =
          spawn(fn ->
            owner_monitor = Process.monitor(caller)

            alert_loop(
              client,
              monitor[:id],
              callback,
              interval_ms,
              nil,
              0,
              caller,
              ref,
              owner_monitor
            )
          end)

        {:ok, %{pid: pid, monitor_id: monitor[:id], ref: ref}}

      {:error, _} = err ->
        err
    end
  end

  @doc """
  Stops an alert by killing its polling process and deleting the monitor.

  ## Examples

      SfVoiceMedia.stop_alert(handle, client)
  """
  @spec stop_alert(
          %{
            required(:pid) => pid(),
            required(:monitor_id) => String.t(),
            optional(:ref) => reference()
          },
          Client.t()
        ) ::
          :ok | {:error, Error.t()}
  def stop_alert(%{pid: pid, monitor_id: monitor_id}, %Client{} = client) do
    monitor_ref = Process.monitor(pid)
    send(pid, {:stop, self()})

    receive do
      {:sf_voice_alert_stopped, ^pid} -> :ok
      {:DOWN, ^monitor_ref, :process, ^pid, _reason} -> :ok
    after
      5_000 ->
        Process.exit(pid, :kill)

        receive do
          {:DOWN, ^monitor_ref, :process, ^pid, _reason} -> :ok
        end
    end

    Process.demonitor(monitor_ref, [:flush])
    delete_monitor(client, monitor_id)
  end

  @doc """
  Polls an ingestion task until its status becomes "ready" or "failed".

  polls `get_task/2` at a fixed interval. returns the final task map when
  the task reaches "ready". raises `SfVoiceMedia.Error` if the task fails
  or the timeout is exceeded.

  ## options

    - `:interval_ms` — milliseconds to wait between polls (default: 1_500)
    - `:timeout_ms`  — maximum total wait in milliseconds (default: 120_000)

  ## examples

      task = SfVoiceMedia.poll_task!(client, tid)
      task = SfVoiceMedia.poll_task!(client, tid, interval_ms: 2_000, timeout_ms: 60_000)
  """
  def poll_task!(%Client{} = client, task_id, opts \\ []) do
    interval_ms = Keyword.get(opts, :interval_ms, 1_500)
    timeout_ms = Keyword.get(opts, :timeout_ms, 120_000)
    deadline = System.monotonic_time(:millisecond) + timeout_ms

    do_poll!(client, task_id, interval_ms, deadline, timeout_ms)
  end

  # ── polling loop (private) ────────────────────────────────────────────────────

  defp do_poll!(client, task_id, interval_ms, deadline, timeout_ms) do
    case get_task(client, task_id) do
      {:ok, %{status: status} = task} when status in ["ready", "failed"] ->
        if status == "failed" do
          raise Error,
            code: "task_failed",
            message: "task #{task_id} failed: #{task[:error] || "unknown reason"}",
            status: nil
        else
          task
        end

      {:ok, _} ->
        # still in progress — check deadline before sleeping
        now = System.monotonic_time(:millisecond)

        if now + interval_ms > deadline do
          raise Error.poll_timeout(task_id, timeout_ms)
        end

        Process.sleep(interval_ms)
        do_poll!(client, task_id, interval_ms, deadline, timeout_ms)

      {:error, %Error{} = err} ->
        raise err
    end
  end

  # ── alert polling loop (private) ──────────────────────────────────────────────

  # polls matched events with a bounded checkpoint pager.
  defp alert_loop(
         client,
         monitor_id,
         callback,
         interval_ms,
         checkpoint,
         failures,
         caller,
         ref,
         owner_monitor
       ) do
    receive do
      {:stop, from} -> send(from, {:sf_voice_alert_stopped, self()})
      {:DOWN, ^owner_monitor, :process, ^caller, _reason} -> delete_monitor(client, monitor_id)
    after
      interval_ms ->
        with {:ok, events, next_checkpoint} <-
               collect_new_monitor_events(client, monitor_id, checkpoint),
             :ok <- deliver_alert_events(events, callback) do
          alert_loop(
            client,
            monitor_id,
            callback,
            interval_ms,
            next_checkpoint || checkpoint,
            0,
            caller,
            ref,
            owner_monitor
          )
        else
          {:error, %Error{} = error} ->
            next_failures = failures + 1

            retryable =
              error.code not in ["callback_error", "pagination_overflow"] and
                (is_nil(error.status) or error.status == 429 or error.status >= 500)

            if retryable and next_failures < 3 do
              alert_loop(
                client,
                monitor_id,
                callback,
                interval_ms,
                checkpoint,
                next_failures,
                caller,
                ref,
                owner_monitor
              )
            else
              send(caller, {:sf_voice_alert_error, ref, error})
              delete_monitor(client, monitor_id)
            end
        end
    end
  end

  defp collect_new_monitor_events(client, monitor_id, checkpoint) do
    collect_new_monitor_events(client, monitor_id, checkpoint, 0, nil, [], MapSet.new(), 0)
  end

  defp collect_new_monitor_events(
         _client,
         _monitor_id,
         _checkpoint,
         _offset,
         _next_checkpoint,
         _pending,
         _seen,
         1_000
       ) do
    {:error,
     %Error{
       code: "pagination_overflow",
       message: "alert event pagination exceeded 1000 pages",
       status: nil
     }}
  end

  defp collect_new_monitor_events(
         client,
         monitor_id,
         checkpoint,
         offset,
         next_checkpoint,
         pending,
         seen,
         page
       ) do
    params = %{matched_only: true, limit: 100, offset: offset}

    case list_monitor_events(client, monitor_id, params) do
      {:ok, %{items: items, total: total}} ->
        newest = next_checkpoint || get_in(items, [Access.at(0), Access.key(:id)])

        {new_pending, new_seen, reached_checkpoint} =
          Enum.reduce_while(items, {pending, seen, false}, fn event, {events, ids, _} ->
            cond do
              event[:id] == checkpoint ->
                {:halt, {events, ids, true}}

              MapSet.member?(ids, event[:id]) ->
                {:cont, {events, ids, false}}

              true ->
                {:cont, {[event | events], MapSet.put(ids, event[:id]), false}}
            end
          end)

        new_offset = offset + length(items)

        if reached_checkpoint or length(items) < 100 or new_offset >= total do
          {:ok, new_pending, newest}
        else
          collect_new_monitor_events(
            client,
            monitor_id,
            checkpoint,
            new_offset,
            newest,
            new_pending,
            new_seen,
            page + 1
          )
        end

      {:error, _} = error ->
        error
    end
  end

  defp deliver_alert_events(events, callback) do
    Enum.each(events, callback)
    :ok
  rescue
    error ->
      {:error, %Error{code: "callback_error", message: Exception.message(error), status: nil}}
  end

  # ── http helpers ─────────────────────────────────────────────────────────────

  defp get(client, path) do
    case request(client, :get, path) do
      {:ok, body} -> {:ok, body}
      {:error, _} = err -> err
    end
  end

  defp post(client, path, body) do
    case request(client, :post, path, body) do
      {:ok, resp} -> {:ok, resp}
      {:error, _} = err -> err
    end
  end

  defp patch(client, path, body) do
    case request(client, :patch, path, body) do
      {:ok, resp} -> {:ok, resp}
      {:error, _} = err -> err
    end
  end

  # low-level dispatcher — builds Req options and handles errors uniformly
  defp request(%Client{} = client, method, path, body \\ nil) do
    url = client.base_url <> path

    base_opts = [
      url: url,
      headers: [{"authorization", "Bearer " <> client.api_key}],
      decode_json: [keys: :atoms]
    ]

    body_opt = if body, do: [json: stringify_keys(body)], else: []

    req_opts = Keyword.merge(base_opts ++ body_opt, client.http_opts)

    result =
      case method do
        :get -> Req.get(req_opts)
        :post -> Req.post(req_opts)
        :patch -> Req.patch(req_opts)
        :delete -> Req.delete(req_opts)
      end

    case result do
      {:ok, %Req.Response{status: status, body: resp_body}} when status in 200..299 ->
        # 204 has no body; normalise to empty map so callers get a consistent type
        {:ok, if(resp_body == "", do: %{}, else: resp_body)}

      {:ok, %Req.Response{status: status, body: resp_body}} ->
        {:error, Error.from_response(status, resp_body)}

      {:error, exception} ->
        # transport-level error (timeout, DNS failure, etc.)
        {:error,
         %Error{
           code: "transport_error",
           message: Exception.message(exception),
           status: nil
         }}
    end
  end

  # converts atom-keyed maps to string-keyed before JSON encoding,
  # so callers can pass %{source: :url} without worrying about encoding
  defp stringify_keys(map) when is_map(map) do
    Map.new(map, fn
      {k, v} when is_atom(k) -> {Atom.to_string(k), stringify_keys(v)}
      {k, v} -> {k, stringify_keys(v)}
    end)
  end

  defp stringify_keys(v) when is_atom(v), do: Atom.to_string(v)
  defp stringify_keys(list) when is_list(list), do: Enum.map(list, &stringify_keys/1)
  defp stringify_keys(v), do: v

  defp path_segment(value), do: URI.encode(value, &URI.char_unreserved?/1)

  # builds a query string from a map, omitting nil values
  defp build_query(params) when map_size(params) == 0, do: ""

  defp build_query(params) do
    qs =
      params
      |> Enum.reject(fn {_, v} -> is_nil(v) end)
      |> Enum.map(fn {k, v} -> "#{k}=#{URI.encode_www_form(to_string(v))}" end)
      |> Enum.join("&")

    if qs == "", do: "", else: "?#{qs}"
  end
end
