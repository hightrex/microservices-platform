import pino, { type Logger, type LoggerOptions } from "pino";

/** Fields that are always redacted from log output. */
const REDACT_PATHS: string[] = [
  "req.headers.authorization",
  "req.headers.cookie",
  "req.headers['set-cookie']",
  "authorization",
  "password",
  "token",
  "refreshToken",
  "accessToken",
  "apiKey",
  "secret",
  "mfaSecret",
];

/** Create the root application logger. Level defaults to env `LOG_LEVEL` or "info". */
function createLogger(overrides?: Partial<LoggerOptions>): Logger {
  const level =
    overrides?.level ?? process.env["LOG_LEVEL"] ?? "info";

  return pino({
    level,
    redact: {
      paths: REDACT_PATHS,
      censor: "[REDACTED]",
    },
    timestamp: pino.stdTimeFunctions.isoTime,
    formatters: {
      level(label: string) {
        return { level: label };
      },
    },
    ...overrides,
  });
}

/** Singleton root logger instance. */
const logger: Logger = createLogger();

/**
 * Build a request-scoped child logger that carries correlation context.
 *
 * @param requestId  - The `X-Request-ID` header value.
 * @param tenantId   - The `X-Tenant-ID` header value (optional).
 * @param extra      - Any additional bindings.
 */
function childLogger(
  requestId: string,
  tenantId?: string,
  extra?: Record<string, unknown>,
): Logger {
  return logger.child({
    requestId,
    ...(tenantId ? { tenantId } : {}),
    ...extra,
  });
}

export { logger, createLogger, childLogger, REDACT_PATHS };
export type { Logger };
