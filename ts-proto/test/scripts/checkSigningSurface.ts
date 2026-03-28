import assert from "node:assert/strict";
import { createVeranaAminoTypes, createVeranaRegistry, veranaTypeUrls } from "../../src/signing";
import { MsgGrantOperatorAuthorization } from "../../src/codec/verana/de/v1/tx";
import { MsgStoreDigest } from "../../src/codec/verana/di/v1/tx";
import { MsgCreatePermission, MsgStartPermissionVP } from "../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../src/codec/verana/perm/v1/types";
import {
  MsgCreateExchangeRate,
  MsgToggleExchangeRateState,
  MsgUpdateExchangeRate,
} from "../../src/codec/verana/xr/v1/tx";

const registry = createVeranaRegistry();
const amino = createVeranaAminoTypes() as any;

const requiredMappings = [
  "MsgCreateTrustRegistry",
  "MsgCreateCredentialSchema",
  "MsgCreatePermission",
  "MsgReclaimTrustDepositYield",
  "MsgGrantOperatorAuthorization",
  "MsgStoreDigest",
  "MsgCreateExchangeRate",
  "MsgUpdateExchangeRate",
  "MsgToggleExchangeRateState",
] as const;

for (const key of requiredMappings) {
  assert.ok(registry.lookupType(veranaTypeUrls[key]), `missing registry mapping for ${key}`);
  assert.ok(amino.register[veranaTypeUrls[key]], `missing amino mapping for ${key}`);
}

const deConverter = amino.register[veranaTypeUrls.MsgGrantOperatorAuthorization];
const deMsg = MsgGrantOperatorAuthorization.fromPartial({
  authority: "verana1authority0000000000000000000000000000000",
  operator: "verana1operator0000000000000000000000000000000",
  grantee: "verana1grantee00000000000000000000000000000000",
  msgTypes: [veranaTypeUrls.MsgCreateTrustRegistry, veranaTypeUrls.MsgCreatePermission],
  expiration: new Date("2026-04-01T12:00:00.123Z"),
  authzSpendLimit: [{ denom: "uvna", amount: "42" }],
  authzSpendLimitPeriod: { seconds: 3600, nanos: 5 },
  withFeegrant: true,
  feegrantSpendLimit: [{ denom: "uvna", amount: "7" }],
  feegrantSpendLimitPeriod: { seconds: 7200, nanos: 0 },
});
const deRoundTrip = deConverter.fromAmino(deConverter.toAmino(deMsg));
assert.equal(deRoundTrip.msgTypes.length, 2);
assert.equal(deRoundTrip.authzSpendLimitPeriod?.seconds, 3600);
assert.equal(deRoundTrip.authzSpendLimitPeriod?.nanos, 5);
assert.equal(deRoundTrip.withFeegrant, true);
assert.equal(deRoundTrip.feegrantSpendLimitPeriod?.seconds, 7200);

const diConverter = amino.register[veranaTypeUrls.MsgStoreDigest];
const diMsg = MsgStoreDigest.fromPartial({
  authority: "verana1corp0000000000000000000000000000000000",
  operator: "verana1operator0000000000000000000000000000000",
  digest: "sha256-abc123",
});
const diRoundTrip = diConverter.fromAmino(diConverter.toAmino(diMsg));
assert.equal(diRoundTrip.digest, "sha256-abc123");

const xrCreateConverter = amino.register[veranaTypeUrls.MsgCreateExchangeRate];
const xrCreateMsg = MsgCreateExchangeRate.fromPartial({
  authority: "verana1authority0000000000000000000000000000000",
  baseAssetType: 1,
  baseAsset: "EUR",
  quoteAssetType: 1,
  quoteAsset: "USD",
  rate: "1.0705",
  rateScale: 4,
  validityDuration: { seconds: 1800, nanos: 9 },
});
const xrCreateRoundTrip = xrCreateConverter.fromAmino(xrCreateConverter.toAmino(xrCreateMsg));
assert.equal(xrCreateRoundTrip.baseAsset, "EUR");
assert.equal(xrCreateRoundTrip.validityDuration?.seconds, 1800);
assert.equal(xrCreateRoundTrip.validityDuration?.nanos, 9);

const xrUpdateConverter = amino.register[veranaTypeUrls.MsgUpdateExchangeRate];
const xrUpdateMsg = MsgUpdateExchangeRate.fromPartial({
  authority: "verana1authority0000000000000000000000000000000",
  operator: "verana1operator0000000000000000000000000000000",
  id: 12,
  rate: "1.0800",
});
const xrUpdateRoundTrip = xrUpdateConverter.fromAmino(xrUpdateConverter.toAmino(xrUpdateMsg));
assert.equal(xrUpdateRoundTrip.id, 12);
assert.equal(xrUpdateRoundTrip.rate, "1.0800");

const xrToggleConverter = amino.register[veranaTypeUrls.MsgToggleExchangeRateState];
const xrToggleMsg = MsgToggleExchangeRateState.fromPartial({
  authority: "verana1authority0000000000000000000000000000000",
  id: 12,
  state: false,
});
const xrToggleRoundTrip = xrToggleConverter.fromAmino(xrToggleConverter.toAmino(xrToggleMsg));
assert.equal(xrToggleRoundTrip.id, 12);
assert.equal(xrToggleRoundTrip.state, false);

const permConverter = amino.register[veranaTypeUrls.MsgCreatePermission];
const permMsg = MsgCreatePermission.fromPartial({
  authority: "verana1authority0000000000000000000000000000000",
  operator: "verana1operator0000000000000000000000000000000",
  type: PermissionType.VERIFIER,
  validatorPermId: 17,
  did: "did:verana:test:perm",
  verificationFees: 0,
  validationFees: 0,
  vsOperator: "verana1vsoperator0000000000000000000000000000",
  vsOperatorAuthzEnabled: true,
  vsOperatorAuthzSpendLimit: [{ denom: "uvna", amount: "15" }],
  vsOperatorAuthzWithFeegrant: true,
  vsOperatorAuthzFeeSpendLimit: [{ denom: "uvna", amount: "5" }],
  vsOperatorAuthzSpendPeriod: { seconds: 5400, nanos: 11 },
});
const permRoundTrip = permConverter.fromAmino(permConverter.toAmino(permMsg));
assert.equal(permRoundTrip.vsOperatorAuthzSpendPeriod?.seconds, 5400);
assert.equal(permRoundTrip.vsOperatorAuthzSpendPeriod?.nanos, 11);

const startVpConverter = amino.register[veranaTypeUrls.MsgStartPermissionVP];
const startVpMsg = MsgStartPermissionVP.fromPartial({
  authority: "verana1authority0000000000000000000000000000000",
  operator: "verana1operator0000000000000000000000000000000",
  type: PermissionType.VALIDATOR,
  validatorPermId: 21,
  did: "did:verana:test:vp",
  validationFees: { value: 100 },
  issuanceFees: { value: 200 },
  verificationFees: { value: 300 },
  vsOperatorAuthzSpendPeriod: { seconds: 900, nanos: 2 },
});
const startVpRoundTrip = startVpConverter.fromAmino(startVpConverter.toAmino(startVpMsg));
assert.equal(startVpRoundTrip.validationFees?.value, 100);
assert.equal(startVpRoundTrip.issuanceFees?.value, 200);
assert.equal(startVpRoundTrip.verificationFees?.value, 300);
assert.equal(startVpRoundTrip.vsOperatorAuthzSpendPeriod?.seconds, 900);
assert.equal(startVpRoundTrip.vsOperatorAuthzSpendPeriod?.nanos, 2);

console.log("signing surface check passed");
