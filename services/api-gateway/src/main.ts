import express from "express";
import helmet from "helmet";
import cors from "cors";
import Redis from "ioredis";
import { logger } from "@microservices-platform/shared";
import { loadConfig } from "./config";
import {
  requestIdMiddleware,
  requestLoggerMiddleware,
  jwtAuthMiddleware,
  rateLimiterMiddleware,
  moduleGateMiddleware,
  createCircuitBreakerRegistry,
} from "./middleware";
import { registerProxyRoutes } from "./routes/proxy";
import { healthHandler } from "./routes/health";
import { metricsHandler, metricsMiddleware } from "./routes/metrics";

/**
 * Bootstrap the API Gateway.
 */
async function main(): Promise<void> {
  // 1. Load and validate configuration
  const config = loadConfig();
  logger.info({ port: config.port, environment: config.environment }, "Starting API Gateway");

  // 2. Connect to Redis
  const redis = new Redis(config.redisUrl, {
    password: config.redisPassword || undefined,
    db: config.redisDb,
    maxRetriesPerRequest: 3,
    retryStrategy(times: number) {
      if (times > 10) return null; // Stop retrying
      return Math.min(times * 200, 2000);
    },
    lazyConnect: true,
  });

  try {
    await redis.connect();
    logger.info("Connected to Redis");
  } catch (err: unknown) {
    logger.error({ err: err instanceof Error ? err.message : "unknown" }, "Failed to connect to Redis");
    process.exit(1);
  }

  // 3. Create Express app
  const app = express();

  // --- Middleware stack (ORDER MATTERS — per TypeScript rules) ---

  // #1: Request ID / Correlation ID
  app.use(requestIdMiddleware);

  // #2: Request logging (pino structured logs)
  app.use(requestLoggerMiddleware);

  // #3: Security headers (helmet)
  app.use(helmet());

  // #4: CORS (explicit allowlist only)
  app.use(
    cors({
      origin: config.corsOrigins,
      methods: ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"],
      allowedHeaders: ["Content-Type", "Authorization", "X-Request-ID", "X-Tenant-ID"],
      exposedHeaders: [
        "X-Request-ID",
        "X-RateLimit-Limit",
        "X-RateLimit-Remaining",
        "X-RateLimit-Reset",
        "Retry-After",
      ],
      credentials: true,
      maxAge: 86400,
    }),
  );

  // #5: Request body limit
  app.use(express.json({ limit: config.bodyLimitBytes }));
  app.use(express.urlencoded({ extended: false, limit: config.bodyLimitBytes }));

  // #6-7: JWT + Tenant context — applied per-route in proxy registration

  // Prometheus metrics middleware (on all requests)
  app.use(metricsMiddleware);

  // #8: Module gating (moved to proxy routes to run AFTER auth)
  // app.use(moduleGateMiddleware(config, redis));

  // #9: Rate limiting (Redis-backed sliding window)
  app.use(rateLimiterMiddleware(config, redis));

  // --- Infrastructure endpoints (no auth required) ---
  app.get("/health", healthHandler(config, redis));
  app.get("/metrics", metricsHandler);

  // --- Circuit breaker registry ---
  const cbRegistry = createCircuitBreakerRegistry(config);

  // --- Proxy routes ---
  const jwtMiddleware = jwtAuthMiddleware(config);
  const moduleGate = moduleGateMiddleware(config, redis) as unknown as import("express").RequestHandler;
  registerProxyRoutes(app, config, cbRegistry, jwtMiddleware, moduleGate);

  // --- Catch-all 404 ---
  app.use((_req, res) => {
    res.status(404).json({
      success: false,
      error: { code: "NOT_FOUND", message: "Route not found" },
    });
  });

  // --- Global error handler ---
  app.use((err: Error, _req: express.Request, res: express.Response, _next: express.NextFunction) => {
    logger.error({ err: err.message, stack: err.stack }, "Unhandled error");
    if (!res.headersSent) {
      res.status(500).json({
        success: false,
        error: { code: "INTERNAL_ERROR", message: "Internal server error" },
      });
    }
  });

  // 4. Start server with graceful shutdown
  const server = app.listen(config.port, () => {
    logger.info({ port: config.port }, `API Gateway listening on port ${config.port}`);
  });

  // Graceful shutdown
  const shutdown = async (signal: string) => {
    logger.info({ signal }, "Received shutdown signal, closing gracefully...");
    server.close(() => {
      logger.info("HTTP server closed");
    });
    try {
      await redis.quit();
      logger.info("Redis connection closed");
    } catch {
      // Ignore cleanup errors
    }
    process.exit(0);
  };

  process.on("SIGTERM", () => void shutdown("SIGTERM"));
  process.on("SIGINT", () => void shutdown("SIGINT"));
}

main().catch((err: unknown) => {
  logger.fatal({ err: err instanceof Error ? err.message : "unknown" }, "Fatal error starting API Gateway");
  process.exit(1);
});
