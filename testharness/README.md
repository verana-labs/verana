# Verana-test-harness

## Overview
This repository contains a test harness for the Verifiable Public Registry (VPR) specification implementation. It allows for automated testing of various customer journeys that represent typical interactions with a VPR blockchain.

## Prerequisites
- Go 1.18 or later
- A running Verana blockchain node
- Account with sufficient funds for testing

## Installation
```bash
git clone https://github.com/your-org/verana-test-harness.git
cd verana-test-harness
go mod tidy
```

## Configuration
Set up environment variables for the test harness:

```bash
export ADDRESS_PREFIX="verana"
export HOME_DIR="~/.verana"
export NODE_RPC="http://localhost:26657"
export GAS="200000"
export FEES="750000uvna"
```

devnet
```bash
export ADDRESS_PREFIX="verana"
export HOME_DIR="~/.verana"
export NODE_RPC="http://node1.devnet.verana.network:26657"
export GAS="300000"
export FEES="750000uvna"
```

testnet

```bash
export ADDRESS_PREFIX="verana"
export HOME_DIR="~/.verana"
export NODE_RPC="http://node1.testnet.verana.network:26657"
export GAS="300000"
export FEES="750000uvna"
```

## Usage
To run a specific journey, use:

```bash
go run cmd/main.go [journey_id]
```

Example:
```bash
go run cmd/main.go 1  # Run Trust Registry Controller Journey
```

To run all the journeys, use:
```
scripts/run_all.sh
```

## Available Journeys

### Journey 1: Trust Registry Controller Journey
1. `Trust_Registry_Controller` account is created with sufficient funds
2. `Trust_Registry_Controller` creates a trust registry:
   - Transaction: `Create New Trust Registry` (MOD-TR-MSG-1)
   - Parameters: DID, governance framework document URL, language
3. `Trust_Registry_Controller` creates a credential schema:
   - Transaction: `Create a Credential Schema` (MOD-CS-MSG-1)
   - Parameters: trust registry ID, JSON schema, issuer/verifier permission management modes
4. `Trust_Registry_Controller` creates root permission:
   - Transaction: `Create Root Permission` (MOD-PERM-MSG-7)
   - Parameters: schema ID, validation service DID, validation/issuance/verification fees
5. `Trust_Registry_Controller` adds DID to directory:
   - Transaction: `Add a DID` (MOD-DD-MSG-1)
   - Parameters: DID, registration period

### Journey 2: Issuer Grantor Validation Journey
1. `Issuer_Grantor_Applicant` account is created with sufficient funds
2. `Trust_Registry_Controller` already exists with trust registry, credential schema, and root permission
3. `Issuer_Grantor_Applicant` starts validation process:
   - Transaction: `Start Permission VP` (MOD-PERM-MSG-1)
   - Parameters: type=ISSUER_GRANTOR, Trust Registry's validator permission ID, country
4. `Issuer_Grantor_Applicant` connects to `Trust_Registry_Controller`'s validation service
5. `Trust_Registry_Controller` validates the applicant:
   - Transaction: `Set Permission VP to Validated` (MOD-PERM-MSG-3)
   - Parameters: permission ID, effective until, validation/issuance/verification fees
6. `Issuer_Grantor_Applicant` adds their DID to directory:
   - Transaction: `Add a DID` (MOD-DD-MSG-1)
   - Parameters: DID, registration period

### Journey 3: Issuer Validation Journey (via Issuer Grantor)
1. `Issuer_Applicant` account is created with sufficient funds
2. Trust Registry and Credential Schema already exist (created by `Trust_Registry_Controller`)
3. `Issuer_Grantor_Applicant` already exists with ISSUER_GRANTOR validation permission
4. `Issuer_Applicant` starts validation process:
   - Transaction: `Start Permission VP` (MOD-PERM-MSG-1)
   - Parameters: type=ISSUER, Issuer Grantor's validator permission ID, country
5. `Issuer_Applicant` connects to `Issuer_Grantor_Applicant`'s validation service
6. `Issuer_Grantor_Applicant` validates the applicant:
   - Transaction: `Set Permission VP to Validated` (MOD-PERM-MSG-3)
   - Parameters: permission ID, effective until, validation/issuance/verification fees
7. `Issuer_Applicant` adds their DID to directory:
   - Transaction: `Add a DID` (MOD-DD-MSG-1)
   - Parameters: DID, registration period

