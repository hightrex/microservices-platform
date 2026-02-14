import type { Express, Request, Response } from "express";
import { createProxyMiddleware, type Options as ProxyOptions } from "http-proxy-middleware";
import { ERROR_CODES, IDENTITY_HEADERS, logger } from "@microservices-platform/shared";
import type { GatewayConfig } from "../config";
import type { CircuitBreakerRegistry } from "../middleware/circuit-breaker";

/** Hop-by-hop headers that must NOT be forwarded. */
const HOP_BY_HOP = new Set([
  "connection",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailers",
  "transfer-encoding",
  "upgrade",
]);

/**
 * Build a proxy options object for a downstream service.
 */
function buildProxyOptions(
  serviceName: string,
  target: string,
  _cbRegistry: CircuitBreakerRegistry,
): ProxyOptions {
  return {
    target,
    changeOrigin: true,
    timeout: 30_000,
    proxyTimeout: 30_000,
    on: {
      proxyReq(proxyReq, req) {
        const incomingReq = req as Request;

        // Reconstruct the full original path. Express strips the mount prefix
        // when using `app.use('/prefix', proxy)`, so we restore it here.
        // Handle trailing-slash normalization to avoid Gin 301 redirects:
        //   req.url="/"         → suffix=""         → "/api/v1/users"
        //   req.url="/?page=1"  → suffix="?page=1"  → "/api/v1/users?page=1"
        //   req.url="/abc"      → suffix="/abc"      → "/api/v1/users/abc"
        //   req.url="/abc?x=1"  → suffix="/abc?x=1"  → "/api/v1/users/abc?x=1"
        let suffix = incomingReq.url;
        if (suffix === "/") {
          suffix = "";
        } else if (suffix.startsWith("/?")) {
          suffix = suffix.slice(1); // "/?page=1" → "?page=1"
        }
        const fullPath = incomingReq.baseUrl + suffix;
        proxyReq.path = fullPath || "/";

        // Forward identity headers (set by JWT middleware)
        const tenantId = incomingReq.headers[IDENTITY_HEADERS.TENANT_ID];
        if (typeof tenantId === "string") proxyReq.setHeader(IDENTITY_HEADERS.TENANT_ID, tenantId);

        const userId = incomingReq.headers[IDENTITY_HEADERS.USER_ID];
        if (typeof userId === "string") proxyReq.setHeader(IDENTITY_HEADERS.USER_ID, userId);

        const userRoles = incomingReq.headers[IDENTITY_HEADERS.USER_ROLES];
        if (typeof userRoles === "string") proxyReq.setHeader(IDENTITY_HEADERS.USER_ROLES, userRoles);

        const requestId = incomingReq.headers[IDENTITY_HEADERS.REQUEST_ID];
        if (typeof requestId === "string") proxyReq.setHeader(IDENTITY_HEADERS.REQUEST_ID, requestId);

        // Forward the Authorization header to downstream services.
        // The gateway has already validated the JWT and set identity headers.
        // Downstream services still use their own JWT middleware in Phase 1,
        // so we forward the token for backward compatibility.
        // TODO(phase-2): once downstream services trust gateway identity
        // headers exclusively, strip Authorization here.

        // Strip hop-by-hop headers
        for (const header of HOP_BY_HOP) {
          proxyReq.removeHeader(header);
        }

        // Fix body forwarding: express.json() consumes the stream, so we
        // must re-write the parsed body onto the proxy request.
        if (incomingReq.body && Object.keys(incomingReq.body as Record<string, unknown>).length > 0) {
          const bodyData = JSON.stringify(incomingReq.body);
          proxyReq.setHeader("Content-Type", "application/json");
          proxyReq.setHeader("Content-Length", Buffer.byteLength(bodyData));
          proxyReq.write(bodyData);
        }
      },
      error(err, _req, res) {
        logger.error({ service: serviceName, err: err.message }, "Proxy error");
        if (res && "writeHead" in res && typeof res.writeHead === "function") {
          const httpRes = res as import("http").ServerResponse;
          if (!httpRes.headersSent) {
            httpRes.writeHead(502);
            httpRes.end(
              JSON.stringify({
                success: false,
                error: { code: ERROR_CODES.SERVICE_UNAVAILABLE, message: `${serviceName} is unavailable` },
              }),
            );
          }
        }
      },
    },
  };
}

