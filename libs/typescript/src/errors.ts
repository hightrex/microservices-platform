import type { ApiErrorResponse, ErrorCode } from "./types";
import { ERROR_CODES } from "./types";

/**
 * Application error with a stable code and HTTP status.
 * Matches the Go `pkg/errors.AppError` response shape.
 */
export class AppError extends Error {
  public readonly code: ErrorCode;
  public readonly httpStatus: number;
  public readonly details?: string;

  constructor(code: ErrorCode, message: string, httpStatus: number, details?: string) {
    super(message);
    this.name = "AppError";
    this.code = code;
    this.httpStatus = httpStatus;
    this.details = details;
    Object.setPrototypeOf(this, AppError.prototype);
  }

  /** Convert to the standardized API error response shape. */
  toResponse(): ApiErrorResponse {
    return {
      success: false,
      error: {
        code: this.code,
        message: this.message,
        ...(this.details ? { details: this.details } : {}),
      },
    };
  }
}

/** Factory helpers matching common HTTP error patterns. */
export function badRequest(message: string, details?: string): AppError {
  return new AppError(ERROR_CODES.BAD_REQUEST, message, 400, details);
}

export function validationFailed(message: string, details?: string): AppError {
  return new AppError(ERROR_CODES.VALIDATION_FAILED, message, 400, details);
}

export function unauthorized(message = "Unauthorized"): AppError {
  return new AppError(ERROR_CODES.UNAUTHORIZED, message, 401);
}

export function forbidden(message = "Forbidden"): AppError {
  return new AppError(ERROR_CODES.FORBIDDEN, message, 403);
}

export function notFound(resource = "Resource"): AppError {
  return new AppError(ERROR_CODES.NOT_FOUND, `${resource} not found`, 404);
}

export function conflict(message: string): AppError {
  return new AppError(ERROR_CODES.CONFLICT, message, 409);
}

export function rateLimited(retryAfterSeconds: number): AppError {
  return new AppError(
    ERROR_CODES.RATE_LIMITED,
    "Too many requests",
    429,
    `Retry after ${retryAfterSeconds} seconds`,
  );
}

export function moduleNotEnabled(moduleName: string): AppError {
  return new AppError(
    ERROR_CODES.MODULE_NOT_ENABLED,
    `Module '${moduleName}' is not enabled for this organization`,
    403,
  );
}

export function serviceUnavailable(service: string): AppError {
  return new AppError(
    ERROR_CODES.SERVICE_UNAVAILABLE,
    `Service '${service}' is temporarily unavailable`,
    503,
  );
}

export function internalError(message = "Internal server error"): AppError {
  return new AppError(ERROR_CODES.INTERNAL_ERROR, message, 500);
}

/**
 * Convert any thrown value into an `ApiErrorResponse`.
 * If it's an `AppError`, use its fields; otherwise treat as 500.
 */
export function toErrorResponse(err: unknown): { status: number; body: ApiErrorResponse } {
  if (err instanceof AppError) {
    return { status: err.httpStatus, body: err.toResponse() };
  }

  const message = err instanceof Error ? err.message : "Unknown error";
  return {
    status: 500,
    body: {
      success: false,
      error: {
        code: ERROR_CODES.INTERNAL_ERROR,
        message: process.env["NODE_ENV"] === "production" ? "Internal server error" : message,
      },
    },
  };
}
