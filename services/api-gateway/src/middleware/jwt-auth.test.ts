import express from "express";
import request from "supertest";
import jwt from "jsonwebtoken";
import { jwtAuthMiddleware } from "./jwt-auth";
import type { GatewayConfig } from "../config";

const SECRET = "test-secret-that-is-at-least-32-characters-long!!";

function buildApp(config: Partial<GatewayConfig> = {}): express.Express {
  const app = express();
  const fullConfig = {
    jwtSecret: SECRET,
    jwtIssuer: "auth-service",
    ...config,
  } as GatewayConfig;

  app.use(jwtAuthMiddleware(fullConfig));
  app.get("/test", (_req, res) => {
    res.json({ success: true, tenantId: _req.headers["x-tenant-id"] });
  });
  return app;
}

function generateToken(claims: Record<string, unknown>, secret = SECRET): string {
  return jwt.sign(claims, secret, { algorithm: "HS256" });
}

describe("jwtAuthMiddleware", () => {
  it("should reject requests without Authorization header", async () => {
    const app = buildApp();
    const res = await request(app).get("/test");
    expect(res.status).toBe(401);
    expect(res.body.error.code).toBe("UNAUTHORIZED");
  });

  it("should reject non-Bearer tokens", async () => {
    const app = buildApp();
    const res = await request(app).get("/test").set("Authorization", "Basic abc123");
    expect(res.status).toBe(401);
  });

  it("should reject expired tokens", async () => {
    const app = buildApp();
    const token = generateToken({
      sub: "550e8400-e29b-41d4-a716-446655440000",
      tid: "660e8400-e29b-41d4-a716-446655440001",
      roles: ["member"],
      iss: "auth-service",
      exp: Math.floor(Date.now() / 1000) - 3600, // expired 1 hour ago
      iat: Math.floor(Date.now() / 1000) - 7200,
    });

    const res = await request(app).get("/test").set("Authorization", `Bearer ${token}`);
    expect(res.status).toBe(401);
    expect(res.body.error.message).toContain("expired");
  });

  it("should reject tokens with wrong signing key", async () => {
    const app = buildApp();
    const token = generateToken(
      {
        sub: "550e8400-e29b-41d4-a716-446655440000",
        tid: "660e8400-e29b-41d4-a716-446655440001",
        roles: ["member"],
        iss: "auth-service",
      },
      "wrong-secret-that-is-at-least-32-characters-long!!",
    );

    const res = await request(app).get("/test").set("Authorization", `Bearer ${token}`);
    expect(res.status).toBe(401);
    expect(res.body.error.message).toContain("Invalid token");
  });

  it("should reject tokens with wrong issuer", async () => {
    const app = buildApp();
    const token = generateToken({
      sub: "550e8400-e29b-41d4-a716-446655440000",
      tid: "660e8400-e29b-41d4-a716-446655440001",
      roles: ["member"],
      iss: "wrong-issuer",
    });

    const res = await request(app).get("/test").set("Authorization", `Bearer ${token}`);
    expect(res.status).toBe(401);
  });

  it("should reject tokens missing required claims (sub)", async () => {
    const app = buildApp();
    const token = generateToken({
      tid: "660e8400-e29b-41d4-a716-446655440001",
      roles: ["member"],
      iss: "auth-service",
    });

    const res = await request(app).get("/test").set("Authorization", `Bearer ${token}`);
    expect(res.status).toBe(401);
    expect(res.body.error.message).toContain("required claims");
  });

  it("should accept valid tokens and set identity headers", async () => {
    const app = buildApp();
    const token = generateToken({
      sub: "550e8400-e29b-41d4-a716-446655440000",
      tid: "660e8400-e29b-41d4-a716-446655440001",
      org: "660e8400-e29b-41d4-a716-446655440001",
      roles: ["org_admin", "member"],
      iss: "auth-service",
    });

    const res = await request(app).get("/test").set("Authorization", `Bearer ${token}`);
    expect(res.status).toBe(200);
    expect(res.body.tenantId).toBe("660e8400-e29b-41d4-a716-446655440001");
  });

  it("should overwrite spoofed identity headers", async () => {
    const app = buildApp();
    const token = generateToken({
      sub: "550e8400-e29b-41d4-a716-446655440000",
      tid: "660e8400-e29b-41d4-a716-446655440001",
      roles: ["member"],
      iss: "auth-service",
    });

    const res = await request(app)
      .get("/test")
      .set("Authorization", `Bearer ${token}`)
      .set("X-Tenant-ID", "spoofed-tenant-id")
      .set("X-User-ID", "spoofed-user-id");

    expect(res.status).toBe(200);
    // Should use JWT-derived value, not the spoofed one
    expect(res.body.tenantId).toBe("660e8400-e29b-41d4-a716-446655440001");
  });
});
