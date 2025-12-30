// Compare TS vs Go credential schema Amino bench outputs.
// Run: node ts-proto/test/scripts/compare-amino-bench-cs.js

const { readFileSync } = require("node:fs");
const { join } = require("node:path");

const outDir = join(__dirname, "..", "out");

function read(path) {
  return readFileSync(join(outDir, path), "utf8").trim();
}

function normalizeJson(text) {
  return JSON.stringify(sortKeysDeep(JSON.parse(text)));
}

function sortKeysDeep(value) {
  if (Array.isArray(value)) {
    return value.map(sortKeysDeep);
  }
  if (value && typeof value === "object") {
    return Object.keys(value)
      .sort()
      .reduce((acc, key) => {
        acc[key] = sortKeysDeep(value[key]);
        return acc;
      }, {});
  }
  return value;
}

function compare(label, a, b) {
  const rawEqual = a === b;
  let normalizedEqual = false;
  try {
    normalizedEqual = normalizeJson(a) === normalizeJson(b);
  } catch {
    normalizedEqual = false;
  }
  console.log(`${label}: raw=${rawEqual} normalized=${normalizedEqual}`);
}

function main() {
  const tsJson = read("amino-sign-bench-cs-ts.json");
  const goJson = read("amino-sign-bench-cs-go.json");
  const tsHex = read("amino-sign-bench-cs-ts.hex");
  const goHex = read("amino-sign-bench-cs-go.hex");

  compare("json", tsJson, goJson);
  console.log(`hex: ${tsHex === goHex}`);
}

try {
  main();
} catch (err) {
  console.error("compare error:", err.message || err);
  process.exit(1);
}
