import type { JwtClaims } from "./types";
import { IDENTITY_HEADERS } from "./types";

/** Generic incoming request shape (framework-agnostic). */
interface IncomingRequest {
  headers: Record<string, string | string[] | undefined>;
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** Validated tenant + user identity extracted from the request. */
export interface RequestIdentity {
  tenantId: string;
  userId: string;
  roles: string[];
  requestId: string;
}

/**
 * Extract tenant and user identity from JWT claims.
 * Returns the validated identity or `null` if claims are insufficient.
 */
export function identityFromClaims(claims: JwtClaims): RequestIdentity | null {
  if (!claims.tid || !claims.sub || !UUID_RE.test(claims.tid) || !UUID_RE.test(claims.sub)) {
    return null;
  }
  return {
    tenantId: claims.tid,
    userId: claims.sub,
    roles: Array.isArray(claims.roles) ? claims.roles : [],
    requestId: claims.jti ?? "",
  };
}

/**
 * Extract tenant/user identity from trusted downstream headers.
 * Used by downstream services that trust the gateway's headers.
 */
export function identityFromHeaders(headers: Record<string, string | string[] | undefined>): RequestIdentity | null {
  const tenantId = headerValue(headers, IDENTITY_HEADERS.TENANT_ID);
  const userId = headerValue(headers, IDENTITY_HEADERS.USER_ID);
  const rolesRaw = headerValue(headers, IDENTITY_HEADERS.USER_ROLES);
  const requestId = headerValue(headers, IDENTITY_HEADERS.REQUEST_ID) ?? "";

  if (!tenantId || !userId || !UUID_RE.test(tenantId) || !UUID_RE.test(userId)) {
    return null;
  }

  const roles = rolesRaw ? rolesRaw.split(",").map((r) => r.trim()) : [];
  return { tenantId, userId, roles, requestId };
}

/**
 * Set identity headers on an outgoing (proxied) request.
 * Overwrites any existing values to prevent spoofing.
 */
export function setIdentityHeaders(
  headers: Record<string, string>,
  identity: RequestIdentity,
): Record<string, string> {
  return {
    ...headers,
    [IDENTITY_HEADERS.TENANT_ID]: identity.tenantId,
    [IDENTITY_HEADERS.USER_ID]: identity.userId,
    [IDENTITY_HEADERS.USER_ROLES]: identity.roles.join(","),
    [IDENTITY_HEADERS.REQUEST_ID]: identity.requestId,
  };
}

/**
 * Validate that a string is a valid UUID v4.
 */
export function isValidUUID(value: string): boolean {
  return UUID_RE.test(value);
}

/** Safely pull a single string value from a header map. */
function headerValue(
  headers: Record<string, string | string[] | undefined>,
  key: string,
): string | undefined {
  const v = headers[key];
  if (Array.isArray(v)) return v[0];
  return v;
}

/**
 * Framework-agnostic helper: extract identity from any request with headers.
 * Works with Express, Fastify, or any framework that exposes `req.headers`.
 */
export function identityFromRequest(req: IncomingRequest): RequestIdentity | null {
  return identityFromHeaders(req.headers);
}
