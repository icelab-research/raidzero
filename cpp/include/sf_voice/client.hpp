#pragma once

#include <atomic>
#include <algorithm>
#include <chrono>
#include <functional>
#include <future>
#include <memory>
#include <mutex>
#include <optional>
#include <set>
#include <string>
#include <thread>

#include <cpr/cpr.h>
#include <nlohmann/json.hpp>

#include "error.hpp"
#include "types.hpp"

namespace sf_voice {

// ── internal helpers ───────────────────────────────────────────────────────
namespace detail {

// default per-request timeout
static constexpr int kTimeoutMs = 30'000;

inline std::string path_segment(const std::string& value) {
    const auto encoded = cpr::util::urlEncode(value);
    return {encoded.begin(), encoded.end()};
}

// parse the { "error": { "code", "message" } } envelope the API returns on errors
inline SfVoiceMediaError parse_api_error(int status_code, const std::string& body) {
    try {
        auto j = nlohmann::json::parse(body);
        if (j.contains("error") && j["error"].is_object()) {
            auto& e = j["error"];
            return SfVoiceMediaError{
                e.value("code",    "api_error"),
                e.value("message", body),
                status_code
            };
        }
    } catch (...) {}
    return SfVoiceMediaError::http(status_code, body);
}

template <typename T>
Result<T> parse_body(const std::string& body) {
    try {
        return Result<T>::success(nlohmann::json::parse(body).get<T>());
    } catch (const std::exception& ex) {
        return Result<T>::failure(SfVoiceMediaError::parse(ex.what()));
    }
}

template <typename T>
Result<T> handle_response(const cpr::Response& r) {
    if (r.error.code != cpr::ErrorCode::OK) {
        return Result<T>::failure(SfVoiceMediaError::network(r.error.message));
    }
    if (r.status_code < 200 || r.status_code >= 300) {
        return Result<T>::failure(parse_api_error(r.status_code, r.text));
    }
    return parse_body<T>(r.text);
}

// 204 No Content — nothing to parse
template <>
inline Result<Empty> handle_response<Empty>(const cpr::Response& r) {
    if (r.error.code != cpr::ErrorCode::OK) {
        return Result<Empty>::failure(SfVoiceMediaError::network(r.error.message));
    }
    if (r.status_code < 200 || r.status_code >= 300) {
        return Result<Empty>::failure(parse_api_error(r.status_code, r.text));
    }
    return Result<Empty>::success(Empty{});
}

} // namespace detail

// ── alert handle ──────────────────────────────────────────────────────────
// returned by alert() — stops the background poll thread on destruction.

class SfVoiceMedia; // forward declaration

struct AlertHandle {
    std::string monitor_id;

    Result<Empty> stop_checked() {
        std::lock_guard<std::mutex> lock(*stop_mutex_);
        if (cleanup_done_) {
            return cleanup_error_
                ? Result<Empty>::failure(*cleanup_error_)
                : Result<Empty>::success(Empty{});
        }
        stopped_->store(true);
        if (thread_.joinable()) thread_.join();
        auto r = cpr::Delete(
            cpr::Url{base_url_ + "/api/v1/monitors/" + detail::path_segment(monitor_id)},
            auth_,
            cpr::Timeout{5000}
        );
        auto delete_result = detail::handle_response<Empty>(r);
        cleanup_done_ = true;
        if (!delete_result.ok) cleanup_error_ = delete_result.error;
        if (!cleanup_error_ && *poll_error_) cleanup_error_ = *poll_error_;
        return cleanup_error_
            ? Result<Empty>::failure(*cleanup_error_)
            : Result<Empty>::success(Empty{});
    }

    void stop() { (void)stop_checked(); }

    ~AlertHandle() { if (stopped_ && !cleanup_done_) stop(); }

    AlertHandle(const AlertHandle&) = delete;
    AlertHandle& operator=(const AlertHandle&) = delete;
    AlertHandle(AlertHandle&&) = default;
    AlertHandle& operator=(AlertHandle&&) = default;

private:
    friend class SfVoiceMedia;
    friend struct Result<AlertHandle>;
    AlertHandle()
        : stopped_(std::make_shared<std::atomic<bool>>(false))
        , poll_error_(std::make_shared<std::optional<SfVoiceMediaError>>())
        , error_mutex_(std::make_shared<std::mutex>())
        , stop_mutex_(std::make_shared<std::mutex>()) {}

