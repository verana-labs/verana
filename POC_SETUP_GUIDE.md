# Anchor POC Setup Guide

This guide walks you through testing the Anchor-based Trust Deposit POC on a running chain.

## Prerequisites

- Chain running locally (`veranad start`)
- `veranad` binary installed
- `cooluser` account already in keyring (genesis account with funds)

---

## Step 1: Recover Test Accounts

We'll use existing test mnemonics to create accounts.

```bash
# Anchor Admin 1 (Trust Registry Controller)
echo "simple stuff order coach cliff advance ugly dial right forward boring rhythm comfort initial girl either universe genre pony sort own cycle hurt grit" | veranad keys add anchor_admin1 --recover --keyring-backend test

# Anchor Admin 2 (Issuer Grantor)
echo "peace load monkey fuel safe rally ship panic vapor script confirm acid size silent grit muscle olive scissors seat drift vital universe affair hero" | veranad keys add anchor_admin2 --recover --keyring-backend test

# VS Operator 1 (Hot Key) - Issuer
echo "aim fold come benefit stuff file host joy doll grid credit garbage helmet frown rubber depart project dinosaur leisure relax equip sting flat grief" | veranad keys add vs_operator1 --recover --keyring-backend test

# VS Operator 2 (Hot Key) - Verifier
echo "intact link bench vapor sense during carbon symptom grab drop ramp city life bomb ice lock mimic wine furnace often buzz muscle bird layer" | veranad keys add vs_operator2 --recover --keyring-backend test
```

Verify accounts:
```bash
veranad keys list --keyring-backend test
```

---

## Step 2: Fund Accounts

Send 2 VNA from cooluser to each account for gas:

```bash
# Get addresses
ADMIN1=$(veranad keys show anchor_admin1 -a --keyring-backend test)
ADMIN2=$(veranad keys show anchor_admin2 -a --keyring-backend test)
OPERATOR1=$(veranad keys show vs_operator1 -a --keyring-backend test)
OPERATOR2=$(veranad keys show vs_operator2 -a --keyring-backend test)

# Fund anchor admins
veranad tx bank send cooluser $ADMIN1 2000000uvna --from cooluser --keyring-backend test --chain-id vna-testnet-1 -y
veranad tx bank send cooluser $ADMIN2 2000000uvna --from cooluser --keyring-backend test --chain-id vna-testnet-1 -y

# Fund VS operators  
veranad tx bank send cooluser $OPERATOR1 2000000uvna --from cooluser --keyring-backend test --chain-id vna-testnet-1 -y
veranad tx bank send cooluser $OPERATOR2 2000000uvna --from cooluser --keyring-backend test --chain-id vna-testnet-1 -y
```

### Check Balances (Visual Verification)

After sending funds, verify each account's balance:

```bash
veranad query bank balances $ADMIN1
veranad query bank balances $ADMIN2
veranad query bank balances $OPERATOR1
veranad query bank balances $OPERATOR2
```

---

## Step 3: Create Group and Group Policy (Anchor)

Create the JSON files for group members and decision policy with **1-minute voting period** for quick testing:

```bash
# Create members.json
cat > /tmp/members.json << EOF
{
  "members": [
    {
      "address": "$ADMIN1",
      "weight": "1",
      "metadata": "Admin 1"
    },
    {
      "address": "$ADMIN2",
      "weight": "1", 
      "metadata": "Admin 2"
    }
  ]
}
EOF

# Create decision policy (threshold of 1, 1-minute voting for testing)
cat > /tmp/policy.json << 'EOF'
{
  "@type": "/cosmos.group.v1.ThresholdDecisionPolicy",
  "threshold": "1",
  "windows": {
    "voting_period": "60s",
    "min_execution_period": "0s"
  }
}
EOF
```

