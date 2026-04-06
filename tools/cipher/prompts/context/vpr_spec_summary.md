# Verana VPR Spec — Rules for AI Implementation
Source: https://verana-labs.github.io/verifiable-trust-vpr-spec/

## Core Objects

### TrustRegistry
Fields: did, aka (optional), language (rfc1766), governance framework URL + digest_sri
States: ACTIVE / SUSPENDED / REVOKED
Must have >= 1 CredentialSchema to be usable for trust resolution.
SUSPENDED: schemas/permissions exist but unusable. REVOKED: children orphaned, not deleted.

### CredentialSchema
Owner: TrustRegistry controller
Fields: json_schema (JSON Schema 2020-12), issuance PermissionManagementMode, verification PermissionManagementMode
Modes: OPEN (anyone) | ECOSYSTEM (controller grants) | GRANTOR (VERIFIER_GRANTOR validates)

### Permission
Types: ISSUER, VERIFIER, VERIFIER_GRANTOR
Fields: schema_id, grantee, created, created_by, extended, extended_by
EXPIRED = not renewed = cannot issue/verify. Must be renewed by paying trust fees.

### DID Directory
Any participant registers DID by paying trust_deposit.
Fields: did, service_endpoint, created, owner_account

## Fee Distribution — EXACT MATH, never float64, use sdkmath.Int
  total_fee          = fee_for_schema
  to_trust_deposit   = total_fee * trust_deposit_rate
  to_wallet_ua       = total_fee * wallet_user_agent_reward_rate
  to_user_agent      = total_fee * user_agent_reward_rate
  to_permission_holder = total_fee - to_trust_deposit - to_wallet_ua - to_user_agent

## Essential Credential Schemas (all 4 required per trust registry)
1. ServiceCredential — the Verifiable Service
2. OrganizationCredential — org running the VS
3. PersonaCredential — VUA user
4. UserAgentCredential — compliant VUA

## TRQP Query API
GET /tr/v1/get?did=...
GET /tr/v1/list?modified_after=...&active_gf_only=...
GET /cs/v1/get?id=...
GET /permission/v1/get?schema_id=&grantee=...
Results ordered by modified asc unless specified.

## Slashing
Partial: reduce trust_deposit by amount.
Full: trust_deposit → 0, effectively banned.
Slash reasons recorded on-chain.
