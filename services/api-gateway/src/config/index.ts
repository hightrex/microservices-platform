import { z } from "zod";

/**
 * Gateway configuration schema — validated at startup via zod.
 * All values come from environment variables; no direct `process.env` access elsewhere.
 */
const configSchema = z.object({
  port: z.coerce.number().int().positive().default(3000),
  environment: z.enum(["development", "production", "test"]).default("development"),
  logLevel: z.enum(["fatal", "error", "warn", "info", "debug", "trace"]).default("info"),

  // Upstream service URLs (internal compose DNS names)
  authBaseUrl: z.string().url(),
  orgBaseUrl: z.string().url(),
  notificationBaseUrl: z.string().url().optional(),
  auditBaseUrl: z.string().url().optional(),
  billingBaseUrl: z.string().url().optional(),
  fileBaseUrl: z.string().url().optional(),

  // Redis
  redisUrl: z.string().default("redis://localhost:6379"),
  redisPassword: z.string().default(""),
  redisDb: z.coerce.number().int().min(0).default(2),

  // CORS — explicit allowlist, never "*"
  corsOrigins: z
    .string()
    .default("http://localhost:3001")
    .transform((s) => s.split(",").map((o) => o.trim()).filter(Boolean)),

  // JWT validation
  jwtSecret: z.string().min(32),
  jwtIssuer: z.string().default("auth-service"),

  // Rate limiting defaults
  rateLimitWindowMs: z.coerce.number().int().positive().default(60_000),
  rateLimitMaxRequests: z.coerce.number().int().positive().default(100),

  // Request body limit
  bodyLimitBytes: z.coerce.number().int().positive().default(1_048_576), // 1 MB

  // Circuit breaker
  cbTimeout: z.coerce.number().int().positive().default(10_000),
  cbErrorThreshold: z.coerce.number().int().min(1).max(100).default(50),
  cbResetTimeout: z.coerce.number().int().positive().default(30_000),

  // Module cache TTL in seconds
  moduleCacheTtlSeconds: z.coerce.number().int().positive().default(300),
});

export type GatewayConfig = z.infer<typeof configSchema>;

/** Map environment variables to config keys. */
function envToConfig(): Record<string, string | undefined> {
  return {
    port: process.env["PORT"] ?? process.env["GATEWAY_PORT"],
    environment: process.env["NODE_ENV"] ?? process.env["ENVIRONMENT"],
    logLevel: process.env["LOG_LEVEL"],
    authBaseUrl: process.env["AUTH_BASE_URL"],
    orgBaseUrl: process.env["ORG_BASE_URL"],
    notificationBaseUrl: process.env["NOTIFICATION_BASE_URL"],
    auditBaseUrl: process.env["AUDIT_BASE_URL"],
    billingBaseUrl: process.env["BILLING_BASE_URL"],
    fileBaseUrl: process.env["FILE_BASE_URL"],
    redisUrl: process.env["REDIS_URL"],
    redisPassword: process.env["REDIS_PASSWORD"],
    redisDb: process.env["REDIS_DB"],
    corsOrigins: process.env["CORS_ORIGINS"],
    jwtSecret: process.env["JWT_SECRET"],
    jwtIssuer: process.env["JWT_ISSUER"],
    rateLimitWindowMs: process.env["RATE_LIMIT_WINDOW_MS"],
    rateLimitMaxRequests: process.env["RATE_LIMIT_MAX_REQUESTS"],
    bodyLimitBytes: process.env["BODY_LIMIT_BYTES"],
    cbTimeout: process.env["CB_TIMEOUT"],
    cbErrorThreshold: process.env["CB_ERROR_THRESHOLD"],
    cbResetTimeout: process.env["CB_RESET_TIMEOUT"],
    moduleCacheTtlSeconds: process.env["MODULE_CACHE_TTL_SECONDS"],
  };
}

/** Load and validate config from environment. Throws on invalid config. */
export function loadConfig(): GatewayConfig {
  const raw = envToConfig();
  const result = configSchema.safeParse(raw);
  if (!result.success) {
    const formatted = result.error.issues
      .map((i) => `  ${i.path.join(".")}: ${i.message}`)
      .join("\n");
    throw new Error(`Invalid gateway configuration:\n${formatted}`);
  }
  return result.data;
}
