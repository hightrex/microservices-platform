import type { Request, Response } from "express";
import { aggregateHealth, type HealthCheckTarget } from "@microservices-platform/shared";
import type Redis from "ioredis";
import type { GatewayConfig } from "../config";

/**
 * Build the health check handler.
 * Aggregates health of Auth + Org services and Redis connectivity.
 */
export function healthHandler(config: GatewayConfig, redis: Redis) {
  const targets: HealthCheckTarget[] = [
    { name: "auth-service", url: `${config.authBaseUrl}/health`, timeoutMs: 3000 },
    { name: "org-service", url: `${config.orgBaseUrl}/health`, timeoutMs: 3000 },
  ];

  return async (_req: Request, res: Response): Promise<void> => {
    // Check downstream services
    const serviceHealth = await aggregateHealth(targets);

    // Check Redis connectivity
    let redisStatus: "healthy" | "unhealthy" = "unhealthy";
    try {
      const pong = await redis.ping();
      redisStatus = pong === "PONG" ? "healthy" : "unhealthy";
    } catch {
      redisStatus = "unhealthy";
    }

    serviceHealth.dependencies.push({
      name: "redis",
      status: redisStatus,
    });

    // Overall status
    if (redisStatus === "unhealthy") {
      serviceHealth.status = "unhealthy";
    }

    const httpStatus = serviceHealth.status === "healthy" ? 200 : serviceHealth.status === "degraded" ? 200 : 503;
    res.status(httpStatus).json(serviceHealth);
  };
}
