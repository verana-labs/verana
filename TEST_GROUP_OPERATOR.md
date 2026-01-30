# Testing Group-Based Operator Allowance

This document outlines the steps to test the group-based operator allowance feature.

## Prerequisites

- Chain running in fresh state
- `cooluser` account with funds (keyring-backend test)

## Test Flow

### Step 1: Create Group Members

```bash
# Create two group member keys
veranad keys add groupadmin1 --keyring-backend test
veranad keys add groupadmin2 --keyring-backend test

# Fund them from cooluser
veranad tx bank send cooluser $(veranad keys show groupadmin1 -a --keyring-backend test) 10000000uvna --from cooluser --keyring-backend test -y
veranad tx bank send cooluser $(veranad keys show groupadmin2 -a --keyring-backend test) 10000000uvna --from cooluser --keyring-backend test -y
```

### Step 2: Create Operator Account

```bash
# Create operator key
veranad keys add operator1 --keyring-backend test

# Fund operator (for gas)
veranad tx bank send cooluser $(veranad keys show operator1 -a --keyring-backend test) 1000000uvna --from cooluser --keyring-backend test -y
```

### Step 3: Create Group

```bash
# Create group with the two admins
veranad tx group create-group $(veranad keys show groupadmin1 -a --keyring-backend test) "Test Group" members.json --from groupadmin1 --keyring-backend test -y
```

**members.json:**
```json
{
  "members": [
    {"address": "<groupadmin1_address>", "weight": "1"},
    {"address": "<groupadmin2_address>", "weight": "1"}
  ]
}
```

### Step 4: Create Group Policy

```bash
veranad tx group create-group-policy $(veranad keys show groupadmin1 -a --keyring-backend test) <group_id> "operational" policy.json --from groupadmin1 --keyring-backend test -y
```

**policy.json:**
```json
{
  "@type": "/cosmos.group.v1.ThresholdDecisionPolicy",
  "threshold": "1",
  "windows": {
    "voting_period": "120s",
    "min_execution_period": "0s"
  }
}
```

### Step 5: Fund Group Policy

```bash
# Get group policy address
GROUP_POLICY=$(veranad q group group-policies-by-group <group_id> -o json | jq -r '.group_policies[0].address')

# Fund the group policy for trust deposits
veranad tx bank send cooluser $GROUP_POLICY 100000000uvna --from cooluser --keyring-backend test -y
```

### Step 6: Add Operator via Group Proposal

Create a proposal to add operator1 with 1000000 uvna allowance:

```bash
# Create proposal JSON (proposal_add_operator.json)
cat > proposal_add_operator.json << EOF
{
  "group_policy_address": "$GROUP_POLICY",
  "proposers": ["<groupadmin1_address>"],
  "messages": [
    {
      "@type": "/verana.td.v1.MsgAddOperator",
      "creator": "$GROUP_POLICY",
      "operator": "<operator1_address>",
      "allowance": "1000000",
      "reset_period_seconds": "86400"
    }
  ],
  "metadata": "Add operator1 with 1M uvna daily allowance",
  "title": "Add Operator",
  "summary": "Add operator1 for trust registry management"
}
EOF

# Submit proposal
veranad tx group submit-proposal proposal_add_operator.json --from groupadmin1 --keyring-backend test -y

# Vote on proposal (both admins)
veranad tx group vote <proposal_id> $(veranad keys show groupadmin1 -a --keyring-backend test) VOTE_OPTION_YES "" --from groupadmin1 --keyring-backend test -y

# Execute proposal
veranad tx group exec <proposal_id> --from groupadmin1 --keyring-backend test -y
```

### Step 7: Test Delegated Execution

```bash
# Operator creates trust registry on behalf of group
veranad tx tr create-trust-registry \
  "did:verana:testregistry" \
  "en" \
  "https://example.com/gf.json" \
  "sha384-xxxxx" \
  --group $GROUP_POLICY \
  --from operator1 \
  --keyring-backend test -y
```

### Step 8: Query and Verify State

```bash
# Query trust registry
veranad q tr list-trust-registries

# Check controller is the group policy address
veranad q tr get-trust-registry 1

# Query operator allowance (if query is available)
# veranad q td operator-allowance $GROUP_POLICY <operator1_address>
```

## Expected Results

1. Trust registry created with controller = group policy address (not operator1)
2. Operator usage incremented by trust deposit amount
3. Event emitted with operator attribution

## Error Cases to Test

1. Unauthorized operator tries to use --group flag → should fail
2. Operator exceeds allowance → should fail
3. No --group flag → works normally (operator is controller)

---

## Actual Test Results (2026-01-26)

### Test Accounts Created
- **groupadmin1**: `verana1678rzc39r0007tgcw5rtuwggt4mzffpfyavm3g`
- **groupadmin2**: `verana1kcglyks64tdhv2ruckuh6x8sm076nkzpsxlyct`
- **operator1**: `verana1dx9l3rm2lz96p2hdqa3js7sgd36f9h0d9pg4zz`

### Group Configuration
- **Group ID**: 1
- **Group Policy Address**: `verana1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsh3z8fv`
- **Decision Policy**: Threshold 1, Voting Period 2m

### Operator Addition
- **Proposal ID**: 1
- **Status**: EXECUTED ✅
- **Allowance**: 10,000,000 uvna
- **Reset Period**: 86400 seconds (daily)

### Trust Registry Created via Delegated Execution
✅ **TEST PASSED**

```json
{
  "id": "1",
  "did": "did:verana:testregistry001",
  "controller": "verana1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsh3z8fv",
  "deposit": "10000000"
}
```

**Key Verification**:
- Controller is the **group policy address**, not the operator
- Trust deposit was charged to the group's balance
- Transaction signed by operator1 but executed on behalf of group
