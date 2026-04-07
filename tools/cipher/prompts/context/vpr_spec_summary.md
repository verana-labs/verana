# VPR v4 Spec — Authoritative Reference for AI Reviews
Source: https://verana-labs.github.io/verifiable-trust-vpr-spec/

## Global Rules
- All precondition checks MUST be verified before AND during execution
- All APIs return valid JSON, snake_case attributes
- For delegable messages: [AUTHZ-CHECK] operator authorization MUST be verified
- Fee math: sdkmath.Int only, never float64
- Time: ctx.BlockTime() only, never time.Now()

## FEE RULES — CRITICAL (do not guess)

### Network Fees Only (NO trust deposit):
- Trust Registry: create, update, archive, GFV/GFD operations
- Credential Schema: create, update, archive
- Schema Authorization Policy: create, revoke
- Permission: root creation, OPEN mode self-creation, adjustments
- Digest storage

### Trust Deposit Required:
- Permission VP start/renew (applicant pays validation_fees + trust_deposit_rate%)
- Permission session (issuance/verification fees with full distribution)

### Fee Distribution Formula (for pay-per operations):
  total_fee = fee_for_operation
  to_trust_deposit = total_fee * trust_deposit_rate
  to_wallet_ua = total_fee * wallet_user_agent_reward_rate
  to_user_agent = total_fee * user_agent_reward_rate
  to_permission_holder = total_fee - to_trust_deposit - to_wallet_ua - to_user_agent

---

## TRUST REGISTRY MODULE (x/tr)

### [MOD-TR-MSG-1] Create New Trust Registry
Signers: corporation + operator (delegable)
Fields: did (mandatory), aka (optional), language (mandatory, BCP47), doc_url, doc_digest_sri
Fees: NETWORK FEES ONLY
State: creates TrustRegistry + GovernanceFrameworkVersion v1 + GovernanceFrameworkDocument
Preconditions: [AUTHZ-CHECK], valid DID, valid language tag

### [MOD-TR-MSG-2] Add Governance Framework Document
Signers: corporation + operator (delegable)
Fields: tr_id, language, url, digest_sri
Fees: NETWORK FEES ONLY
Note: does NOT increment governance version

### [MOD-TR-MSG-3] Increase Active Governance Framework Version
Signers: corporation + operator (delegable)
Fees: NETWORK FEES ONLY

### [MOD-TR-MSG-4] Update Trust Registry
Signers: corporation + operator (delegable)
Updatable: aka, language, active_version
Constraint: archived timestamp MUST remain null until archival
Fees: NETWORK FEES ONLY

### [MOD-TR-MSG-5] Archive Trust Registry
Signers: corporation + operator (delegable)
Effect: sets archived timestamp, prevents further modifications
Fees: NETWORK FEES ONLY

### [MOD-TR-MSG-6] Update TR Module Parameters
Signers: governance proposal ONLY (not delegable)

### Queries:
- [MOD-TR-QRY-1] Get Trust Registry: /tr/v1/get?tr_id=
- [MOD-TR-QRY-2] List Trust Registries: /tr/v1/list (pagination, filters: corporation, ecosystem, active)
- [MOD-TR-QRY-3] List Module Parameters: /tr/v1/params

---

## CREDENTIAL SCHEMA MODULE (x/cs)

### [MOD-CS-MSG-1] Create New Credential Schema
Signers: corporation + operator (delegable)
Fields: tr_id, json_schema, onboarding modes (issuer/verifier/holder), pricing config, validation periods
Fees: NETWORK FEES ONLY
Validation: json_schema size MUST NOT exceed credential_schema_schema_max_size

Onboarding modes:
- OPEN: anyone can self-create permission
- ECOSYSTEM_VALIDATION_PROCESS: corporation grants via VP
- GRANTOR_VALIDATION_PROCESS: grantor validates

Holder modes:
- ISSUER_VALIDATION_PROCESS: issuer validates holder
- PERMISSIONLESS: no validation needed

### [MOD-CS-MSG-2] Update Credential Schema
Updatable: pricing, validation periods, onboarding modes (with constraints)
Fees: NETWORK FEES ONLY

### [MOD-CS-MSG-3] Archive Credential Schema
Prevents new permissions; existing unaffected until explicit revocation
Fees: NETWORK FEES ONLY

### [MOD-CS-MSG-5] Create Schema Authorization Policy
Fields: schema_id, role (ISSUER|VERIFIER), url, digest_sri, effective_from/until
Fees: NETWORK FEES ONLY

### [MOD-CS-MSG-6] Increase Active SAP Version
### [MOD-CS-MSG-7] Revoke SAP

