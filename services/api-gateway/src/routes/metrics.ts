import type { Request, Response, NextFunction } from "express";
import client from "prom-client";

// Create a dedicated registry to avoid polluting the default
const register = new client.Registry();
register.setDefaultLabels({ service: "api-gateway" });

// Collect default Node.js metrics
client.collectDefaultMetrics({ register });

// Custom metrics
const httpRequestDuration = new client.Histogram({
  name: "gateway_request_duration_seconds",
  help: "Duration of HTTP requests in seconds",
  labelNames: ["method", "path", "status"] as const,
  buckets: [0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10],
  registers: [register],
});

const httpRequestsTotal = new client.Counter({
  name: "gateway_requests_total",
  help: "Total number of HTTP requests",
  labelNames: ["method", "path", "status"] as const,
  registers: [register],
});

const rateLimitTotal = new client.Counter({
  name: "gateway_ratelimit_total",
  help: "Total number of rate-limited requests",
  labelNames: ["method", "path"] as const,
  registers: [register],
});

/**
 * Middleware: record request duration and count for Prometheus.
 */
export function metricsMiddleware(req: Request, res: Response, next: NextFunction): void {
  const start = process.hrtime.bigint();

  res.on("finish", () => {
    const durationNs = Number(process.hrtime.bigint() - start);
    const durationSeconds = durationNs / 1e9;

    // Normalize path to avoid high cardinality (strip UUIDs)
    const normalizedPath = req.route?.path ?? req.path.replace(
      /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi,
      ":id",
    );

    httpRequestDuration.observe(
      { method: req.method, path: normalizedPath, status: res.statusCode.toString() },
      durationSeconds,
    );

    httpRequestsTotal.inc(
      { method: req.method, path: normalizedPath, status: res.statusCode.toString() },
    );

    if (res.statusCode === 429) {
      rateLimitTotal.inc({ method: req.method, path: normalizedPath });
    }
  });

  next();
}

/**
 * Handler: expose Prometheus metrics at /metrics.
 */
export async function metricsHandler(_req: Request, res: Response): Promise<void> {
  res.setHeader("Content-Type", register.contentType);
  const metrics = await register.metrics();
  res.end(metrics);
}
