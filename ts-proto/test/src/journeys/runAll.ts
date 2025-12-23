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
  // Trust Registry (tr) module
  { name: "Create Trust Registry", script: "test:create-tr" },
  { name: "Update Trust Registry", script: "test:update-tr" },
  { name: "Archive Trust Registry", script: "test:archive-tr" },
  { name: "Add Governance Framework Document", script: "test:add-gf-doc" },
  { name: "Increase Active Governance Framework Version", script: "test:increase-gf-version" },
  // DID Directory (dd) module
  { name: "Add DID", script: "test:add-did" },
  { name: "Renew DID", script: "test:renew-did" },
  { name: "Remove DID", script: "test:remove-did" },
  { name: "Touch DID", script: "test:touch-did" },
  // Credential Schema (cs) module
  { name: "Create Credential Schema", script: "test:create-cs" },
  { name: "Update Credential Schema", script: "test:update-cs" },
  { name: "Archive Credential Schema", script: "test:archive-cs" },
  // Permission (perm) module
  { name: "Create Root Permission", script: "test:create-root-perm" },
  { name: "Create Permission", script: "test:create-perm" },
  { name: "Extend Permission", script: "test:extend-perm" },
  { name: "Revoke Permission", script: "test:revoke-perm" },
  { name: "Start Permission VP", script: "test:start-perm-vp" },
  { name: "Renew Permission VP", script: "test:renew-perm-vp" },
  { name: "Set Permission VP To Validated", script: "test:set-perm-vp-validated" },
  { name: "Cancel Permission VP Last Request", script: "test:cancel-perm-vp" },
  { name: "Create Or Update Permission Session", script: "test:create-perm-session" },
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