### Queries:
- [MOD-CS-QRY-1] List: /cs/v1/list
- [MOD-CS-QRY-2] Get: /cs/v1/get
- [MOD-CS-QRY-3] Render JSON Schema: /cs/v1/js/
- [MOD-CS-QRY-5] Get SAP: /cs/v1/sap/get
- [MOD-CS-QRY-6] List SAPs: /cs/v1/sap/list

---

## PERMISSION MODULE (x/perm)

### Permission States
- vp_state: PENDING → VALIDATED (success) or PENDING → TERMINATED (cancel)
- Active = effective_from <= now AND (effective_until null or > now) AND revoked/slashed null
- Future = effective_from > now, effective_until null or > effective_from, revoked/slashed null

### Role Hierarchy
ECOSYSTEM > ISSUER_GRANTOR/VERIFIER_GRANTOR > ISSUER/VERIFIER > HOLDER

### [MOD-PERM-MSG-1] Start Permission VP
Signers: corporation + operator (delegable)
Fields: schema_id, applicant_did, validator_perm_id, applicant_info_digest_sri
Preconditions: validator has appropriate active permission, no concurrent VP for same (schema, applicant, role)
Fees: TRUST DEPOSIT — escrows validation_fees + (validation_fees * trust_deposit_rate)
State: creates Permission with vp_state=PENDING, sets vp_exp

### [MOD-PERM-MSG-2] Renew Permission VP
Precondition: vp_state=PENDING, vp_exp < now
Re-escrows validation fees

### [MOD-PERM-MSG-3] Set Permission VP to Validated
Caller: validator_perm corporation
Effect: vp_state→VALIDATED, releases escrow (trust_deposit_rate% → validator deposit, rest → wallet)
Permission becomes ACTIVE (effective_from = now)

### [MOD-PERM-MSG-6] Cancel Permission VP
Precondition: vp_state=PENDING
Effect: refunds escrow, vp_state→TERMINATED

### [MOD-PERM-MSG-7] Create Root Permission
Use: initial ECOSYSTEM permission creation
Fees: NETWORK FEES ONLY (no trust deposit)

### [MOD-PERM-MSG-8] Adjust Permission
Updatable: validation_fees, issuance_fees, verification_fees, effective_until, discounts
Cannot adjust PENDING or TERMINATED
Fees: NETWORK FEES ONLY

### [MOD-PERM-MSG-9] Revoke Permission
Sets revoked timestamp, deposits freed (minus slashed)

### [MOD-PERM-MSG-10] Create or Update Permission Session
Uses [AUTHZ-CHECK-3] and [AUTHZ-CHECK-4] (VSOperatorAuthorization)
Pay-per-use: issuer/verifier pays issuance/verification fees with full distribution formula

### [MOD-PERM-MSG-12] Slash Permission Trust Deposit
### [MOD-PERM-MSG-13] Repay Slashed Trust Deposit
### [MOD-PERM-MSG-14] Self Create Permission (OPEN mode)
Fees: NETWORK FEES ONLY

### Queries:
- [MOD-PERM-QRY-1] List: /perm/v1/list
- [MOD-PERM-QRY-2] Get: /perm/v1/get
- [MOD-PERM-QRY-4] Find Beneficiaries: /perm/v1/beneficiaries
- [MOD-PERM-QRY-5] Get Session: /perm/v1/session/get

---

## TRUST DEPOSIT MODULE (x/td)

TrustDeposit per corporation: deposit, claimable, slashed_deposit, repaid_deposit, share
Yield: block fees distributed to deposit holders (trust_deposit_block_reward_share, default 20%)

### [MOD-TD-MSG-1] Adjust Trust Deposit (module-internal)
### [MOD-TD-MSG-2] Reclaim Yield (delegable)
### [MOD-TD-MSG-5] Slash (governance only)
### [MOD-TD-MSG-6] Repay Slashed (delegable)
### [MOD-TD-MSG-7] Burn Slashed (module-internal)

---

## DELEGATION MODULE (x/de)

### Authorization Types:
- OperatorAuthorization: corporation → operator for specific Msg types
- VSOperatorAuthorization: corporation → vs_operator for permission-specific ops
- FeeGrant: corporation → operator for network fee spending

### [AUTHZ-CHECK-1] Operator: exists, not expired, spend_limit sufficient
### [AUTHZ-CHECK-2] FeeGrant: exists, not expired, spend_limit sufficient
### [AUTHZ-CHECK-3] VS Operator: exists, permission in list, authz enabled
### [AUTHZ-CHECK-4] VS FeeGrant: feegrant enabled, limit sufficient

---

## EXCHANGE RATE MODULE (x/xr)
- [MOD-XR-MSG-1] Create (governance only)
- [MOD-XR-MSG-2] Update (authorized operator)
- [MOD-XR-MSG-3] Toggle state (governance only)
- Conversion: amount * (rate / 10^rate_scale)

## DIGESTS MODULE (x/di)
- [MOD-DI-MSG-1] Store Digest: network fees only
