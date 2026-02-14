import { loadConfig } from "./index";

describe("loadConfig", () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
    // Required env vars
    process.env["AUTH_BASE_URL"] = "http://auth:8080";
    process.env["ORG_BASE_URL"] = "http://org:8081";
    process.env["JWT_SECRET"] = "test-secret-that-is-at-least-32-characters-long!!";
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it("should load valid config from environment", () => {
    const config = loadConfig();
    expect(config.port).toBe(3000);
    expect(config.authBaseUrl).toBe("http://auth:8080");
    expect(config.orgBaseUrl).toBe("http://org:8081");
    expect(config.jwtIssuer).toBe("auth-service");
    expect(config.environment).toBe("test"); // jest sets NODE_ENV=test
  });

  it("should use custom port from env", () => {
    process.env["PORT"] = "4000";
    const config = loadConfig();
    expect(config.port).toBe(4000);
  });

  it("should parse CORS origins from comma-separated string", () => {
    process.env["CORS_ORIGINS"] = "http://localhost:3001,http://localhost:3002";
    const config = loadConfig();
    expect(config.corsOrigins).toEqual(["http://localhost:3001", "http://localhost:3002"]);
  });

  it("should throw on missing required AUTH_BASE_URL", () => {
    delete process.env["AUTH_BASE_URL"];
    expect(() => loadConfig()).toThrow("Invalid gateway configuration");
  });

  it("should throw on JWT_SECRET shorter than 32 characters", () => {
    process.env["JWT_SECRET"] = "short";
    expect(() => loadConfig()).toThrow("Invalid gateway configuration");
  });

  it("should use default rate limit values", () => {
    const config = loadConfig();
    expect(config.rateLimitWindowMs).toBe(60000);
    expect(config.rateLimitMaxRequests).toBe(100);
  });

  it("should use custom rate limit values", () => {
    process.env["RATE_LIMIT_WINDOW_MS"] = "30000";
    process.env["RATE_LIMIT_MAX_REQUESTS"] = "50";
    const config = loadConfig();
    expect(config.rateLimitWindowMs).toBe(30000);
    expect(config.rateLimitMaxRequests).toBe(50);
  });
});