Create the group with policy:
```bash
veranad tx group create-group-with-policy \
  $ADMIN1 \
  "Anchor POC Group" \
  "Anchor Policy for POC" \
  /tmp/members.json \
  /tmp/policy.json \
  --group-policy-as-admin \
  --from anchor_admin1 \
  --keyring-backend test \
  --chain-id vna-testnet-1 \
  -y
```

Query to get the group policy address (this is the ANCHOR_ID):
```bash
# Get group policy address (ANCHOR_ID)
veranad query group group-policies-by-group 1
```

Save the group policy address:
```bash
# Copy address from query result
ANCHOR_ID="verana1..."
```

---

## Step 4: Register Anchor via Group Proposal

Since anchor operations require `creator == anchor_id`, we must submit via group proposal.

### 4.1 Create Register Anchor Proposal

```bash
# Create the proposal message JSON
cat > /tmp/register_anchor_msg.json << EOF
{
  "group_policy_address": "$ANCHOR_ID",
  "messages": [
    {
      "@type": "/verana.td.v1.MsgRegisterAnchor",
      "creator": "$ANCHOR_ID",
      "anchor_id": "$ANCHOR_ID",
      "group_id": "1",
      "metadata": "POC Test Anchor"
    }
  ],
  "metadata": "Register anchor for POC testing",
  "proposers": ["$ADMIN1"],
  "title": "Register Anchor",
  "summary": "Register group policy as anchor in TD module"
}
EOF
```

### 4.2 Submit Proposal

```bash
veranad tx group submit-proposal /tmp/register_anchor_msg.json \
  --from anchor_admin1 \
  --keyring-backend test \
  --chain-id vna-testnet-1 \
  -y
```

### 4.3 Vote on Proposal

```bash
# Get proposal ID (usually 1)
veranad query group proposals-by-group-policy $ANCHOR_ID

# Vote yes (args: proposal-id, voter, vote-option, metadata)
veranad tx group vote 1 $ADMIN1 VOTE_OPTION_YES "" \
  --from anchor_admin1 \
  --keyring-backend test \
  --chain-id vna-testnet-1 \
  -y
```

### 4.4 Execute Proposal (wait 1 minute for voting period)

```bash
# Wait for voting period to end, then execute
sleep 65
veranad tx group exec 1 \
  --from anchor_admin1 \
  --keyring-backend test \
  --chain-id vna-testnet-1 \
  -y
```

### 4.5 Verify Anchor Registration

```bash
# Query the anchor directly
veranad query td get-anchor $ANCHOR_ID

# Expected output:
# anchor:
#   anchor_id: verana1...
#   created: "2026-01-13T..."
#   group_id: "1"
#   metadata: POC Test Anchor

# Query proposal status (should be PROPOSAL_EXECUTOR_RESULT_SUCCESS)
veranad query group proposals-by-group-policy $ANCHOR_ID
```

---

## Step 5: Register VS Operators via Group Proposal

### 5.1 Create VS Registration Proposals

```bash
# Register VS Operator 1
cat > /tmp/register_vs1_msg.json << EOF
{
  "group_policy_address": "$ANCHOR_ID",
  "messages": [
    {
      "@type": "/verana.td.v1.MsgRegisterVerifiableService",
      "creator": "$ANCHOR_ID",
      "anchor_id": "$ANCHOR_ID",
      "operator_account": "$OPERATOR1",
      "metadata": "VS Operator 1 - Issuer Service"
    }
  ],
  "metadata": "Register VS Operator 1",
  "proposers": ["$ADMIN1"],
  "title": "Register VS Operator 1",
  "summary": "Register hot key for VS operator 1"
}
EOF

# Submit proposal
veranad tx group submit-proposal /tmp/register_vs1_msg.json \
  --from anchor_admin1 \
  --keyring-backend test \
  --chain-id vna-testnet-1 \
  -y

# Vote and execute (after voting period)
veranad tx group vote 2 $ADMIN1 VOTE_OPTION_YES "" --from anchor_admin1 --keyring-backend test --chain-id vna-testnet-1 -y
sleep 65
veranad tx group exec 2 --from anchor_admin1 --keyring-backend test --chain-id vna-testnet-1 -y
```

