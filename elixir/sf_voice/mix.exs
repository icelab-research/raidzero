defmodule SfVoiceMedia.MixProject do
  use Mix.Project

  @version "0.1.1"
  @source_url "https://github.com/icelab-research/raidzero"

  @doc """
  Defines the Mix project configuration for the `:sf_voice` application.

  Returns a keyword list used by Mix containing project metadata and build configuration such as the application name, version, Elixir requirement, start mode, dependencies, package metadata, package description, and documentation settings.
  """
  @spec project() :: keyword()
  def project do
    [
      app: :sf_voice,
      version: @version,
      elixir: "~> 1.14",
      start_permanent: Mix.env() == :prod,
      deps: deps(),
      package: package(),
      description:
        "Elixir SDK for the sf-voice media API — ingest, query, and search audio/video.",
      docs: docs()
    ]
  end

  @doc """
  Provides the OTP application configuration for the project.

  Specifies that the `:logger` application is included in `:extra_applications` so it is started at runtime.
  """
  @spec application() :: keyword()
  def application do
    [
      extra_applications: [:logger]
    ]
  end

  defp deps do
    [
      {:req, "~> 0.5"},
      # dev / docs only
      {:ex_doc, ">= 0.0.0", only: :dev, runtime: false}
    ]
  end

  defp package do
    [
      name: "sf_voice",
      licenses: ["MIT"],
      links: %{
        "GitHub" => @source_url
      },
      maintainers: ["sf-voice"]
    ]
  end

  defp docs do
    [
      main: "SfVoiceMedia",
      source_url: @source_url,
      extras: ["README.md"]
    ]
  end
end
