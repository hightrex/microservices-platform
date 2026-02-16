import CircuitBreaker from "opossum";
import { logger, ERROR_CODES } from "@microservices-platform/shared";
import type { GatewayConfig } from "../config";

interface CircuitBreakerRegistry {
  get(serviceName: string): CircuitBreaker | undefined;
  register(serviceName: string, baseUrl: string): CircuitBreaker;
}

/**
 * Create a circuit breaker registry that manages per-service breakers.
 * Each downstream service gets its own breaker instance.
 */
export function createCircuitBreakerRegistry(config: GatewayConfig): CircuitBreakerRegistry {
  const breakers = new Map<string, CircuitBreaker>();

  function healthCheckFn(baseUrl: string): () => Promise<boolean> {
    return async () => {
      const controller = new AbortController();
      const timer = setTimeout(() => controller.abort(), 3000);
      try {
        const response = await fetch(`${baseUrl}/health`, { signal: controller.signal });
        clearTimeout(timer);
        if (!response.ok) {
          throw new Error(`Health check failed: HTTP ${response.status}`);
        }
        return true;
      } catch (err) {
        clearTimeout(timer);
        throw err;
      }
    };
  }

  return {
    get(serviceName: string) {
      return breakers.get(serviceName);
    },

    register(serviceName: string, baseUrl: string) {
      const existing = breakers.get(serviceName);
      if (existing) return existing;

      const breaker = new CircuitBreaker(healthCheckFn(baseUrl), {
        timeout: config.cbTimeout,
        errorThresholdPercentage: config.cbErrorThreshold,
        resetTimeout: config.cbResetTimeout,
        name: serviceName,
        volumeThreshold: 5,
      });

      breaker.on("open", () => {
        logger.warn({ service: serviceName }, "Circuit breaker OPENED");
      });
      breaker.on("halfOpen", () => {
        logger.info({ service: serviceName }, "Circuit breaker half-open, testing...");
      });
      breaker.on("close", () => {
        logger.info({ service: serviceName }, "Circuit breaker CLOSED");
      });

      breakers.set(serviceName, breaker);
      return breaker;
    },
  };
}

/**
 * Build the standardized 503 response when a circuit breaker is open.
 */
export function circuitOpenResponse(serviceName: string) {
  return {
    success: false,
    error: {
      code: ERROR_CODES.SERVICE_UNAVAILABLE,
      message: `Service '${serviceName}' is temporarily unavailable`,
    },
  };
}

export type { CircuitBreakerRegistry };