    std::shared_ptr<std::atomic<bool>> stopped_;
    std::shared_ptr<std::optional<SfVoiceMediaError>> poll_error_;
    std::shared_ptr<std::mutex> error_mutex_;
    std::shared_ptr<std::mutex> stop_mutex_;
    std::thread thread_;
    std::string base_url_;
    cpr::Header auth_;
    bool cleanup_done_ = false;
    std::optional<SfVoiceMediaError> cleanup_error_;
};

// ── client ─────────────────────────────────────────────────────────────────

class SfVoiceMedia {
public:
    explicit SfVoiceMedia(std::string api_key,
                          std::string base_url = "https://api.sf-voice.com")
        : api_key_(std::move(api_key))
        , base_url_(std::move(base_url))
    {
        // strip trailing slash so every path concat is predictable
        while (!base_url_.empty() && base_url_.back() == '/') {
            base_url_.pop_back();
        }
    }

    // ── ingest ───────────────────────────────────────────────────────────

    std::future<Result<IngestResponse>> ingest(IngestRequest request) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url = std::move(base_url),
             auth     = std::move(auth),
             request  = std::move(request)]()
        {
            auto r = cpr::Post(
                cpr::Url{base_url + "/api/v1/ingest"},
                auth,
                cpr::Header{{"Content-Type", "application/json"}},
                cpr::Body{to_json(request).dump()},
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<IngestResponse>(r);
        });
    }

    // ── tasks ────────────────────────────────────────────────────────────

    std::future<Result<Task>> get_task(std::string task_id) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url = std::move(base_url),
             auth     = std::move(auth),
             task_id  = std::move(task_id)]()
        {
            auto r = cpr::Get(
                cpr::Url{base_url + "/api/v1/tasks/" + detail::path_segment(task_id)},
                auth,
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<Task>(r);
        });
    }

    // polls until the task reaches ready or failed, or the timeout elapses
    std::future<Result<Task>> poll_task(
        std::string task_id,
        std::chrono::milliseconds interval = std::chrono::milliseconds{1'500},
        std::chrono::milliseconds timeout  = std::chrono::milliseconds{120'000}
    ) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url = std::move(base_url),
             auth     = std::move(auth),
             task_id  = std::move(task_id),
             interval,
             timeout]() -> Result<Task>
        {
            using clock = std::chrono::steady_clock;
            const auto deadline = clock::now() + timeout;

            while (clock::now() < deadline) {
                auto r = cpr::Get(
                    cpr::Url{base_url + "/api/v1/tasks/" + detail::path_segment(task_id)},
                    auth,
                    cpr::Timeout{detail::kTimeoutMs}
                );
                auto result = detail::handle_response<Task>(r);

                if (!result.ok) return result;

                if (result.value.status == TaskStatus::Ready ||
                    result.value.status == TaskStatus::Failed) {
                    return result;
                }

                if (clock::now() + interval < deadline) {
                    std::this_thread::sleep_for(interval);
                } else {
                    break;
                }
            }

            return Result<Task>::failure(SfVoiceMediaError::timeout(task_id));
        });
    }

    // ── assets ───────────────────────────────────────────────────────────

    std::future<Result<AssetListResponse>> list_assets(ListAssetsRequest request = {}) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url = std::move(base_url), auth = std::move(auth), request]()
        {
            auto r = cpr::Get(
                cpr::Url{base_url + "/api/v1/assets"},
                auth,
                cpr::Parameters{
                    {"page",  std::to_string(request.page)},
                    {"limit", std::to_string(request.limit)}
                },
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<AssetListResponse>(r);
        });
    }

    std::future<Result<Asset>> get_asset(std::string asset_id) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url  = std::move(base_url),
             auth      = std::move(auth),
             asset_id  = std::move(asset_id)]()
        {
            auto r = cpr::Get(
                cpr::Url{base_url + "/api/v1/assets/" + detail::path_segment(asset_id)},
                auth,
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<Asset>(r);
        });
    }

    std::future<Result<Empty>> delete_asset(std::string asset_id) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url  = std::move(base_url),
             auth      = std::move(auth),
             asset_id  = std::move(asset_id)]()
        {
            auto r = cpr::Delete(
                cpr::Url{base_url + "/api/v1/assets/" + detail::path_segment(asset_id)},
                auth,
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<Empty>(r);
        });
    }

    // ── search ───────────────────────────────────────────────────────────

    std::future<Result<SearchResponse>> search(SearchRequest request) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url = std::move(base_url),
             auth     = std::move(auth),
             request  = std::move(request)]()
        {
            auto r = cpr::Post(
                cpr::Url{base_url + "/api/v1/search"},
                auth,
                cpr::Header{{"Content-Type", "application/json"}},
                cpr::Body{to_json(request).dump()},
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<SearchResponse>(r);
        });
    }

    std::future<Result<PrefixListResponse>> list_prefixes() {
        auto base_url = base_url_;
        auto auth = auth_header();
        return std::async(std::launch::async,
            [base_url = std::move(base_url), auth = std::move(auth)]() {
                auto r = cpr::Get(
                    cpr::Url{base_url + "/api/v1/prefixes"},
                    auth,
                    cpr::Timeout{detail::kTimeoutMs}
                );
                return detail::handle_response<PrefixListResponse>(r);
            }
        );
    }

    // ── monitors ────────────────────────────────────────────────────────

    std::future<Result<Monitor>> create_monitor(CreateMonitorRequest request) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url = std::move(base_url),
             auth     = std::move(auth),
             request  = std::move(request)]()
        {
            auto r = cpr::Post(
                cpr::Url{base_url + "/api/v1/monitors"},
                auth,
                cpr::Header{{"Content-Type", "application/json"}},
                cpr::Body{to_json(request).dump()},
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<Monitor>(r);
        });
    }

    std::future<Result<MonitorListResponse>> list_monitors() {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url = std::move(base_url), auth = std::move(auth)]()
        {
            auto r = cpr::Get(
                cpr::Url{base_url + "/api/v1/monitors"},
                auth,
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<MonitorListResponse>(r);
        });
    }

    std::future<Result<Monitor>> get_monitor(std::string monitor_id) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url   = std::move(base_url),
             auth       = std::move(auth),
             monitor_id = std::move(monitor_id)]()
        {
            auto r = cpr::Get(
                cpr::Url{base_url + "/api/v1/monitors/" + monitor_id},
                auth,
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<Monitor>(r);
        });
    }

    std::future<Result<Monitor>> update_monitor(std::string monitor_id,
                                                UpdateMonitorRequest request) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url   = std::move(base_url),
             auth       = std::move(auth),
             monitor_id = std::move(monitor_id),
             request    = std::move(request)]()
        {
            auto r = cpr::Patch(
                cpr::Url{base_url + "/api/v1/monitors/" + monitor_id},
                auth,
                cpr::Header{{"Content-Type", "application/json"}},
                cpr::Body{to_json(request).dump()},
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<Monitor>(r);
        });
    }

    std::future<Result<Empty>> delete_monitor(std::string monitor_id) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url   = std::move(base_url),
             auth       = std::move(auth),
             monitor_id = std::move(monitor_id)]()
        {
            auto r = cpr::Delete(
                cpr::Url{base_url + "/api/v1/monitors/" + monitor_id},
                auth,
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<Empty>(r);
        });
    }

    std::future<Result<MonitorEventListResponse>> list_monitor_events(
        std::string monitor_id,
        bool matched_only = false,
        int limit  = 50,
        int offset = 0
    ) {
        auto base_url = base_url_;
        auto auth     = auth_header();
        return std::async(std::launch::async,
            [base_url     = std::move(base_url),
             auth         = std::move(auth),
             monitor_id   = std::move(monitor_id),
             matched_only,
             limit,
             offset]()
        {
            auto r = cpr::Get(
                cpr::Url{base_url + "/api/v1/monitors/" + detail::path_segment(monitor_id) + "/events"},
                auth,
                cpr::Parameters{
                    {"matched_only", matched_only ? "true" : "false"},
                    {"limit",        std::to_string(limit)},
                    {"offset",       std::to_string(offset)}
                },
                cpr::Timeout{detail::kTimeoutMs}
            );
            return detail::handle_response<MonitorEventListResponse>(r);
        });
    }

    // ── alert (high-level convenience) ──────────────────────────────────
    // creates a monitor, then polls for matched events on a background
    // thread. returns an AlertHandle whose stop() tears everything down.

    Result<AlertHandle> alert(
        std::string text,
        std::function<void(MonitorEvent)> callback,
        CreateMonitorRequest opts = {},
        std::chrono::milliseconds interval = std::chrono::milliseconds{5000}
    ) {
        // merge text into opts
        opts.text = std::move(text);

        // synchronously create the monitor
        auto create_result = create_monitor(std::move(opts)).get();
        if (!create_result.ok) {
            return Result<AlertHandle>::failure(create_result.error);
        }

        AlertHandle handle;
        handle.monitor_id = create_result.value.id;
        handle.base_url_  = base_url_;
        handle.auth_      = auth_header();

        // capture what the thread needs by value — stopped is a shared_ptr
        // so the thread and the (moved) handle share the same flag
        auto poll_base_url   = base_url_;
        auto poll_auth       = auth_header();
        auto poll_monitor_id = handle.monitor_id;
        auto stopped         = handle.stopped_;
        auto poll_error       = handle.poll_error_;
        auto error_mutex      = handle.error_mutex_;

        handle.thread_ = std::thread(
            [poll_base_url = std::move(poll_base_url),
             poll_auth     = std::move(poll_auth),
             poll_monitor_id = std::move(poll_monitor_id),
             callback      = std::move(callback),
             interval,
             stopped       = std::move(stopped),
             poll_error    = std::move(poll_error),
             error_mutex   = std::move(error_mutex)]()
        {
            std::string checkpoint;
            int consecutive_failures = 0;

            while (!stopped->load()) {
                std::vector<MonitorEvent> pending;
                std::set<std::string> seen_this_poll;
                std::string next_checkpoint;
                bool poll_failed = false;
                SfVoiceMediaError failure;
                int offset = 0;

                for (int page = 0; page < 1000 && !stopped->load(); ++page) {
                    auto r = cpr::Get(
                        cpr::Url{poll_base_url + "/api/v1/monitors/" +
                            detail::path_segment(poll_monitor_id) + "/events"},
                        poll_auth,
                        cpr::Parameters{
                            {"matched_only", "true"},
                            {"limit",        "100"},
                            {"offset",       std::to_string(offset)}
                        },
                        cpr::Timeout{detail::kTimeoutMs}
                    );
                    auto parsed = detail::handle_response<MonitorEventListResponse>(r);
                    if (!parsed.ok) {
                        poll_failed = true;
                        failure = parsed.error;
                        break;
                    }
                    if (next_checkpoint.empty() && !parsed.value.items.empty()) {
                        next_checkpoint = parsed.value.items.front().id;
                    }

                    bool reached_checkpoint = false;
                    for (auto& event : parsed.value.items) {
                        if (event.id == checkpoint) {
                            reached_checkpoint = true;
                            break;
                        }
                        if (seen_this_poll.insert(event.id).second) {
                            pending.push_back(std::move(event));
                        }
                    }
                    offset += static_cast<int>(parsed.value.items.size());
                    if (reached_checkpoint || parsed.value.items.size() < 100 ||
                        offset >= parsed.value.total) {
                        break;
                    }
                    if (page == 999) {
                        poll_failed = true;
                        failure = SfVoiceMediaError{
                            "pagination_overflow",
                            "alert event pagination exceeded 1000 pages",
                            0
                        };
                    }
                }

                if (!poll_failed) {
                    try {
                        for (auto event = pending.rbegin(); event != pending.rend(); ++event) {
                            if (stopped->load()) break;
                            callback(*event);
                        }
                        checkpoint = next_checkpoint.empty() ? checkpoint : next_checkpoint;
                        consecutive_failures = 0;
                    } catch (const std::exception& error) {
                        poll_failed = true;
                        failure = SfVoiceMediaError{
                            "callback_error",
                            error.what(),
                            0
                        };
                    } catch (...) {
                        poll_failed = true;
                        failure = SfVoiceMediaError{
                            "callback_error",
                            "alert callback failed",
                            0
                        };
                    }
                }

                if (poll_failed) {
                    consecutive_failures++;
                    const bool retryable = failure.status_code == 0 ||
                        failure.status_code == 429 || failure.status_code >= 500;
                    if (!retryable || consecutive_failures >= 3 ||
                        failure.code == "callback_error" || failure.code == "pagination_overflow") {
                        std::lock_guard<std::mutex> lock(*error_mutex);
                        *poll_error = failure;
                        return;
                    }
                }

                auto remaining = interval;
                const auto step = std::chrono::milliseconds{250};
                while (remaining > std::chrono::milliseconds{0} && !stopped->load()) {
                    auto sleep_for = std::min(remaining, step);
                    std::this_thread::sleep_for(sleep_for);
                    remaining -= sleep_for;
                }
            }
        });

        return Result<AlertHandle>::success(std::move(handle));
    }

private:
    std::string api_key_;
    std::string base_url_;

    cpr::Header auth_header() const {
        return cpr::Header{{"Authorization", "Bearer " + api_key_}};
    }
};

} // namespace sf_voice
