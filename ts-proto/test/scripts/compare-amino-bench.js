// Compare TS vs Go amino bench outputs.
// Run: node scripts/compare-amino-bench.js

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
  const tsServerJson = read("amino-sign-bench-ts-server.json");
  const tsClientJson = read("amino-sign-bench-ts-client.json");
  const goServerJson = read("amino-sign-bench-go-server.json");
  const goClientJson = read("amino-sign-bench-go-client.json");

  const tsServerHex = read("amino-sign-bench-ts-server.hex");
  const tsClientHex = read("amino-sign-bench-ts-client.hex");
  const goServerHex = read("amino-sign-bench-go-server.hex");
  const goClientHex = read("amino-sign-bench-go-client.hex");

  compare("server json", tsServerJson, goServerJson);
  compare("client json", tsClientJson, goClientJson);

  console.log(`server hex: ${tsServerHex === goServerHex}`);
  console.log(`client hex: ${tsClientHex === goClientHex}`);
}

try {
  main();
} catch (err) {
  console.error("compare error:", err.message || err);
  process.exit(1);
}
