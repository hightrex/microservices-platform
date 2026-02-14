import { logger, createLogger, childLogger, REDACT_PATHS } from "./logger";

describe("logger", () => {
  it("should be a pino logger instance", () => {
    expect(logger).toBeDefined();
    expect(typeof logger.info).toBe("function");
    expect(typeof logger.error).toBe("function");
    expect(typeof logger.warn).toBe("function");
    expect(typeof logger.debug).toBe("function");
  });

  it("should create a new logger with custom level", () => {
    const custom = createLogger({ level: "warn" });
    expect(custom).toBeDefined();
    expect(custom.level).toBe("warn");
  });

  it("should create child logger with request context", () => {
    const child = childLogger("req-123", "tenant-abc");
    expect(child).toBeDefined();
    expect(typeof child.info).toBe("function");
  });

  it("should create child logger without tenant", () => {
    const child = childLogger("req-456");
    expect(child).toBeDefined();
  });

  it("should create child logger with extra bindings", () => {
    const child = childLogger("req-789", "tenant-xyz", { userId: "user-1" });
    expect(child).toBeDefined();
  });

  it("should define redact paths for sensitive fields", () => {
    expect(REDACT_PATHS).toContain("req.headers.authorization");
    expect(REDACT_PATHS).toContain("password");
    expect(REDACT_PATHS).toContain("token");
    expect(REDACT_PATHS).toContain("refreshToken");
    expect(REDACT_PATHS).toContain("apiKey");
    expect(REDACT_PATHS).toContain("secret");
  });
});