Repeat for Operator 2:
```bash
cat > /tmp/register_vs2_msg.json << EOF
{
  "group_policy_address": "$ANCHOR_ID",
  "messages": [
    {
      "@type": "/verana.td.v1.MsgRegisterVerifiableService",
      "creator": "$ANCHOR_ID",
      "anchor_id": "$ANCHOR_ID",
      "operator_account": "$OPERATOR2",
      "metadata": "VS Operator 2 - Verifier Service"
    }
  ],
  "metadata": "Register VS Operator 2",
  "proposers": ["$ADMIN1"],
  "title": "Register VS Operator 2",
  "summary": "Register hot key for VS operator 2"
}
EOF

veranad tx group submit-proposal /tmp/register_vs2_msg.json --from anchor_admin1 --keyring-backend test --chain-id vna-testnet-1 -y
veranad tx group vote 3 $ADMIN1 VOTE_OPTION_YES "" --from anchor_admin1 --keyring-backend test --chain-id vna-testnet-1 -y
sleep 65
veranad tx group exec 3 --from anchor_admin1 --keyring-backend test --chain-id vna-testnet-1 -y
```

### 5.2 Verify VS Registrations

```bash
# Verify Operator 1 is registered
veranad query td get-verifiable-service $OPERATOR1

# Verify Operator 2 is registered
veranad query td get-verifiable-service $OPERATOR2
```

---

## Step 6: Set Operator Allowances via Group Proposal

```bash
# Set allowance for Operator 1: 500,000 uvna per day
cat > /tmp/set_allowance1_msg.json << EOF
{
  "group_policy_address": "$ANCHOR_ID",
  "messages": [
    {
      "@type": "/verana.td.v1.MsgSetOperatorAllowance",
      "creator": "$ANCHOR_ID",
      "anchor_id": "$ANCHOR_ID",
      "operator_account": "$OPERATOR1",
      "allowance_limit": "500000",
      "reset_period": "86400"
    }
  ],
  "metadata": "Set allowance for Operator 1",
  "proposers": ["$ADMIN1"],
  "title": "Set Operator 1 Allowance",
  "summary": "500K uvna daily limit"
}
EOF

veranad tx group submit-proposal /tmp/set_allowance1_msg.json --from anchor_admin1 --keyring-backend test --chain-id vna-testnet-1 -y
veranad tx group vote 4 $ADMIN1 VOTE_OPTION_YES "" --from anchor_admin1 --keyring-backend test --chain-id vna-testnet-1 -y
sleep 65
veranad tx group exec 4 --from anchor_admin1 --keyring-backend test --chain-id vna-testnet-1 -y
```

### 6.1 Verify Operator Allowance

```bash
# Verify allowance for Operator 1
veranad query td get-operator-allowance $ANCHOR_ID $OPERATOR1

# Expected output:
# operator_allowance:
#   allowance_limit: "500000"
#   anchor_id: verana1...
#   operator_account: verana1...
#   reset_period: "86400"
#   spent: "0"
```

---

## Step 7: Grant x/authz Permissions (Required)

> **Important**: For VS operators to execute transactions **on behalf of the anchor** (where the anchor is the creator/owner of the resulting on-chain entity), x/authz grants are **required**. This is essential for the anchor-based model where:
> - DIDs are owned by the anchor (not the operator)
> - Permissions are granted to the anchor (not the operator)
> - The anchor accumulates trust deposits from operations

Without x/authz: Operators can only sign as themselves. The TD module will route trust deposits to the anchor, but the created resources would be owned by the operator's account.

With x/authz: Operators execute `MsgExec` wrapping the actual message, where `creator = anchor_id`. The anchor becomes the owner.