/**
 * Service route mapping configuration.
 */
interface ServiceRoute {
  /** Route prefix to match (e.g. "/api/v1/auth"). */
  prefix: string;
  /** Downstream service name. */
  serviceName: string;
  /** Target base URL. */
  target: string;
  /** Whether this route requires JWT authentication. */
  requiresAuth: boolean;
  /** Whether the service is available (false = returns 503 placeholder). */
  available: boolean;
}

/**
 * Register all proxy routes on the Express app.
 */
export function registerProxyRoutes(
  app: Express,
  config: GatewayConfig,
  cbRegistry: CircuitBreakerRegistry,
  jwtMiddleware: ReturnType<typeof import("../middleware/jwt-auth").jwtAuthMiddleware>,
  moduleGateMiddleware: import("express").RequestHandler,
): void {
  const routes: ServiceRoute[] = [
    // Phase 1 routes — fully available
    { prefix: "/api/v1/auth", serviceName: "auth-service", target: config.authBaseUrl, requiresAuth: false, available: true },
    { prefix: "/api/v1/users", serviceName: "auth-service", target: config.authBaseUrl, requiresAuth: true, available: true },
    { prefix: "/api/v1/organizations", serviceName: "org-service", target: config.orgBaseUrl, requiresAuth: true, available: true },
    { prefix: "/api/v1/plans", serviceName: "org-service", target: config.orgBaseUrl, requiresAuth: true, available: true },

    // Phase 2/3 placeholder routes — return 503
    { prefix: "/api/v1/notifications", serviceName: "notification-service", target: "", requiresAuth: true, available: false },
    { prefix: "/api/v1/billing", serviceName: "billing-service", target: "", requiresAuth: true, available: false },
    { prefix: "/api/v1/files", serviceName: "file-service", target: "", requiresAuth: true, available: false },
    { prefix: "/api/v1/audit", serviceName: "audit-service", target: "", requiresAuth: true, available: false },
    { prefix: "/api/v1/analytics", serviceName: "analytics-service", target: "", requiresAuth: true, available: false },
  ];

  for (const route of routes) {
    const middlewares: Array<import("express").RequestHandler> = [];

    // Auth routes that need JWT: /auth/logout, /auth/mfa/*, /auth/refresh are protected
    // /auth/login, /auth/register are public
    if (route.requiresAuth) {
      middlewares.push(jwtMiddleware);
    }

    // Apply module gating after auth (so we have identity/tenant context)
    middlewares.push(moduleGateMiddleware);

    if (!route.available) {
      // Placeholder: service not yet implemented
      app.use(route.prefix, ...middlewares, (_req: Request, res: Response) => {
        res.status(503).json({
          success: false,
          error: {
            code: ERROR_CODES.SERVICE_UNAVAILABLE,
            message: `${route.serviceName} is not yet available`,
          },
        });
      });
      continue;
    }

    // Register circuit breaker for the service
    cbRegistry.register(route.serviceName, route.target);

    // Create proxy middleware
    const proxyOptions = buildProxyOptions(route.serviceName, route.target, cbRegistry);
    const proxy = createProxyMiddleware(proxyOptions);

    app.use(route.prefix, ...middlewares, proxy);
  }

  // Special handling: /auth routes have mixed auth requirements.
  // The auth service proxy is registered without JWT above, but some auth
  // endpoints (logout, mfa/setup, mfa/verify) need JWT. The auth service
  // itself enforces this via its own middleware, so the gateway allows
  // passthrough. The JWT is still forwarded via headers for auth-service
  // to validate internally.
}
