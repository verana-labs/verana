/**
 * Run CO/DE/EC/GF TypeScript Journey Suite
 *
 * Scoped hard-fail runner for the journeys that are stable and exercised in CI.
 * Stops immediately on the first failure and exits non-zero.
 *
 * Usage:
 *   npm run test:co-ec-gf
 */

import { spawn } from "child_process";

interface TestResult {
  name: string;
  passed: boolean;
  error?: string;
}

const tests = [
  { name: "CO: Create Corporation",                script: "test:co-create" },
  { name: "DE: Grant Operator Authorization",      script: "test:de-grant-auth" },
  { name: "DE: AUTHZ-CHECK-5 negative (unregistered)", script: "test:authz-check5" },
  { name: "EC: Create Ecosystem",                  script: "test:ec-create" },
  { name: "GF: Add Governance Framework Document", script: "test:gf-add-doc" },
  { name: "GF: Increase Active GF Version",        script: "test:gf-increase-version" },
  { name: "EC: Update Ecosystem",                  script: "test:ec-update" },
  { name: "EC: Archive Ecosystem",                 script: "test:ec-archive" },
];

async function runTest(name: string, script: string): Promise<TestResult> {
  console.log("\n" + "=".repeat(60));
  console.log(`Running: ${name}`);
  console.log("=".repeat(60));

  return new Promise((resolve) => {
    const child = spawn("npm", ["run", script], {
      stdio: "inherit",
      env: { ...process.env },
    });

    child.on("close", (code) => {
      if (code === 0) {
        console.log(`\n✅ ${name} passed`);
        resolve({ name, passed: true });
      } else {
        console.log(`\n❌ ${name} failed with exit code ${code}`);
        resolve({ name, passed: false, error: `Exit code: ${code}` });
      }
    });

    child.on("error", (error) => {
      console.log(`\n❌ ${name} failed: ${error.message}`);
      resolve({ name, passed: false, error: error.message });
    });
  });
}

async function main() {
  console.log("=".repeat(60));
  console.log("Verana CO/EC/GF TypeScript Journey Suite");
  console.log("=".repeat(60));
  console.log(`Running ${tests.length} journeys sequentially (hard-fail on first error)\n`);

  for (const test of tests) {
    const result = await runTest(test.name, test.script);
    if (!result.passed) {
      console.log("\n" + "=".repeat(60));
      console.log(`❌ Suite failed at: ${test.name}`);
      if (result.error) console.log(`   Error: ${result.error}`);
      console.log("=".repeat(60));
      process.exit(1);
    }
  }

  console.log("\n" + "=".repeat(60));
  console.log("✅ All CO/EC/GF journey tests passed!");
  console.log("=".repeat(60));
  process.exit(0);
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
