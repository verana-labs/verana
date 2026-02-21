#!/bin/bash
# scripts/setup_group.sh
#
# This script creates a Cosmos SDK group with 3 members and a decision policy
# with a 1-minute voting period. It assumes the chain is already running and
# the 'cooluser' account has funds to pay for the transactions.
#
# Prerequisites:
#   - Chain is running (veranad start)
#   - 'cooluser' key exists in the test keyring
#
# The script will:
#   1. Recover 3 group member keys from hardcoded mnemonics
#   2. Fund the group admin from the cooluser account
#   3. Create a group with policy (threshold=2, voting_period=1m)

set -e

# Function to log messages
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Variables
CHAIN_ID="vna-testnet-1"
BINARY="veranad"
KEYRING="test"
FEES="750000uvna"

# ─────────────────────────────────────────────────────────────────────────────
# Hardcoded mnemonics for reproducible key generation (generated once, reused)
# ─────────────────────────────────────────────────────────────────────────────
SEED_PHRASE_GROUP_ADMIN="nature noble gospel breeze flight salt clerk shuffle match secret cheese alarm artwork unit luxury can other vehicle wall wagon view tiger blue strong"
SEED_PHRASE_GROUP_MEMBER1="wagon crater tent spawn year north beach menu item unhappy damage spin flush south tackle van hat rabbit virtual holiday quote antique lock cereal"
SEED_PHRASE_GROUP_MEMBER2="camera nice autumn border illegal drill robot final elevator usage device unhappy blast enough weather ordinary clean document acoustic pistol behind equal what local"

# Existing validator / funder
FUNDER="cooluser"

log "=========================================="
log "  Setting up Cosmos SDK Group"
log "=========================================="

# ─────────────────────────────────────────────────────────────────────────────
# Step 1: Recover group member keys (idempotent – deletes if already exist)
# ─────────────────────────────────────────────────────────────────────────────
recover_key() {
    local name="$1"
    local mnemonic="$2"

    # Delete if already exists (ignore errors)
    $BINARY keys delete "$name" --keyring-backend $KEYRING -y 2>/dev/null || true

    echo "$mnemonic" | $BINARY keys add "$name" --recover --keyring-backend $KEYRING
    log "✅ Recovered key: $name"
}

log "Recovering group member keys..."
recover_key "group_admin"   "$SEED_PHRASE_GROUP_ADMIN"
recover_key "group_member1" "$SEED_PHRASE_GROUP_MEMBER1"
recover_key "group_member2" "$SEED_PHRASE_GROUP_MEMBER2"

# Get addresses
ADMIN_ADDR=$($BINARY keys show group_admin   --keyring-backend $KEYRING -a)
MEMBER1_ADDR=$($BINARY keys show group_member1 --keyring-backend $KEYRING -a)
MEMBER2_ADDR=$($BINARY keys show group_member2 --keyring-backend $KEYRING -a)

log ""
log "Addresses:"
log "  group_admin:   $ADMIN_ADDR"
log "  group_member1: $MEMBER1_ADDR"
log "  group_member2: $MEMBER2_ADDR"

# ─────────────────────────────────────────────────────────────────────────────
# Step 2: Fund the group admin from cooluser
# ─────────────────────────────────────────────────────────────────────────────
FUND_AMOUNT="100000000uvna"

log ""
log "Funding group_admin with $FUND_AMOUNT from $FUNDER..."
$BINARY tx bank send $FUNDER $ADMIN_ADDR $FUND_AMOUNT \
    --chain-id $CHAIN_ID \
    --keyring-backend $KEYRING \
    --fees $FEES \
    -y \
    --broadcast-mode sync

log "Waiting for transaction to be included in a block..."
sleep 6

# Verify balance
BALANCE=$($BINARY q bank balances $ADMIN_ADDR -o json | jq -r '.balances[] | select(.denom=="uvna") | .amount')
log "✅ group_admin balance: ${BALANCE}uvna"

