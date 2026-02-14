import type { Request, Response, NextFunction } from "express";
import { childLogger } from "@microservices-platform/shared";

/**
 * Middleware #2: Structured request logging via pino.
 * Logs method, path, status, and duration. Redacts sensitive headers.
 */
export function requestLoggerMiddleware(req: Request, res: Response, next: NextFunction): void {
  const start = Date.now();
  const requestId = (req.headers["x-request-id"] as string) ?? "unknown";
  const tenantId = req.headers["x-tenant-id"] as string | undefined;
  const log = childLogger(requestId, tenantId);

  // Attach logger to request for downstream use
  (req as unknown as Record<string, unknown>)["log"] = log;

  res.on("finish", () => {
    const duration = Date.now() - start;
    const level = res.statusCode >= 500 ? "error" : res.statusCode >= 400 ? "warn" : "info";
    log[level](
      {
        method: req.method,
        path: req.originalUrl,
        statusCode: res.statusCode,
        durationMs: duration,
        contentLength: res.getHeader("content-length"),
      },
      `${req.method} ${req.originalUrl} ${res.statusCode} ${duration}ms`,
    );
  });

  next();
}
