/** Barrel exports for the shared TypeScript library. */

export { logger, createLogger, childLogger, REDACT_PATHS } from "./logger";
export type { Logger } from "./logger";

export {
  AppError,
  badRequest,
  validationFailed,
  unauthorized,
  forbidden,
  notFound,
  conflict,
  rateLimited,
  moduleNotEnabled,
  serviceUnavailable,
  internalError,
  toErrorResponse,
} from "./errors";

export { checkDependency, aggregateHealth } from "./health";
export type { HealthCheckTarget } from "./health";

export {
  identityFromClaims,
  identityFromHeaders,
  identityFromRequest,
  setIdentityHeaders,
  isValidUUID,
} from "./tenant";
export type { RequestIdentity } from "./tenant";

export {
  IDENTITY_HEADERS,
  ERROR_CODES,
} from "./types";
export type {
  ApiSuccessResponse,
  ApiErrorDetail,
  ApiErrorResponse,
  ApiResponse,
  JwtClaims,
  PaginationParams,
  PaginatedResponse,
  DependencyHealth,
  HealthResponse,
  ErrorCode,
} from "./types";
