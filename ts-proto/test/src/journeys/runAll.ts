/**
 * Run All TypeScript Client Tests
 *
 * This script runs all TypeScript client journey tests sequentially.
 * It validates that all transaction types can be signed and broadcast
 * correctly using the TypeScript protobuf types.
 *
 * Usage:
 *   npm run test:all
 *
 * Or with environment variables:
 *   export VERANA_RPC_ENDPOINT="http://localhost:26657"
 *   export VERANA_LCD_ENDPOINT="http://localhost:1317"
 *   npm run test:all
 */

import { spawn } from "child_process";

interface TestResult {
  name: string;
  passed: boolean;
  error?: string;
}

interface TestConfig {
  name: string;
  script: string;
  isGoJourney?: boolean;
}

const tests: TestConfig[] = [
  // Delegation Engine (DE) module: Grant operator authorization
  { name: "DE: Grant Operator Authorization", script: "test:de-grant-auth" },
  // Trust Registry (TR) module: All 5 operations (operator-signed)
  { name: "TR: Create Trust Registry", script: "test:tr-create" },
  { name: "TR: Add GF Document", script: "test:tr-add-gfd" },
  { name: "TR: Increase Active GF Version", script: "test:tr-increase-gf-version" },
  { name: "TR: Update Trust Registry", script: "test:tr-update" },
  { name: "TR: Archive Trust Registry", script: "test:tr-archive" },
];

/**
 * Run a single test script
 */
async function runTest(test: TestConfig): Promise<TestResult> {
  console.log("\n" + "=".repeat(60));
  console.log(`Running: ${test.name}`);
  console.log("=".repeat(60));

  return new Promise((resolve) => {
    let child;

    if (test.isGoJourney) {
      // Run Go test harness journey (journey 20 for TD yield proposal setup)
      // Assumes the Go binary is available in the testharness directory
      const goCommand = process.env.GO_TEST_HARNESS_PATH || "go";
      const journeyId = test.script === "test:setup-td-proposal" ? "20" : "";
      // Path from ts-proto/test to testharness (go up 2 levels: test -> ts-proto -> verana)
      const testharnessPath = process.env.TESTHARNESS_DIR || "../../testharness";
      child = spawn(goCommand, ["run", "cmd/main.go", journeyId], {
        stdio: "inherit",
        env: { ...process.env },
        cwd: testharnessPath,
      });
    } else {
      // Run npm script (TypeScript journey)
      child = spawn("npm", ["run", test.script], {
        stdio: "inherit",
        env: { ...process.env, RUNNING_ALL_TESTS: "true" },
      });
    }

    child.on("close", (code) => {
      if (code === 0) {
        console.log(`✅ ${test.name} passed\n`);
        resolve({ name: test.name, passed: true });
      } else {
        console.log(`❌ ${test.name} failed with exit code ${code}\n`);
        resolve({
          name: test.name,
          passed: false,
          error: `Exit code: ${code}`,
        });
      }
    });

    child.on("error", (error) => {
      console.log(`❌ ${test.name} failed with error: ${error.message}\n`);
      resolve({
        name: test.name,
        passed: false,
        error: error.message,
      });
    });
  });
}

/**
 * Main function to run all tests
 */
async function main() {
  console.log("=".repeat(60));
  console.log("Verana TypeScript Client Test Suite");
  console.log("=".repeat(60));
  console.log(`Running ${tests.length} test(s)...\n`);

  const results: TestResult[] = [];

  // Run tests sequentially
  for (const test of tests) {
    const result = await runTest(test);
    results.push(result);

    // If a test fails, you can choose to continue or stop
    // For now, we continue to see all results
    if (!result.passed) {
      console.log(`⚠️  Warning: ${test.name} failed, but continuing...\n`);
    }
  }

  // Print summary
  console.log("\n" + "=".repeat(60));
  console.log("Test Summary");
  console.log("=".repeat(60));

  const passed = results.filter((r) => r.passed).length;
  const failed = results.filter((r) => !r.passed).length;

  console.log(`Total tests: ${results.length}`);
  console.log(`✅ Passed: ${passed}`);
  console.log(`❌ Failed: ${failed}`);

  if (failed > 0) {
    console.log("\nFailed tests:");
    results
      .filter((r) => !r.passed)
      .forEach((r) => {
        console.log(`  - ${r.name}: ${r.error || "Unknown error"}`);
      });
  }

  console.log("=".repeat(60));

  // Exit with error code if any tests failed
  if (failed > 0) {
    console.log("\n❌ Some tests failed. Please review the output above.");
    process.exit(1);
  } else {
    console.log("\n✅ All tests passed!");
    process.exit(0);
  }
}

main().catch((error) => {
  console.error("Fatal error running tests:", error);
  process.exit(1);
});

