#pragma once

#include <cstdint>
#include <iomanip>
#include <optional>
#include <sstream>
#include <string>
#include <vector>

#include <nlohmann/json.hpp>

#include "error.hpp"

namespace sf_voice {

inline std::string compatibility_asset_id(const std::string& value) {
    std::uint64_t hash = 0xcbf29ce484222325ULL;
    for (const unsigned char value_byte : value) {
        hash ^= value_byte;
        hash *= 0x100000001b3ULL;
    }
    std::ostringstream stream;
    stream << "legacy-" << std::hex << std::setw(16) << std::setfill('0') << hash;
    return stream.str();
}

// ── result type ────────────────────────────────────────────────────────────
// ok/err container — no std::expected dependency, works on C++17.
// always check result.ok before accessing result.value.
template <typename T>
struct Result {
    bool ok = false;
    T value{};
    SfVoiceMediaError error{};

    static Result success(T v) {
        Result r;
        r.ok    = true;
        r.value = std::move(v);
        return r;
    }
    static Result failure(SfVoiceMediaError e) {
        Result r;
        r.ok    = false;
        r.error = std::move(e);
        return r;
    }
};

// ── enums ──────────────────────────────────────────────────────────────────

enum class TaskStatus {
    Pending,
    Indexing,
    Ready,
    Failed,
    Unknown  // forward-compat fallback for new api values
};

inline TaskStatus task_status_from_string(const std::string& s) {
    if (s == "pending")  return TaskStatus::Pending;
    if (s == "indexing") return TaskStatus::Indexing;
    if (s == "ready")    return TaskStatus::Ready;
    if (s == "failed")   return TaskStatus::Failed;
    return TaskStatus::Unknown;
}

// search modalities — matches the API's match_type field
enum class SearchType {
    Visual,
    Conversation,
    TextInVideo,
    Video,
    Audio,
    Transcript
};

inline std::string search_type_to_string(SearchType t) {
    switch (t) {
        case SearchType::Visual:       return "visual";
        case SearchType::Conversation: return "conversation";
        case SearchType::TextInVideo:  return "text_in_video";
        case SearchType::Video:        return "video";
        case SearchType::Audio:        return "audio";
        case SearchType::Transcript:   return "transcript";
        default:                       return "visual";
    }
}

// ── pagination ─────────────────────────────────────────────────────────────

struct PageInfo {
    int  page      = 1;
    int  limit     = 20;
    int  total     = 0;
    bool has_more  = false;
};

inline void from_json(const nlohmann::json& j, PageInfo& p) {
    j.at("page").get_to(p.page);
    j.at("limit").get_to(p.limit);
    if (j.contains("total"))    j.at("total").get_to(p.total);
    if (j.contains("has_more")) j.at("has_more").get_to(p.has_more);
}

// ── asset ──────────────────────────────────────────────────────────────────

struct Asset {
    std::string asset_id;
    std::string id;  // deprecated: use asset_id
    std::optional<std::string> filename;
    std::optional<std::string> project_id;
    std::optional<std::string> asset_class;
    std::vector<SearchType> types;
    std::string status;
    std::optional<std::string> title;
    std::optional<std::string> url;
    std::optional<std::string> created_at;
    std::optional<std::string> updated_at;
    nlohmann::json metadata;  // full response preserved for unknown fields
};

inline void from_json(const nlohmann::json& j, Asset& a) {
    j.at("asset_id").get_to(a.asset_id);
    a.id = a.asset_id;
    j.at("status").get_to(a.status);
    if (j.contains("filename")    && !j["filename"].is_null())    a.filename    = j["filename"].get<std::string>();
    if (j.contains("project_id")  && !j["project_id"].is_null())  a.project_id  = j["project_id"].get<std::string>();
    if (j.contains("asset_class") && !j["asset_class"].is_null()) a.asset_class = j["asset_class"].get<std::string>();
    if (j.contains("title")      && !j["title"].is_null())      a.title      = j["title"].get<std::string>();
    if (j.contains("url")        && !j["url"].is_null())        a.url        = j["url"].get<std::string>();
    if (j.contains("created_at") && !j["created_at"].is_null()) a.created_at = j["created_at"].get<std::string>();
    if (j.contains("updated_at") && !j["updated_at"].is_null()) a.updated_at = j["updated_at"].get<std::string>();
    a.metadata = j;
}

// ── task ───────────────────────────────────────────────────────────────────

struct Task {
    std::string task_id;
    std::string asset_id;
    TaskStatus  status     = TaskStatus::Unknown;
    std::string status_raw;
    std::optional<std::string> error;
    std::optional<std::string> created_at;
    std::optional<std::string> completed_at;
};

inline void from_json(const nlohmann::json& j, Task& t) {
    j.at("task_id").get_to(t.task_id);
    j.at("asset_id").get_to(t.asset_id);
    j.at("status").get_to(t.status_raw);
    t.status = task_status_from_string(t.status_raw);
    if (j.contains("error")        && !j["error"].is_null())        t.error        = j["error"].get<std::string>();
    if (j.contains("created_at")   && !j["created_at"].is_null())   t.created_at   = j["created_at"].get<std::string>();
    if (j.contains("completed_at") && !j["completed_at"].is_null()) t.completed_at = j["completed_at"].get<std::string>();
}

// ── ingest ─────────────────────────────────────────────────────────────────

struct IngestRequest {
    std::string source;                        // "url" or "s3"
    std::string asset_id;                      // your ID — returned on every search result
    std::optional<std::string> asset_class;    // logical group for scoped search
    std::optional<std::string> url;            // required when source == "url"
    std::optional<std::string> s3_key;         // required when source == "s3"
    std::optional<std::string> project_id;
    std::optional<std::string> media_kind;     // "audio" or "video"
    std::optional<std::string> media_type;     // deprecated: use media_kind
    std::optional<std::string> prefix;
    std::optional<bool> transcript_only;
    nlohmann::json metadata;                   // flat key/value pairs for your own use
    nlohmann::json extra;                      // additional fields forwarded as-is
};

// builds the POST body — not a from_json since this goes outbound
inline nlohmann::json to_json(const IngestRequest& req) {
    nlohmann::json j = req.extra.is_object() ? req.extra : nlohmann::json::object();
    j["source"]   = req.source;
    if (!req.asset_id.empty()) j["asset_id"] = req.asset_id;
    else if (req.url)           j["asset_id"] = compatibility_asset_id(*req.url);
    else if (req.s3_key)        j["asset_id"] = compatibility_asset_id(*req.s3_key);
    if (req.asset_class) j["asset_class"] = *req.asset_class;
    if (req.url)         j["url"]         = *req.url;
    if (req.s3_key)      j["s3_key"]      = *req.s3_key;
    if (req.project_id)  j["project_id"]  = *req.project_id;
    if (req.media_kind)       j["media_kind"] = *req.media_kind;
    else if (req.media_type)  j["media_kind"] = *req.media_type;
    if (req.prefix)      j["prefix"]      = *req.prefix;
    if (req.transcript_only) j["transcript_only"] = *req.transcript_only;
    if (req.metadata.is_object() && !req.metadata.empty()) j["metadata"] = req.metadata;
    return j;
}

struct IngestResponse {
    std::string asset_id;
    std::string task_id;
    std::string status;
};

inline void from_json(const nlohmann::json& j, IngestResponse& r) {
    j.at("asset_id").get_to(r.asset_id);
    j.at("task_id").get_to(r.task_id);
    j.at("status").get_to(r.status);
}

// ── list assets ────────────────────────────────────────────────────────────

struct ListAssetsRequest {
    int page  = 1;
    int limit = 20;
};

struct AssetListResponse {
    std::vector<Asset> items;
    PageInfo page_info;
};

inline void from_json(const nlohmann::json& j, AssetListResponse& r) {
    j.at("items").get_to(r.items);
    if (j.contains("page_info")) j.at("page_info").get_to(r.page_info);
    else r.page_info = j.get<PageInfo>();
}

// ── search ─────────────────────────────────────────────────────────────────

struct SearchRequest {
    std::string query;
    std::vector<SearchType> types;             // modalities to search; empty = all
    std::optional<std::string> asset_class;    // scope to one logical group
    std::optional<std::string> project_id;
    std::optional<std::string> prefix;
    std::vector<std::string> asset_ids;        // scope to specific assets
    std::optional<double> threshold;           // 0.0–1.0; higher = fewer, more confident
    int page  = 1;
    int limit = 20;
};

inline nlohmann::json to_json(const SearchRequest& req) {
    nlohmann::json j;
    j["query"] = req.query;
    if (!req.types.empty()) {
        auto arr = nlohmann::json::array();
        for (const auto& t : req.types) arr.push_back(search_type_to_string(t));
        j["types"] = arr;
    }
    if (req.asset_class)         j["asset_class"] = *req.asset_class;
    if (req.project_id)          j["project_id"]  = *req.project_id;
    if (req.prefix)              j["prefix"]      = *req.prefix;
    if (!req.asset_ids.empty())  j["asset_ids"]   = req.asset_ids;
    if (req.threshold)           j["threshold"]   = *req.threshold;
    j["page"]  = req.page;
    j["limit"] = req.limit;
    return j;
}

struct SearchResult {
    std::string document_id;
    std::string asset_id;
    std::optional<std::string> asset_class;
    std::optional<std::string> twelvelabs_media_id;
    std::optional<double> score;
    std::optional<double> start_ms;
    std::optional<double> end_ms;
    std::optional<std::string> match_type;
    std::optional<std::string> text;
    std::optional<std::string> transcript_snippet;
    nlohmann::json metadata;  // full result preserved for unknown fields
};

inline void from_json(const nlohmann::json& j, SearchResult& r) {
    j.at("document_id").get_to(r.document_id);
    if (j.contains("asset_id")           && !j["asset_id"].is_null())           r.asset_id           = j["asset_id"].get<std::string>();
    if (j.contains("asset_class")        && !j["asset_class"].is_null())        r.asset_class        = j["asset_class"].get<std::string>();
    if (j.contains("twelvelabs_media_id") && !j["twelvelabs_media_id"].is_null()) r.twelvelabs_media_id = j["twelvelabs_media_id"].get<std::string>();
    if (j.contains("score")              && !j["score"].is_null())              r.score              = j["score"].get<double>();
    if (j.contains("start_ms")           && !j["start_ms"].is_null())           r.start_ms           = j["start_ms"].get<double>();
    if (j.contains("end_ms")             && !j["end_ms"].is_null())             r.end_ms             = j["end_ms"].get<double>();
    if (j.contains("match_type")         && !j["match_type"].is_null())         r.match_type         = j["match_type"].get<std::string>();
    if (j.contains("text")               && !j["text"].is_null()) {
        r.text = j["text"].get<std::string>();
        r.transcript_snippet = r.text;
    } else if (j.contains("transcript_snippet") && !j["transcript_snippet"].is_null()) {
        r.transcript_snippet = j["transcript_snippet"].get<std::string>();
    }
    r.metadata = j;
}

struct SearchResponse {
    std::string search_query_id;
    std::vector<SearchResult> results;
    PageInfo page_info;
};

inline void from_json(const nlohmann::json& j, SearchResponse& r) {
    j.at("search_query_id").get_to(r.search_query_id);
    j.at("results").get_to(r.results);
    if (j.contains("page_info")) j.at("page_info").get_to(r.page_info);
    else r.page_info = j.get<PageInfo>();
}

struct Prefix {
    std::string prefix;
    bool owned_by_caller = false;
};

inline void from_json(const nlohmann::json& j, Prefix& p) {
    j.at("prefix").get_to(p.prefix);
    j.at("owned_by_caller").get_to(p.owned_by_caller);
}

struct PrefixListResponse {
    std::vector<Prefix> items;
};

inline void from_json(const nlohmann::json& j, PrefixListResponse& r) {
    j.at("items").get_to(r.items);
}

// ── monitors ──────────────────────────────────────────────────────────

struct CreateMonitorRequest {
    std::string text;
    std::optional<std::string> slug;
    std::optional<std::string> project_id;
    std::optional<std::string> asset_class;
    std::optional<float> threshold;
};

inline nlohmann::json to_json(const CreateMonitorRequest& req) {
    nlohmann::json j;
    j["text"] = req.text;
    if (req.slug)        j["slug"]        = *req.slug;
    if (req.project_id)  j["project_id"]  = *req.project_id;
    if (req.asset_class) j["asset_class"] = *req.asset_class;
    if (req.threshold)   j["threshold"]   = *req.threshold;
    return j;
}

struct UpdateMonitorRequest {
    std::optional<std::string> text;
    std::optional<float> threshold;
    std::optional<bool> enabled;
    std::optional<std::string> asset_class;
};

inline nlohmann::json to_json(const UpdateMonitorRequest& req) {
    nlohmann::json j;
    if (req.text)        j["text"]        = *req.text;
    if (req.threshold)   j["threshold"]   = *req.threshold;
    if (req.enabled)     j["enabled"]     = *req.enabled;
    if (req.asset_class) j["asset_class"] = *req.asset_class;
    return j;
}

struct Monitor {
    std::string id, slug, text, created_at, updated_at;
    std::optional<std::string> project_id, asset_class;
    float threshold = 0.7f;
    bool enabled = true;
};

inline void from_json(const nlohmann::json& j, Monitor& m) {
    j.at("id").get_to(m.id);
    j.at("slug").get_to(m.slug);
    j.at("text").get_to(m.text);
    j.at("created_at").get_to(m.created_at);
    j.at("updated_at").get_to(m.updated_at);
    if (j.contains("project_id")  && !j["project_id"].is_null())  m.project_id  = j["project_id"].get<std::string>();
    if (j.contains("asset_class") && !j["asset_class"].is_null()) m.asset_class = j["asset_class"].get<std::string>();
    if (j.contains("threshold"))  j.at("threshold").get_to(m.threshold);
    if (j.contains("enabled"))    j.at("enabled").get_to(m.enabled);
}

struct MonitorListResponse {
    std::vector<Monitor> items;
    int64_t total = 0;
};

inline void from_json(const nlohmann::json& j, MonitorListResponse& r) {
    j.at("items").get_to(r.items);
    if (j.contains("total")) j.at("total").get_to(r.total);
}

struct MonitorEvent {
    std::string id, monitor_id, document_id, created_at;
    std::optional<std::string> asset_id;
    bool matched = false;
    std::optional<float> score;
    bool webhook_sent = false;
    nlohmann::json match_detail;
};

inline void from_json(const nlohmann::json& j, MonitorEvent& e) {
    j.at("id").get_to(e.id);
    j.at("monitor_id").get_to(e.monitor_id);
    j.at("document_id").get_to(e.document_id);
    j.at("created_at").get_to(e.created_at);
    if (j.contains("asset_id")      && !j["asset_id"].is_null())      e.asset_id      = j["asset_id"].get<std::string>();
    if (j.contains("matched"))       j.at("matched").get_to(e.matched);
    if (j.contains("score")         && !j["score"].is_null())         e.score         = j["score"].get<float>();
    if (j.contains("webhook_sent"))  j.at("webhook_sent").get_to(e.webhook_sent);
    if (j.contains("match_detail")  && !j["match_detail"].is_null()) e.match_detail  = j["match_detail"];
}

struct MonitorEventListResponse {
    std::vector<MonitorEvent> items;
    int64_t total = 0;
};

inline void from_json(const nlohmann::json& j, MonitorEventListResponse& r) {
    j.at("items").get_to(r.items);
    if (j.contains("total")) j.at("total").get_to(r.total);
}

// placeholder for 204 No Content responses
struct Empty {};

} // namespace sf_voice
