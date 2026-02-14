import express from "express";
import request from "supertest";
import { moduleGateMiddleware } from "./module-gate";
import type { GatewayConfig } from "../config";

// Mock Redis
const mockRedis = {
  get: jest.fn(),
  set: jest.fn(),
} as unknown as import("ioredis").default;

const defaultConfig = {
  orgBaseUrl: "http://org:8081",
  moduleCacheTtlSeconds: 300,
} as GatewayConfig;

function buildApp(): express.Express {
  const app = express();
  // Simulate tenant header being set (by JWT middleware)
  app.use((req, _res, next) => {
    if (req.headers["x-tenant-id"]) {
      // Already set
    }
    next();
  });
  app.use(moduleGateMiddleware(defaultConfig, mockRedis));
  app.get("/api/v1/auth/login", (_req, res) => res.json({ success: true }));
  app.get("/api/v1/notifications", (_req, res) => res.json({ success: true }));
  app.get("/api/v1/billing/invoices", (_req, res) => res.json({ success: true }));
  app.get("/api/v1/organizations/:id", (_req, res) => res.json({ success: true }));
  return app;
}

describe("moduleGateMiddleware", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("should pass through routes not subject to module gating", async () => {
    const app = buildApp();
    const res = await request(app)
      .get("/api/v1/auth/login")
      .set("x-tenant-id", "550e8400-e29b-41d4-a716-446655440000");

    expect(res.status).toBe(200);
    expect(mockRedis.get).not.toHaveBeenCalled();
  });

  it("should pass through organization routes without gating", async () => {
    const app = buildApp();
    const res = await request(app)
      .get("/api/v1/organizations/123")
      .set("x-tenant-id", "550e8400-e29b-41d4-a716-446655440000");

    expect(res.status).toBe(200);
  });

  it("should allow gated route when module is enabled (from cache)", async () => {
    (mockRedis.get as jest.Mock).mockResolvedValue(JSON.stringify(["notifications", "billing"]));

    const app = buildApp();
    const res = await request(app)
      .get("/api/v1/notifications")
      .set("x-tenant-id", "550e8400-e29b-41d4-a716-446655440000");

    expect(res.status).toBe(200);
  });

  it("should deny gated route when module is not enabled (from cache)", async () => {
    (mockRedis.get as jest.Mock).mockResolvedValue(JSON.stringify(["billing"]));

    const app = buildApp();
    const res = await request(app)
      .get("/api/v1/notifications")
      .set("x-tenant-id", "550e8400-e29b-41d4-a716-446655440000");

    expect(res.status).toBe(403);
    expect(res.body.error.code).toBe("MODULE_NOT_ENABLED");
    expect(res.body.error.message).toContain("notifications");
  });

  it("should reject gated route without tenant context", async () => {
    const app = buildApp();
    const res = await request(app).get("/api/v1/notifications");

    expect(res.status).toBe(401);
    expect(res.body.error.code).toBe("UNAUTHORIZED");
  });
});
