import type { Request, Response, NextFunction } from "express";
import jwt from "jsonwebtoken";
import type { JwtClaims } from "@microservices-platform/shared";
import { ERROR_CODES, IDENTITY_HEADERS } from "@microservices-platform/shared";
import type { GatewayConfig } from "../config";

/**
 * Middleware #6: JWT validation.
 *
 * - Extracts Bearer token from Authorization header.
 * - Verifies signature, issuer, and expiry.
 * - Extracts claims (sub, tid, org, roles).
 * - Sets downstream identity headers (overwrites any incoming spoofed headers).
 */
export function jwtAuthMiddleware(config: GatewayConfig) {
  return (req: Request, res: Response, next: NextFunction): void => {
    const authHeader = req.headers.authorization;
    if (!authHeader || !authHeader.startsWith("Bearer ")) {
      res.status(401).json({
        success: false,
        error: { code: ERROR_CODES.UNAUTHORIZED, message: "Missing or invalid Authorization header" },
      });
      return;
    }

    const token = authHeader.slice(7);

    try {
      const decoded = jwt.verify(token, config.jwtSecret, {
        issuer: config.jwtIssuer,
        algorithms: ["HS256"],
      }) as JwtClaims;

      // Validate required claims
      if (!decoded.sub || !decoded.tid) {
        res.status(401).json({
          success: false,
          error: { code: ERROR_CODES.UNAUTHORIZED, message: "Token missing required claims" },
        });
        return;
      }

      // Spoof protection: overwrite any incoming identity headers with JWT-derived values
      req.headers[IDENTITY_HEADERS.TENANT_ID] = decoded.tid;
      req.headers[IDENTITY_HEADERS.USER_ID] = decoded.sub;
      req.headers[IDENTITY_HEADERS.USER_ROLES] = Array.isArray(decoded.roles)
        ? decoded.roles.join(",")
        : "";

      // Store decoded claims on request for downstream middleware
      (req as unknown as Record<string, unknown>)["claims"] = decoded;

      next();
    } catch (err: unknown) {
      const message =
        err instanceof jwt.TokenExpiredError
          ? "Token has expired"
          : err instanceof jwt.JsonWebTokenError
            ? "Invalid token"
            : "Token validation failed";

      res.status(401).json({
        success: false,
        error: { code: ERROR_CODES.UNAUTHORIZED, message },
      });
    }
  };
}
