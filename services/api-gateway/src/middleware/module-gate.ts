import type { Request, Response, NextFunction } from "express";
import type Redis from "ioredis";
import { ERROR_CODES, logger } from "@microservices-platform/shared";
import type { GatewayConfig } from "../config";

/**
 * Maps route prefixes to module names.
 * Only routes whose prefix appears here are subject to module gating.
 */
const ROUTE_MODULE_MAP: Record<string, string> = {
  "/api/v1/notifications": "notifications",
  "/api/v1/billing": "billing",
  "/api/v1/files": "file_management",
  "/api/v1/audit": "audit_logging",
  "/api/v1/analytics": "analytics",
};

/**
 * Fetch enabled modules for an org from the Org service, with Redis caching.
 */
async function getEnabledModules(
  orgId: string,
  config: GatewayConfig,
  redis: Redis,
): Promise<string[]> {
  const cacheKey = `gateway:modules:${orgId}`;

  // Try cache first
  try {
    const cached = await redis.get(cacheKey);
    if (cached !== null) {
      return JSON.parse(cached) as string[];
    }
  } catch {
    // Cache miss or error — fall through to upstream
  }

  // Call org service
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 5000);

  try {
    const url = `${config.orgBaseUrl}/api/v1/organizations/${orgId}/modules`;
    const response = await fetch(url, {
      method: "GET",
      signal: controller.signal,
      headers: {
        Accept: "application/json",
        "x-tenant-id": orgId,
        "x-service-auth": "gateway-internal",
      },
    });

    clearTimeout(timer);

    if (!response.ok) {
      logger.warn({ orgId, status: response.status }, "Failed to fetch modules from org service");
      return [];
    }

    const body = (await response.json()) as {
      success: boolean;
      data: { modules?: Array<{ module_name: string; enabled: boolean }> };
    };

    const enabledModules = (body.data?.modules ?? [])
      .filter((m) => m.enabled)
      .map((m) => m.module_name);

    // Cache with TTL
    try {
      await redis.set(cacheKey, JSON.stringify(enabledModules), "EX", config.moduleCacheTtlSeconds);
    } catch {
      // Non-fatal cache write failure
    }

    return enabledModules;
  } catch (err: unknown) {
    clearTimeout(timer);
    logger.warn({ orgId, err: err instanceof Error ? err.message : "unknown" }, "Module fetch error");
    return [];
  }
}

/**
 * Middleware #8: Module gating.
 *
 * Checks whether the requested route's module is enabled for the org.
 * Returns 403 MODULE_NOT_ENABLED if the module is disabled.
 */
export function moduleGateMiddleware(config: GatewayConfig, redis: Redis) {
  return async (req: Request, res: Response, next: NextFunction): Promise<void> => {
    // Find the matching module for this route
    const matchedPrefix = Object.keys(ROUTE_MODULE_MAP).find((prefix) =>
      req.path.startsWith(prefix),
    );

    if (!matchedPrefix) {
      // Route not subject to module gating (auth, org, users, etc.)
      next();
      return;
    }

    const moduleName = ROUTE_MODULE_MAP[matchedPrefix];
    if (!moduleName) {
      next();
      return;
    }

    const tenantId = req.headers["x-tenant-id"] as string | undefined;
    if (!tenantId) {
      res.status(401).json({
        success: false,
        error: { code: ERROR_CODES.UNAUTHORIZED, message: "Tenant context required" },
      });
      return;
    }

    const enabledModules = await getEnabledModules(tenantId, config, redis);
    if (!enabledModules.includes(moduleName)) {
      res.status(403).json({
        success: false,
        error: {
          code: ERROR_CODES.MODULE_NOT_ENABLED,
          message: `Module '${moduleName}' is not enabled for this organization`,
        },
      });
      return;
    }

    next();
  };
}
