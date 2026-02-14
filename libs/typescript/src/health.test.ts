import { checkDependency, aggregateHealth } from "./health";
import type { HealthCheckTarget } from "./health";

// Mock global fetch
const originalFetch = global.fetch;

afterEach(() => {
  global.fetch = originalFetch;
});

describe("checkDependency", () => {
  it("should return healthy when response is OK", async () => {
    global.fetch = jest.fn().mockResolvedValue({ ok: true, status: 200 });

    const target: HealthCheckTarget = { name: "auth-service", url: "http://auth:8080/health" };
    const result = await checkDependency(target);

    expect(result.name).toBe("auth-service");
    expect(result.status).toBe("healthy");
    expect(result.latencyMs).toBeDefined();
  });

  it("should return degraded when response is not OK", async () => {
    global.fetch = jest.fn().mockResolvedValue({ ok: false, status: 503 });

    const target: HealthCheckTarget = { name: "org-service", url: "http://org:8081/health" };
    const result = await checkDependency(target);

    expect(result.status).toBe("degraded");
    expect(result.message).toBe("HTTP 503");
  });

  it("should return unhealthy on network error", async () => {
    global.fetch = jest.fn().mockRejectedValue(new Error("ECONNREFUSED"));

    const target: HealthCheckTarget = { name: "down-service", url: "http://down:9999/health", timeoutMs: 100 };
    const result = await checkDependency(target);

    expect(result.status).toBe("unhealthy");
    expect(result.message).toContain("ECONNREFUSED");
  });
});

describe("aggregateHealth", () => {
  it("should return healthy when all dependencies are healthy", async () => {
    global.fetch = jest.fn().mockResolvedValue({ ok: true, status: 200 });

    const targets: HealthCheckTarget[] = [
      { name: "auth-service", url: "http://auth:8080/health" },
      { name: "org-service", url: "http://org:8081/health" },
    ];
    const result = await aggregateHealth(targets);

    expect(result.status).toBe("healthy");
    expect(result.dependencies).toHaveLength(2);
  });

  it("should return unhealthy if any dependency is unhealthy", async () => {
    global.fetch = jest
      .fn()
      .mockResolvedValueOnce({ ok: true, status: 200 })
      .mockRejectedValueOnce(new Error("ECONNREFUSED"));

    const targets: HealthCheckTarget[] = [
      { name: "auth-service", url: "http://auth:8080/health" },
      { name: "org-service", url: "http://org:8081/health" },
    ];
    const result = await aggregateHealth(targets);

    expect(result.status).toBe("unhealthy");
  });

  it("should return degraded if any dependency is degraded but none unhealthy", async () => {
    global.fetch = jest
      .fn()
      .mockResolvedValueOnce({ ok: true, status: 200 })
      .mockResolvedValueOnce({ ok: false, status: 503 });

    const targets: HealthCheckTarget[] = [
      { name: "auth-service", url: "http://auth:8080/health" },
      { name: "org-service", url: "http://org:8081/health" },
    ];
    const result = await aggregateHealth(targets);

    expect(result.status).toBe("degraded");
  });

  it("should handle empty targets", async () => {
    const result = await aggregateHealth([]);
    expect(result.status).toBe("healthy");
    expect(result.dependencies).toHaveLength(0);
  });
});
