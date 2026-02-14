import { IDENTITY_HEADERS, ERROR_CODES } from "./types";
import type {
  ApiSuccessResponse,
  ApiErrorResponse,
  JwtClaims,
  HealthResponse,
  PaginatedResponse,
} from "./types";

describe("IDENTITY_HEADERS", () => {
  it("should define all required header constants", () => {
    expect(IDENTITY_HEADERS.TENANT_ID).toBe("x-tenant-id");
    expect(IDENTITY_HEADERS.USER_ID).toBe("x-user-id");
    expect(IDENTITY_HEADERS.USER_ROLES).toBe("x-user-roles");
    expect(IDENTITY_HEADERS.REQUEST_ID).toBe("x-request-id");
  });
});

describe("ERROR_CODES", () => {
  it("should define all required error codes", () => {
    expect(ERROR_CODES.VALIDATION_FAILED).toBe("VALIDATION_FAILED");
    expect(ERROR_CODES.UNAUTHORIZED).toBe("UNAUTHORIZED");
    expect(ERROR_CODES.FORBIDDEN).toBe("FORBIDDEN");
    expect(ERROR_CODES.NOT_FOUND).toBe("RESOURCE_NOT_FOUND");
    expect(ERROR_CODES.RATE_LIMITED).toBe("RATE_LIMITED");
    expect(ERROR_CODES.MODULE_NOT_ENABLED).toBe("MODULE_NOT_ENABLED");
    expect(ERROR_CODES.SERVICE_UNAVAILABLE).toBe("SERVICE_UNAVAILABLE");
    expect(ERROR_CODES.INTERNAL_ERROR).toBe("INTERNAL_ERROR");
  });
});

describe("type shapes (compile-time verification)", () => {
  it("should allow constructing a success response", () => {
    const resp: ApiSuccessResponse<{ id: string }> = {
      success: true,
      data: { id: "123" },
      message: "Created",
    };
    expect(resp.success).toBe(true);
    expect(resp.data.id).toBe("123");
  });

  it("should allow constructing an error response", () => {
    const resp: ApiErrorResponse = {
      success: false,
      error: { code: "VALIDATION_FAILED", message: "Invalid input" },
    };
    expect(resp.success).toBe(false);
    expect(resp.error.code).toBe("VALIDATION_FAILED");
  });

  it("should allow constructing JWT claims", () => {
    const claims: JwtClaims = {
      sub: "user-id",
      tid: "tenant-id",
      org: "org-id",
      roles: ["admin"],
      iss: "auth-service",
      iat: 1000,
      exp: 2000,
      jti: "token-id",
    };
    expect(claims.sub).toBe("user-id");
  });

  it("should allow constructing a health response", () => {
    const health: HealthResponse = {
      status: "healthy",
      dependencies: [
        { name: "auth-service", status: "healthy", latencyMs: 5 },
        { name: "org-service", status: "degraded", message: "slow" },
      ],
    };
    expect(health.dependencies).toHaveLength(2);
  });

  it("should allow constructing a paginated response", () => {
    const paginated: PaginatedResponse<{ name: string }> = {
      items: [{ name: "test" }],
      total: 1,
      page: 1,
      pageSize: 10,
      totalPages: 1,
    };
    expect(paginated.items).toHaveLength(1);
  });
});
