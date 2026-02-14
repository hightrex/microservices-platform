import express from "express";
import request from "supertest";
import { rateLimiterMiddleware } from "./rate-limiter";
import type { GatewayConfig } from "../config";

// Mock Redis
const mockPipeline = {
  zremrangebyscore: jest.fn().mockReturnThis(),
  zadd: jest.fn().mockReturnThis(),
  zcard: jest.fn().mockReturnThis(),
  pexpire: jest.fn().mockReturnThis(),
  exec: jest.fn(),
};

const mockRedis = {
  pipeline: jest.fn(() => mockPipeline),
} as unknown as import("ioredis").default;

const defaultConfig = {
  rateLimitWindowMs: 60000,
  rateLimitMaxRequests: 5,
} as GatewayConfig;

function buildApp(config = defaultConfig): express.Express {
  const app = express();
  app.use(rateLimiterMiddleware(config, mockRedis));
  app.get("/api/v1/test", (_req, res) => res.json({ success: true }));
  app.post("/api/v1/auth/login", (_req, res) => res.json({ success: true }));
  return app;
}

describe("rateLimiterMiddleware", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("should allow requests within limit and set rate limit headers", async () => {
    mockPipeline.exec.mockResolvedValue([
      [null, 0],         // zremrangebyscore
      [null, 1],         // zadd
      [null, 3],         // zcard (count = 3, under limit of 5)
      [null, 1],         // pexpire
    ]);

    const app = buildApp();
    const res = await request(app).get("/api/v1/test");

    expect(res.status).toBe(200);
    expect(res.headers["x-ratelimit-limit"]).toBe("5");
    expect(res.headers["x-ratelimit-remaining"]).toBe("2");
    expect(res.headers["x-ratelimit-reset"]).toBeDefined();
  });

  it("should return 429 when limit exceeded", async () => {
    mockPipeline.exec.mockResolvedValue([
      [null, 0],
      [null, 1],
      [null, 6],         // zcard (count = 6, over limit of 5)
      [null, 1],
    ]);

    const app = buildApp();
    const res = await request(app).get("/api/v1/test");

    expect(res.status).toBe(429);
    expect(res.body.error.code).toBe("RATE_LIMITED");
    expect(res.headers["retry-after"]).toBeDefined();
    expect(res.headers["x-ratelimit-remaining"]).toBe("0");
  });

  it("should fail open on Redis errors", async () => {
    mockPipeline.exec.mockRejectedValue(new Error("Redis connection lost"));

    const app = buildApp();
    const res = await request(app).get("/api/v1/test");

    expect(res.status).toBe(200);
  });
});
