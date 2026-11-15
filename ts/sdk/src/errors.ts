import type { ApiErrorCode } from "./types.js";

/**
 * thrown by every RaidzeroMedia method on a non-2xx response.
 * `code` is the machine-readable error code from the API.
 * `status` is the HTTP status code.
 */
export class RaidzeroMediaError extends Error {
  readonly code: ApiErrorCode;
  readonly status: number;

  constructor(code: ApiErrorCode, message: string, status: number) {
    super(message);
    this.name = "RaidzeroMediaError";
    this.code = code;
    this.status = status;
    // restore prototype chain when targeting ES5
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/**
 * thrown by `request` when a fetch stalls past the client-level timeout.
 */
export class RaidzeroMediaRequestTimeoutError extends Error {
  constructor(timeoutMs: number) {
    super(`request timed out after ${timeoutMs}ms`);
    this.name = "RaidzeroMediaRequestTimeoutError";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/**
 * thrown by `pollTask` when the timeout expires before the task
 * reaches a terminal state.
 */
export class RaidzeroMediaPollTimeoutError extends Error {
  readonly taskId: string;

  constructor(taskId: string, timeoutMs: number) {
    super(
      `task ${taskId} did not complete within ${timeoutMs}ms`
    );
    this.name = "RaidzeroMediaPollTimeoutError";
    this.taskId = taskId;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}
