import {
  AppError,
  badRequest,
  validationFailed,
  unauthorized,
  forbidden,
  notFound,
  conflict,
  rateLimited,
  moduleNotEnabled,
  serviceUnavailable,
  internalError,
  toErrorResponse,
} from "./errors";
import { ERROR_CODES } from "./types";

describe("AppError", () => {
  it("should construct with code, message, status, and details", () => {
    const err = new AppError(ERROR_CODES.BAD_REQUEST, "bad input", 400, "field X invalid");
    expect(err.code).toBe("BAD_REQUEST");
    expect(err.message).toBe("bad input");
    expect(err.httpStatus).toBe(400);
    expect(err.details).toBe("field X invalid");
    expect(err.name).toBe("AppError");
    expect(err).toBeInstanceOf(Error);
    expect(err).toBeInstanceOf(AppError);
  });

  it("should convert to standardized API error response", () => {
    const err = new AppError(ERROR_CODES.NOT_FOUND, "User not found", 404);
    const resp = err.toResponse();
    expect(resp).toEqual({
      success: false,
      error: { code: "RESOURCE_NOT_FOUND", message: "User not found" },
    });
  });

  it("should include details in response when present", () => {
    const err = new AppError(ERROR_CODES.VALIDATION_FAILED, "Invalid", 400, "email is required");
    const resp = err.toResponse();
    expect(resp.error.details).toBe("email is required");
  });
});

describe("factory helpers", () => {
  it("badRequest", () => {
    const err = badRequest("bad");
    expect(err.httpStatus).toBe(400);
    expect(err.code).toBe(ERROR_CODES.BAD_REQUEST);
  });

  it("validationFailed", () => {
    const err = validationFailed("invalid input");
    expect(err.httpStatus).toBe(400);
    expect(err.code).toBe(ERROR_CODES.VALIDATION_FAILED);
  });

  it("unauthorized", () => {
    const err = unauthorized();
    expect(err.httpStatus).toBe(401);
    expect(err.code).toBe(ERROR_CODES.UNAUTHORIZED);
  });

  it("forbidden", () => {
    const err = forbidden();
    expect(err.httpStatus).toBe(403);
    expect(err.code).toBe(ERROR_CODES.FORBIDDEN);
  });

  it("notFound", () => {
    const err = notFound("Organization");
    expect(err.httpStatus).toBe(404);
    expect(err.message).toBe("Organization not found");
  });

  it("conflict", () => {
    const err = conflict("Already exists");
    expect(err.httpStatus).toBe(409);
  });

  it("rateLimited", () => {
    const err = rateLimited(60);
    expect(err.httpStatus).toBe(429);
    expect(err.details).toContain("60");
  });

  it("moduleNotEnabled", () => {
    const err = moduleNotEnabled("billing");
    expect(err.httpStatus).toBe(403);
    expect(err.code).toBe(ERROR_CODES.MODULE_NOT_ENABLED);
    expect(err.message).toContain("billing");
  });

  it("serviceUnavailable", () => {
    const err = serviceUnavailable("auth-service");
    expect(err.httpStatus).toBe(503);
  });

  it("internalError", () => {
    const err = internalError();
    expect(err.httpStatus).toBe(500);
  });
});

describe("toErrorResponse", () => {
  it("should convert AppError to status + body", () => {
    const err = notFound("User");
    const result = toErrorResponse(err);
    expect(result.status).toBe(404);
    expect(result.body.success).toBe(false);
    expect(result.body.error.code).toBe(ERROR_CODES.NOT_FOUND);
  });

  it("should convert unknown Error to 500", () => {
    const result = toErrorResponse(new Error("oops"));
    expect(result.status).toBe(500);
    expect(result.body.error.code).toBe(ERROR_CODES.INTERNAL_ERROR);
  });

  it("should convert non-Error to 500", () => {
    const result = toErrorResponse("string error");
    expect(result.status).toBe(500);
  });
});