```bash
# Grant Operator 1 permission to execute MsgAddDID on behalf of anchor
cat > /tmp/authz_grant_msg.json << EOF
{
  "group_policy_address": "$ANCHOR_ID",
  "messages": [
    {
      "@type": "/cosmos.authz.v1beta1.MsgGrant",
      "granter": "$ANCHOR_ID",
      "grantee": "$OPERATOR1",
      "grant": {
        "authorization": {
          "@type": "/cosmos.authz.v1beta1.GenericAuthorization",
          "msg": "/verana.dd.v1.MsgAddDID"
        }
      }
    }
  ],
  "metadata": "Grant authz to Operator 1 for DID operations",
  "proposers": ["$ADMIN1"],
  "title": "Grant Authz to Operator 1",
  "summary": "Allow operator to execute MsgAddDID on behalf of anchor"
}
EOF

veranad tx group submit-proposal /tmp/authz_grant_msg.json --from anchor_admin1 --keyring-backend test --chain-id vna-testnet-1 -y
veranad tx group vote 5 $ADMIN1 VOTE_OPTION_YES "" --from anchor_admin1 --keyring-backend test --chain-id vna-testnet-1 -y
sleep 65
veranad tx group exec 5 --from anchor_admin1 --keyring-backend test --chain-id vna-testnet-1 -y
```

### 7.1 Operator Executing via Authz

Once the grant is in place, the operator can execute operations on behalf of the anchor. The workflow requires two steps:

**Step 1: Generate the unsigned transaction (as if from anchor)**
```bash
# Generate the DID registration tx as if from anchor (unsigned)
# Syntax: veranad tx dd add-did [did] [years] [flags]
veranad tx dd add-did did:web:example.com 1 \
  --from $ANCHOR_ID \
  --chain-id vna-testnet-1 \
  --generate-only > /tmp/add_did_tx.json
```

**Step 2: Execute via authz (signed by operator)**
```bash
# Operator executes the tx on behalf of anchor
veranad tx authz exec /tmp/add_did_tx.json \
  --from vs_operator1 \
  --keyring-backend test \
  --chain-id vna-testnet-1 \
  -y
```

This creates a DID where:
- **Owner**: `$ANCHOR_ID` (the anchor)
- **Signer**: `vs_operator1` (the operator with authz grant)
- **Trust Deposit**: Routed to the anchor via `adjustTrustDepositAnchorAware`

---

## Quick Reference

| Entity | Key Name | Purpose |
|--------|----------|---------|
| Admin 1 | `anchor_admin1` | Group member, can sign txs |
| Admin 2 | `anchor_admin2` | Group member, can sign txs |
| Operator 1 | `vs_operator1` | VS hot key, 500K daily limit |
| Operator 2 | `vs_operator2` | VS hot key |
| Funder | `cooluser` | Genesis account with funds |
| Anchor | `$ANCHOR_ID` | Group policy address |

---

## Understanding the Flow

1. **Trust Deposits Accumulate from Operations** - NOT from direct funding
2. When a VS operator performs an operation (e.g., registers a DID), the `adjustTrustDepositAnchorAware` helper is called
3. **Module Integration Complete**: All VPR modules (`x/dd`, `x/perm`, `x/cs`, `x/tr`) now automatically:
   - Check if the account is an **anchor** → route to `AdjustAnchorTrustDeposit`
   - Check if the account is a **VS operator** → resolve anchor and enforce spending limits
   - Fall back to regular `AdjustTrustDeposit` for non-anchor accounts
4. Positive adjustments increase the trust deposit
5. Negative adjustments (debits) check operator allowance first
6. All anchor-level operations require group proposals

---

## Troubleshooting

**Error: unauthorized - creator must be the anchor**
- Submit the transaction via group proposal (see Step 4)

**Error: anchor not found**
- Make sure to register the anchor first before registering VS or setting allowances

**Error: proposal not found**
- Check proposal IDs with: `veranad query group proposals-by-group-policy $ANCHOR_ID`

**Error: voting period not ended**
- Wait 60 seconds (or check the voting_period in your policy)
