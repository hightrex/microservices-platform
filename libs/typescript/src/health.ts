import type { DependencyHealth, HealthResponse } from "./types";
import { logger } from "./logger";

interface HealthCheckTarget {
  /** Human-readable name (e.g. "auth-service"). */
  name: string;
  /** Full URL to the dependency health endpoint (e.g. "http://auth-service:8080/health"). */
  url: string;
  /** Request timeout in ms (default 3000). */
  timeoutMs?: number;
}

/**
 * Check a single downstream dependency's health endpoint.
 */
async function checkDependency(target: HealthCheckTarget): Promise<DependencyHealth> {
  const start = Date.now();
  const timeoutMs = target.timeoutMs ?? 3000;

  try {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);

    const response = await fetch(target.url, {
      method: "GET",
      signal: controller.signal,
      headers: { Accept: "application/json" },
    });

    clearTimeout(timer);
    const latencyMs = Date.now() - start;

    if (response.ok) {
      return { name: target.name, status: "healthy", latencyMs };
    }

    return {
      name: target.name,
      status: "degraded",
      latencyMs,
      message: `HTTP ${response.status}`,
    };
  } catch (err: unknown) {
    const latencyMs = Date.now() - start;
    const message = err instanceof Error ? err.message : "Unknown error";
    logger.warn({ dependency: target.name, err: message }, "Health check failed");
    return { name: target.name, status: "unhealthy", latencyMs, message };
  }
}

/**
 * Aggregate health from multiple downstream dependencies.
 * Returns "healthy" only if ALL are healthy; "degraded" if any degraded; "unhealthy" if any unhealthy.
 */
async function aggregateHealth(targets: HealthCheckTarget[]): Promise<HealthResponse> {
  const results = await Promise.all(targets.map(checkDependency));

  let overall: HealthResponse["status"] = "healthy";
  for (const dep of results) {
    if (dep.status === "unhealthy") {
      overall = "unhealthy";
      break;
    }
    if (dep.status === "degraded") {
      overall = "degraded";
    }
  }

  return { status: overall, dependencies: results };
}

export { checkDependency, aggregateHealth };
export type { HealthCheckTarget };