### Journey 4: Verifier Validation Journey (via Trust Registry)
1. `Verifier_Applicant` account is created with sufficient funds
2. Trust Registry and Credential Schema already exist (created by `Trust_Registry_Controller`)
3. `Trust_Registry_Controller` has root permission
4. `Verifier_Applicant` starts validation process:
   - Transaction: `Start Permission VP` (MOD-PERM-MSG-1)
   - Parameters: type=VERIFIER, Trust Registry's validator permission ID, country
5. `Verifier_Applicant` connects to `Trust_Registry_Controller`'s validation service
6. `Trust_Registry_Controller` validates the applicant:
   - Transaction: `Set Permission VP to Validated` (MOD-PERM-MSG-3)
   - Parameters: permission ID, effective until, validation/issuance/verification fees
7. `Verifier_Applicant` adds their DID to directory:
   - Transaction: `Add a DID` (MOD-DD-MSG-1)
   - Parameters: DID, registration period

### Journey 5: Credential Issuance Journey
1. `Credential_Holder` account is created with sufficient funds
2. Trust Registry and Credential Schema already exist (created by `Trust_Registry_Controller`)
3. `Issuer_Applicant` already has validated ISSUER permission
4. `Credential_Holder` has a compatible wallet application
5. `Credential_Holder` requests credential from `Issuer_Applicant`
6. `Credential_Holder`'s wallet provides UUID and wallet_agent_perm_id to `Issuer_Applicant`
7. `Issuer_Applicant` creates permission session:
   - Transaction: `Create or Update Permission Session` (MOD-PERM-MSG-10)
   - Parameters: session ID (UUID), issuer_perm_id, agent_perm_id, wallet_agent_perm_id
8. `Issuer_Applicant` issues credential to `Credential_Holder`'s wallet

### Journey 6: Credential Verification Journey
1. Trust Registry and Credential Schema already exist (created by `Trust_Registry_Controller`)
2. `Issuer_Applicant` already has validated ISSUER permission
3. `Verifier_Applicant` already has validated VERIFIER permission
4. `Credential_Holder` already has a credential in wallet
5. `Credential_Holder` presents credential to `Verifier_Applicant`
6. `Verifier_Applicant` creates permission session:
   - Transaction: `Create or Update Permission Session` (MOD-PERM-MSG-10)
   - Parameters: session ID, verifier permission ID, agent permission ID
7. `Verifier_Applicant` verifies the credential

### Journey 7: Permission Renewal Journey
1. Trust Registry and Credential Schema already exist (created by `Trust_Registry_Controller`)
2. `Permission_Holder` (could be `Issuer_Applicant`, `Verifier_Applicant`, etc.) already has a permission
3. `Permission_Holder`'s permission is approaching expiration
4. `Permission_Holder` initiates renewal:
   - Transaction: `Renew Permission VP` (MOD-PERM-MSG-2)
   - Parameters: permission ID
5. `Permission_Holder` connects to validator's service (either `Trust_Registry_Controller` or `Issuer_Grantor_Applicant`)
6. Validator validates renewal:
   - Transaction: `Set Permission VP to Validated` (MOD-PERM-MSG-3)
   - Parameters: permission ID, new effective until

### Journey 8: Permission Termination Journey
1. Trust Registry and Credential Schema already exist (created by `Trust_Registry_Controller`)
2. `Permission_Holder` already has a permission (ISSUER, VERIFIER, etc.)
3. `Permission_Holder` wants to terminate permission
4. `Permission_Holder` requests termination:
   - Transaction: `Request Permission VP Termination` (MOD-PERM-MSG-4)
   - Parameters: permission ID
5. Validator (`Trust_Registry_Controller` or `Issuer_Grantor_Applicant`) confirms termination:
   - Transaction: `Confirm Permission VP Termination` (MOD-PERM-MSG-5)
   - Parameters: permission ID
6. `Permission_Holder` reclaims trust deposit:
   - Transaction: `Reclaim Trust Deposit` (MOD-TD-MSG-3)
   - Parameters: amount to reclaim

### Journey 9: Governance Framework Update Journey
1. Trust Registry already exists (created by `Trust_Registry_Controller`)
2. `Trust_Registry_Controller` adds new governance framework document:
   - Transaction: `Add Governance Framework Document` (MOD-TR-MSG-2)
   - Parameters: trust registry ID, document URL, language, version
3. `Trust_Registry_Controller` activates new version:
   - Transaction: `Increase Active Governance Framework Version` (MOD-TR-MSG-3)
   - Parameters: trust registry ID
4. All ecosystem participants operate under updated framework

