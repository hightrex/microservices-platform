import type { Request, Response, NextFunction } from "express";
import crypto from "crypto";

/**
 * Middleware #1: Request ID / Correlation ID.
 * Generates `X-Request-ID` if missing, propagates on response.
 */
export function requestIdMiddleware(req: Request, res: Response, next: NextFunction): void {
  const existing = req.headers["x-request-id"];
  const requestId = (typeof existing === "string" && existing.length > 0)
    ? existing
    : crypto.randomUUID();

  req.headers["x-request-id"] = requestId;
  res.setHeader("x-request-id", requestId);
  next();
}
