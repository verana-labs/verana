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

const tests = [
  { name: "Create Trust Registry", script: "test:create-tr" },
  // Add more tests here as they are created
  // { name: "Create Credential Schema", script: "test:create-cs" },
  // { name: "Add DID", script: "test:add-did" },
  // etc.
  // Note: Query tests removed - focus on transaction signing validation
];

/**
 * Run a single test script
 */
async function runTest(testName: string, script: string): Promise<TestResult> {
  console.log("\n" + "=".repeat(60));
  console.log(`Running: ${testName}`);
  console.log("=".repeat(60));

  return new Promise((resolve) => {
    const child = spawn("npm", ["run", script], {
      stdio: "inherit",
      env: { ...process.env },
    });

    child.on("close", (code) => {
      if (code === 0) {
        console.log(`✅ ${testName} passed\n`);
        resolve({ name: testName, passed: true });
      } else {
        console.log(`❌ ${testName} failed with exit code ${code}\n`);
        resolve({
          name: testName,
          passed: false,
          error: `Exit code: ${code}`,
        });
      }
    });

    child.on("error", (error) => {
      console.log(`❌ ${testName} failed with error: ${error.message}\n`);
      resolve({
        name: testName,
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
    const result = await runTest(test.name, test.script);
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

