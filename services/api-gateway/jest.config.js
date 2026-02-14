/** @type {import('jest').Config} */
module.exports = {
  preset: "ts-jest",
  testEnvironment: "node",
  roots: ["<rootDir>/src"],
  testMatch: ["**/*.test.ts"],
  collectCoverageFrom: ["src/**/*.ts", "!src/main.ts"],
  coverageDirectory: "coverage",
  moduleFileExtensions: ["ts", "js", "json"],
  moduleNameMapper: {
    "^@microservices-platform/shared$": "<rootDir>/../../libs/typescript/src/index",
  },
};
