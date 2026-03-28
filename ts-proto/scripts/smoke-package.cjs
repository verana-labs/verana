const assert = require("node:assert/strict");

const root = require("../dist");
const signing = require("../dist/signing");

assert.equal(typeof root.createVeranaRegistry, "function");
assert.equal(typeof root.createVeranaAminoTypes, "function");
assert.equal(typeof root.pickOptionalUInt32, "function");
assert.equal(typeof signing.createVeranaRegistry, "function");
assert.equal(typeof signing.createVeranaAminoTypes, "function");

const requiredRegistry = [
  "MsgCreateTrustRegistry",
  "MsgCreateCredentialSchema",
  "MsgCreatePermission",
  "MsgReclaimTrustDepositYield",
  "MsgGrantOperatorAuthorization",
  "MsgStoreDigest",
  "MsgCreateExchangeRate",
];

const registry = signing.createVeranaRegistry();
for (const key of requiredRegistry) {
  assert.ok(
    registry.lookupType(signing.veranaTypeUrls[key]),
    `missing registry type for ${key}`,
  );
}

const amino = signing.createVeranaAminoTypes();
for (const key of requiredRegistry) {
  assert.ok(
    amino.register[signing.veranaTypeUrls[key]],
    `missing amino converter for ${key}`,
  );
}

console.log("ts-proto package smoke check passed");
