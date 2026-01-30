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
  // Create Root Permission - now split into 2 steps to avoid race conditions
  { name: "Setup Create Root Permission Prerequisites", script: "test:setup-create-root-perm-prereqs" },
  { name: "Create Root Permission", script: "test:create-root-perm" },
  // Create Permission - now split into 2 steps to avoid race conditions
  { name: "Setup Create Permission Prerequisites", script: "test:setup-create-perm-prereqs" },
  { name: "Create Permission", script: "test:create-perm" },
  { name: "Extend Permission", script: "test:extend-perm" },
  { name: "Revoke Permission", script: "test:revoke-perm" },
  { name: "Start Permission VP", script: "test:start-perm-vp" },
  { name: "Renew Permission VP", script: "test:renew-perm-vp" },
  { name: "Set Permission VP To Validated", script: "test:set-perm-vp-validated" },
  { name: "Cancel Permission VP Last Request", script: "test:cancel-perm-vp" },
  // Permission Session - now split into 3 steps to avoid race conditions
  { name: "Setup Permission Session Prerequisites", script: "test:setup-perm-session-prereqs" },
  { name: "Setup Permission Session Permissions", script: "test:setup-perm-session-perms" },
  { name: "Create Or Update Permission Session", script: "test:create-perm-session" },
  // Trust Deposit (td) module - Setup funding proposal first (Journey 20)
  { name: "Setup TD Yield Funding Proposal (Journey 20)", script: "test:setup-td-proposal", isGoJourney: true },
  { name: "Reclaim Trust Deposit Yield", script: "test:reclaim-td-yield" },
  { name: "Reclaim Trust Deposit", script: "test:reclaim-td" },
  // Note: Query tests removed - focus on transaction signing validation
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