### Journey 10: Trust Deposit Management Journey
1. `Deposit_Holder` account has accumulated trust deposit from various activities
2. `Deposit_Holder` claims interest:
   - Transaction: `Reclaim Trust Deposit Interests` (MOD-TD-MSG-2)
   - No parameters
3. `Deposit_Holder` reclaims portion of deposit:
   - Transaction: `Reclaim Trust Deposit` (MOD-TD-MSG-3)
   - Parameters: amount to reclaim

### Journey 11: DID Management Journey
1. `DID_Owner` already has DID in directory
2. `DID_Owner` renews DID registration:
   - Transaction: `Renew a DID` (MOD-DD-MSG-2)
   - Parameters: DID, renewal period
3. `DID_Owner` updates DID indexing:
   - Transaction: `Touch a DID` (MOD-DD-MSG-4)
   - Parameters: DID
4. `DID_Owner` removes DID from directory:
   - Transaction: `Remove a DID` (MOD-DD-MSG-3)
   - Parameters: DID
5. `DID_Owner` reclaims associated trust deposit

### Journey 12: Permission Revocation Journey
1. Trust Registry and Credential Schema already exist (created by `Trust_Registry_Controller`)
2. Validator (`Trust_Registry_Controller` or `Issuer_Grantor_Applicant`) has granted permission to an entity
3. Validator detects non-compliance or other issue with `Permission_Holder`
4. Validator revokes granted permission:
   - Transaction: `Revoke Permission` (MOD-PERM-MSG-9)
   - Parameters: permission ID
5. Revoked `Permission_Holder` can no longer issue/verify credentials with that permission

### Journey 13: Permission Extension Journey
1. Trust Registry and Credential Schema already exist (created by `Trust_Registry_Controller`)
2. `Permission_Holder` already has a valid permission
3. Validator (`Trust_Registry_Controller` or `Issuer_Grantor_Applicant`) decides to extend permission validity
4. Validator extends permission validity:
   - Transaction: `Extend Permission` (MOD-PERM-MSG-8)
   - Parameters: permission ID, new effective until
5. `Permission_Holder` continues operations with extended validity period

### Journey 14: Credential Schema Update Journey
1. `Trust_Registry_Controller` creates a trust registry:
   - Transaction: `Create New Trust Registry` (MOD-TR-MSG-1)
   - Parameters: DID, governance framework document URL, language
2. `Trust_Registry_Controller` creates a credential schema:
   - Transaction: `Create a Credential Schema` (MOD-CS-MSG-1)
   - Parameters: trust registry ID, JSON schema, issuer/verifier permission management modes
3. `Trust_Registry_Controller` updates credential schema:
   - Transaction: `Update a Credential Schema` (MOD-CS-MSG-2)
   - Parameters: schema ID, validation periods
4. `Trust_Registry_Controller` archives schema when obsolete:
   - Transaction: `Archive Credential Schema` (MOD-CS-MSG-3)
   - Parameters: schema ID, archive=true
5. `Trust_Registry_Controller` unarchives schema if needed again:
   - Transaction: `Archive Credential Schema` (MOD-CS-MSG-3)
   - Parameters: schema ID, archive=false

### Journey 15: Failed Validation Journey
1. Trust Registry and Credential Schema already exist (created by `Trust_Registry_Controller`)
2. Validator (`Trust_Registry_Controller` or `Issuer_Grantor_Applicant`) has permission to validate applicants
3. `Failed_Applicant` starts validation process:
   - Transaction: `Start Permission VP` (MOD-PERM-MSG-1)
   - Parameters: type (ISSUER, VERIFIER, etc.), validator permission ID, country
4. `Failed_Applicant` connects to validator's service but fails requirements
5. `Failed_Applicant` cancels validation:
   - Transaction: `Cancel Permission VP Last Request` (MOD-PERM-MSG-6)
   - Parameters: permission ID
6. `Failed_Applicant` receives refund of validation fees
7. `Failed_Applicant` may start new validation process with different validator or after correcting issues

## Project Structure
```
verana-test-harness/
├── cmd/
│   └── main.go           # Entry point
├── lib/
│   ├── client.go         # Client setup
│   ├── helpers.go        # Helper functions
│   ├── queries.go        # Query operations
│   ├── transactions.go   # Transaction operations
│   └── utils.go          # Utility functions
├── journeys/
│   ├── journey01_trust_registry.go
│   ├── journey02_issuer_grantor.go
│   ├── journey03_issuer_validation.go
│   └── ...
├── go.mod
└── go.sum
```