# ─────────────────────────────────────────────────────────────────────────────
# Step 3: Create members JSON file
# ─────────────────────────────────────────────────────────────────────────────
MEMBERS_FILE=$(mktemp /tmp/group_members.XXXXXX.json)
cat > "$MEMBERS_FILE" <<EOF
{
    "members": [
        {
            "address": "$ADMIN_ADDR",
            "weight": "1",
            "metadata": "group admin"
        },
        {
            "address": "$MEMBER1_ADDR",
            "weight": "1",
            "metadata": "group member 1"
        },
        {
            "address": "$MEMBER2_ADDR",
            "weight": "1",
            "metadata": "group member 2"
        }
    ]
}
EOF

log ""
log "Members JSON:"
cat "$MEMBERS_FILE"

# ─────────────────────────────────────────────────────────────────────────────
# Step 4: Create decision policy JSON file (threshold=2, voting_period=1m)
# ─────────────────────────────────────────────────────────────────────────────
POLICY_FILE=$(mktemp /tmp/group_policy.XXXXXX.json)
cat > "$POLICY_FILE" <<EOF
{
    "@type": "/cosmos.group.v1.ThresholdDecisionPolicy",
    "threshold": "2",
    "windows": {
        "voting_period": "60s",
        "min_execution_period": "0s"
    }
}
EOF

log ""
log "Decision Policy JSON:"
cat "$POLICY_FILE"

# ─────────────────────────────────────────────────────────────────────────────
# Step 5: Create group with policy
# ─────────────────────────────────────────────────────────────────────────────
log ""
log "Creating group with policy..."
TX_RESULT=$($BINARY tx group create-group-with-policy \
    "$ADMIN_ADDR" \
    "verana-trust-registry-group" \
    "trust registry group policy" \
    "$MEMBERS_FILE" \
    "$POLICY_FILE" \
    --group-policy-as-admin \
    --chain-id $CHAIN_ID \
    --keyring-backend $KEYRING \
    --fees $FEES \
    --gas auto \
    --gas-adjustment 1.5 \
    -y \
    --broadcast-mode sync \
    -o json 2>&1)

log "Transaction result:"
echo "$TX_RESULT" | jq . 2>/dev/null || echo "$TX_RESULT"

log "Waiting for transaction to be included in a block..."
sleep 6

# ─────────────────────────────────────────────────────────────────────────────
# Step 6: Query and display the created group
# ─────────────────────────────────────────────────────────────────────────────
log ""
log "Querying created groups..."
GROUPS=$($BINARY q group groups -o json)
echo "$GROUPS" | jq .

# Get the group ID (latest created group)
GROUP_ID=$(echo "$GROUPS" | jq -r '.groups[-1].id // empty')

log ""
log "Querying group policies for group $GROUP_ID..."
GROUP_POLICIES=$($BINARY q group group-policies-by-group "$GROUP_ID" -o json)
echo "$GROUP_POLICIES" | jq .

GROUP_POLICY_ADDR=$(echo "$GROUP_POLICIES" | jq -r '.group_policies[0].address // empty')

if [ -n "$GROUP_POLICY_ADDR" ]; then
    log ""
    log "=========================================="
    log "  ✅ Group Setup Complete!"
    log "=========================================="
    log ""
    log "  Group Policy Address: $GROUP_POLICY_ADDR"
    log "  This is the 'authority' address to use in trust registry messages."
    log ""
    log "  Members:"
    log "    - group_admin:   $ADMIN_ADDR (weight: 1)"
    log "    - group_member1: $MEMBER1_ADDR (weight: 1)"
    log "    - group_member2: $MEMBER2_ADDR (weight: 1)"
    log ""
    log "  Decision Policy:"
    log "    - Threshold: 2 (need 2 of 3 votes)"
    log "    - Voting Period: 60s (1 minute)"
    log "    - Min Execution Period: 0s"
    log ""
    log "=========================================="
else
    log "⚠️  Could not determine group policy address. Check the transaction output above."
fi

# Cleanup temp files
rm -f "$MEMBERS_FILE" "$POLICY_FILE"

log ""
log "Done!"
