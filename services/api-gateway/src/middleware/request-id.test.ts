import express from "express";
import request from "supertest";
import { requestIdMiddleware } from "./request-id";

function buildApp(): express.Express {
  const app = express();
  app.use(requestIdMiddleware);
  app.get("/test", (req, res) => {
    res.json({ requestId: req.headers["x-request-id"] });
  });
  return app;
}

describe("requestIdMiddleware", () => {
  it("should generate a request ID when none is provided", async () => {
    const app = buildApp();
    const res = await request(app).get("/test");

    expect(res.status).toBe(200);
    expect(res.body.requestId).toBeDefined();
    expect(typeof res.body.requestId).toBe("string");
    expect(res.body.requestId.length).toBeGreaterThan(0);
    // Should also be on the response header
    expect(res.headers["x-request-id"]).toBe(res.body.requestId);
  });

  it("should preserve an existing request ID", async () => {
    const app = buildApp();
    const existingId = "my-custom-request-id-123";
    const res = await request(app).get("/test").set("X-Request-ID", existingId);

    expect(res.status).toBe(200);
    expect(res.body.requestId).toBe(existingId);
    expect(res.headers["x-request-id"]).toBe(existingId);
  });
});
