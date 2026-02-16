import type { Request, Response, NextFunction } from "express";
import type Redis from "ioredis";
import { ERROR_CODES, logger } from "@microservices-platform/shared";
import type { GatewayConfig } from "../config";

/** Per-endpoint rate limit overrides. */
const ENDPOINT_OVERRIDES: Record<string, { windowMs: number; max: number }> = {
  "POST:/api/v1/auth/login": { windowMs: 60_000, max: 10 },
  "POST:/api/v1/auth/register": { windowMs: 60_000, max: 5 },
  "POST:/api/v1/auth/mfa/verify": { windowMs: 60_000, max: 10 },
};

/**
 * Middleware #9: Redis-backed sliding window rate limiter.
 *
 * - Per-org limits from plan (default from config).
 * - Per-endpoint overrides for auth routes.
 * - Returns X-RateLimit-* headers + Retry-After on 429.
 */
export function rateLimiterMiddleware(config: GatewayConfig, redis: Redis) {
  return async (req: Request, res: Response, next: NextFunction): Promise<void> => {
    try {
      const tenantId = req.headers["x-tenant-id"] as string | undefined;
      const identifier = tenantId ?? req.ip ?? "anonymous";

      // Determine limits: per-endpoint override or default
      const endpointKey = `${req.method}:${req.path}`;
      const override = ENDPOINT_OVERRIDES[endpointKey];
      const windowMs = override?.windowMs ?? config.rateLimitWindowMs;
      const maxRequests = override?.max ?? config.rateLimitMaxRequests;

      const now = Date.now();
      const windowStart = now - windowMs;
      const redisKey = `ratelimit:${identifier}:${endpointKey}`;

      // Sliding window: use sorted set with timestamps as scores
      const pipeline = redis.pipeline();
      pipeline.zremrangebyscore(redisKey, 0, windowStart);
      pipeline.zadd(redisKey, now, `${now}:${Math.random().toString(36).slice(2, 8)}`);
      pipeline.zcard(redisKey);
      pipeline.pexpire(redisKey, windowMs);

      const results = await pipeline.exec();
      // results[2] = [err, count]
      const countResult = results?.[2];
      const currentCount = (countResult && !countResult[0]) ? (countResult[1] as number) : 0;

      const remaining = Math.max(0, maxRequests - currentCount);
      const resetEpoch = Math.ceil((now + windowMs) / 1000);

      // Set rate limit headers on every response
      res.setHeader("X-RateLimit-Limit", maxRequests);
      res.setHeader("X-RateLimit-Remaining", remaining);
      res.setHeader("X-RateLimit-Reset", resetEpoch);

      if (currentCount > maxRequests) {
        const retryAfter = Math.ceil(windowMs / 1000);
        res.setHeader("Retry-After", retryAfter);
        res.status(429).json({
          success: false,
          error: {
            code: ERROR_CODES.RATE_LIMITED,
            message: "Too many requests",
            details: `Retry after ${retryAfter} seconds`,
          },
        });
        return;
      }

      next();
    } catch (err) {
      // Fail closed on Redis errors per foundation rules: "Fail closed, not open"
      logger.error({ err }, "Rate limiter Redis error — rejecting request (fail closed)");
      res.status(503).json({
        success: false,
        error: {
          code: ERROR_CODES.SERVICE_UNAVAILABLE,
          message: "Service temporarily unavailable",
        },
      });
      return;
    }
  };
}
