#pragma once

#include <stdexcept>
#include <string>

namespace raidzero {

// api error payload — carries the structured error returned by the server,
// or a synthetic error for network/parse failures.
struct RaidzeroMediaError {
    std::string code;      // machine-readable code (e.g. "not_found", "unauthorized")
    std::string message;   // human-readable description
    int status_code = 0;   // HTTP status; 0 for transport/parse errors

    RaidzeroMediaError() = default;

    RaidzeroMediaError(std::string code, std::string message, int status_code = 0)
        : code(std::move(code))
        , message(std::move(message))
        , status_code(status_code) {}

    // factory helpers — keep call sites readable
    static RaidzeroMediaError network(const std::string& msg) {
        return {"network_error", msg, 0};
    }
    static RaidzeroMediaError http(int status, const std::string& body) {
        return {"http_error", body, status};
    }
    static RaidzeroMediaError parse(const std::string& detail) {
        return {"parse_error", detail, 0};
    }
    static RaidzeroMediaError timeout(const std::string& task_id) {
        return {
            "poll_timeout",
            "task " + task_id + " did not reach a terminal state before the timeout",
            0
        };
    }
};

// throwable variant — wraps RaidzeroMediaError for exception-based usage
class RaidzeroMediaException : public std::runtime_error {
public:
    RaidzeroMediaError error;

    explicit RaidzeroMediaException(RaidzeroMediaError err)
        : std::runtime_error(err.message)
        , error(std::move(err)) {}

    RaidzeroMediaException(std::string code, std::string message, int status_code = 0)
        : RaidzeroMediaException(RaidzeroMediaError{std::move(code), std::move(message), status_code}) {}
};

} // namespace raidzero
