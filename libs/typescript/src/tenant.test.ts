import {
  identityFromClaims,
  identityFromHeaders,
  setIdentityHeaders,
  isValidUUID,
} from "./tenant";
import type { JwtClaims } from "./types";
import { IDENTITY_HEADERS } from "./types";

const VALID_UUID = "550e8400-e29b-41d4-a716-446655440000";
const VALID_UUID_2 = "660e8400-e29b-41d4-a716-446655440001";

describe("identityFromClaims", () => {
  it("should extract identity from valid JWT claims", () => {
    const claims: JwtClaims = {
      sub: VALID_UUID,
      tid: VALID_UUID_2,
      org: VALID_UUID_2,
      roles: ["org_admin", "member"],
      iss: "auth-service",
      iat: Math.floor(Date.now() / 1000),
      exp: Math.floor(Date.now() / 1000) + 900,
      jti: "token-123",
    };

    const identity = identityFromClaims(claims);
    expect(identity).not.toBeNull();
    expect(identity?.tenantId).toBe(VALID_UUID_2);
    expect(identity?.userId).toBe(VALID_UUID);
    expect(identity?.roles).toEqual(["org_admin", "member"]);
  });

  it("should return null for missing tenant_id", () => {
    const claims = {
      sub: VALID_UUID,
      tid: "",
      org: "",
      roles: [],
      iss: "auth-service",
      iat: 0,
      exp: 0,
      jti: "",
    };
    expect(identityFromClaims(claims)).toBeNull();
  });

  it("should return null for invalid UUID in sub", () => {
    const claims = {
      sub: "not-a-uuid",
      tid: VALID_UUID,
      org: VALID_UUID,
      roles: [],
      iss: "auth-service",
      iat: 0,
      exp: 0,
      jti: "",
    };
    expect(identityFromClaims(claims)).toBeNull();
  });
});

describe("identityFromHeaders", () => {
  it("should extract identity from valid headers", () => {
    const headers: Record<string, string> = {
      [IDENTITY_HEADERS.TENANT_ID]: VALID_UUID,
      [IDENTITY_HEADERS.USER_ID]: VALID_UUID_2,
      [IDENTITY_HEADERS.USER_ROLES]: "org_owner,manager",
      [IDENTITY_HEADERS.REQUEST_ID]: "req-1",
    };

    const identity = identityFromHeaders(headers);
    expect(identity).not.toBeNull();
    expect(identity?.tenantId).toBe(VALID_UUID);
    expect(identity?.userId).toBe(VALID_UUID_2);
    expect(identity?.roles).toEqual(["org_owner", "manager"]);
    expect(identity?.requestId).toBe("req-1");
  });

  it("should return null for missing tenant_id header", () => {
    const headers: Record<string, string> = {
      [IDENTITY_HEADERS.USER_ID]: VALID_UUID_2,
    };
    expect(identityFromHeaders(headers)).toBeNull();
  });

  it("should return null for invalid UUID in tenant_id", () => {
    const headers: Record<string, string> = {
      [IDENTITY_HEADERS.TENANT_ID]: "invalid",
      [IDENTITY_HEADERS.USER_ID]: VALID_UUID,
    };
    expect(identityFromHeaders(headers)).toBeNull();
  });

  it("should handle empty roles", () => {
    const headers: Record<string, string> = {
      [IDENTITY_HEADERS.TENANT_ID]: VALID_UUID,
      [IDENTITY_HEADERS.USER_ID]: VALID_UUID_2,
      [IDENTITY_HEADERS.USER_ROLES]: "",
    };

    const identity = identityFromHeaders(headers);
    expect(identity?.roles).toEqual([]);
  });
});

describe("setIdentityHeaders", () => {
  it("should set identity headers on outgoing request", () => {
    const base = { "content-type": "application/json" };
    const identity = {
      tenantId: VALID_UUID,
      userId: VALID_UUID_2,
      roles: ["admin", "member"],
      requestId: "req-abc",
    };

    const result = setIdentityHeaders(base, identity);
    expect(result[IDENTITY_HEADERS.TENANT_ID]).toBe(VALID_UUID);
    expect(result[IDENTITY_HEADERS.USER_ID]).toBe(VALID_UUID_2);
    expect(result[IDENTITY_HEADERS.USER_ROLES]).toBe("admin,member");
    expect(result[IDENTITY_HEADERS.REQUEST_ID]).toBe("req-abc");
    expect(result["content-type"]).toBe("application/json");
  });
});

describe("isValidUUID", () => {
  it("should validate correct UUIDs", () => {
    expect(isValidUUID(VALID_UUID)).toBe(true);
    expect(isValidUUID("00000000-0000-0000-0000-000000000000")).toBe(true);
  });

  it("should reject invalid UUIDs", () => {
    expect(isValidUUID("")).toBe(false);
    expect(isValidUUID("not-a-uuid")).toBe(false);
    expect(isValidUUID("550e8400-e29b-41d4-a716")).toBe(false);
  });
});
