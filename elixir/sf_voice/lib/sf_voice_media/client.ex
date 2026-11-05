defmodule SfVoiceMedia.Client do
  @moduledoc """
  holds connection config for a single API tenant.

  build one with `SfVoiceMedia.new/2` — don't construct this struct directly.

  fields:
  - `api_key`   — sent as an authorization bearer token
  - `base_url`  — scheme + host, no trailing slash (e.g. "https://api.sf-voice.com")
  - `http_opts` — keyword list forwarded verbatim to every `Req` call
  """

  @enforce_keys [:api_key, :base_url]
  defstruct [:api_key, :base_url, http_opts: []]

  @type t :: %__MODULE__{
          api_key: String.t(),
          base_url: String.t(),
          http_opts: keyword()
        }
end
