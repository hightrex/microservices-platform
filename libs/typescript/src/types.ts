/**
 * Standardized API response types — matches Go `pkg/errors` response shape.
 */

/** Successful API response wrapper. */
export interface ApiSuccessResponse<T = unknown> {
  success: true;
  data: T;
  message?: string;
}

/** Error detail shape returned by all services. */
export interface ApiErrorDetail {
  code: string;
  message: string;
  details?: string;
}

/** Error API response wrapper. */
export interface ApiErrorResponse {
  success: false;
  error: ApiErrorDetail;
}

/** Union of all possible API responses. */
export type ApiResponse<T = unknown> = ApiSuccessResponse<T> | ApiErrorResponse;

/** JWT claims carried in the access token (matches auth-service token_service.go). */
export interface JwtClaims {
  /** User ID (subject). */
  sub: string;
  /** Tenant / organization ID. */
  tid: string;
  /** Organization ID alias (same as tid in most cases). */
  org: string;
  /** Roles assigned to the user within the tenant. */
  roles: string[];
  /** Issuer. */
  iss: string;
  /** Issued-at (epoch seconds). */
  iat: number;
  /** Expiration (epoch seconds). */
  exp: number;
  /** Token ID for blacklisting. */
  jti: string;
}

/** Pagination request parameters. */
export interface PaginationParams {
  page: number;
  pageSize: number;
}

/** Paginated response wrapper. */
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

/** Dependency health status for aggregate health checks. */
export interface DependencyHealth {
  name: string;
  status: "healthy" | "degraded" | "unhealthy";
  latencyMs?: number;
  message?: string;
}

/** Aggregated health check response. */
export interface HealthResponse {
  status: "healthy" | "degraded" | "unhealthy";
  dependencies: DependencyHealth[];
}

/** Well-known identity headers forwarded between services. */
export const IDENTITY_HEADERS = {
  TENANT_ID: "x-tenant-id",
  USER_ID: "x-user-id",
  USER_ROLES: "x-user-roles",
  REQUEST_ID: "x-request-id",
} as const;

/** Error codes matching Go `pkg/errors` codes. */
export const ERROR_CODES = {
  VALIDATION_FAILED: "VALIDATION_FAILED",
  UNAUTHORIZED: "UNAUTHORIZED",
  FORBIDDEN: "FORBIDDEN",
  NOT_FOUND: "RESOURCE_NOT_FOUND",
  CONFLICT: "CONFLICT",
  RATE_LIMITED: "RATE_LIMITED",
  MODULE_NOT_ENABLED: "MODULE_NOT_ENABLED",
  SERVICE_UNAVAILABLE: "SERVICE_UNAVAILABLE",
  INTERNAL_ERROR: "INTERNAL_ERROR",
  BAD_REQUEST: "BAD_REQUEST",
} as const;

export type ErrorCode = (typeof ERROR_CODES)[keyof typeof ERROR_CODES];
