#!/bin/bash
# scripts/e2e_operator_create_trust_registry.sh
#
# End-to-end flow: Group proposal -> Operator onboarding -> Trust Registry creation
#
# Prerequisites:
#   - Chain is running (veranad start)
#   - 'cooluser' key exists in the test keyring with funds
#
# This script will:
#   1. Create 3 group member accounts and fund them
#   2. Create a group with threshold=2, voting_period=60s
#   3. Fund the group policy account
#   4. Create an operator account and fund with dust amount
#   5. Submit a group proposal to grant operator authorization (MsgGrantOperatorAuthorization)
#   6. Vote YES on the proposal with 2 members
#   7. Wait for voting period to expire
#   8. Execute the proposal
#   9. Create a trust registry using the operator

set -e

# ─────────────────────────────────────────────────────────────────────────────
# Configuration
# ─────────────────────────────────────────────────────────────────────────────
CHAIN_ID="vna-testnet-1"
BINARY="veranad"
KEYRING="test"
FEES="750000uvna"
FUNDER="cooluser"
FUND_AMOUNT="100000000uvna"       # 100 VNA for group admin
OPERATOR_FUND="15000000uvna"        # 15 VNA for operator (10M trust deposit + gas)

log() {
    echo ""
    echo "$(date '+%H:%M:%S') | $1"
}

wait_tx() {
    log "  Waiting for tx to land..."
    sleep 6
}

# ─────────────────────────────────────────────────────────────────────────────
# Hardcoded mnemonics (deterministic, for test only)
# ─────────────────────────────────────────────────────────────────────────────
SEED_GROUP_ADMIN="nature noble gospel breeze flight salt clerk shuffle match secret cheese alarm artwork unit luxury can other vehicle wall wagon view tiger blue strong"
SEED_GROUP_MEMBER1="wagon crater tent spawn year north beach menu item unhappy damage spin flush south tackle van hat rabbit virtual holiday quote antique lock cereal"
SEED_GROUP_MEMBER2="camera nice autumn border illegal drill robot final elevator usage device unhappy blast enough weather ordinary clean document acoustic pistol behind equal what local"
SEED_OPERATOR="venture quality volcano maximum lesson smoke someone make lunar vintage flag lunch people inherit sock bike quote diet federal chef spike remain grab school"

# ═════════════════════════════════════════════════════════════════════════════
# STEP 1: Create and fund group member accounts
# ═════════════════════════════════════════════════════════════════════════════
log "==========================================="
log "  STEP 1: Create and fund group members"
log "==========================================="

recover_key() {
    local name="$1"
    local mnemonic="$2"
    $BINARY keys delete "$name" --keyring-backend $KEYRING -y 2>/dev/null || true
    echo "$mnemonic" | $BINARY keys add "$name" --recover --keyring-backend $KEYRING 2>/dev/null
    log "  Recovered key: $name"
}

recover_key "group_admin"    "$SEED_GROUP_ADMIN"
recover_key "group_member1"  "$SEED_GROUP_MEMBER1"
recover_key "group_member2"  "$SEED_GROUP_MEMBER2"
recover_key "tr_operator"    "$SEED_OPERATOR"

ADMIN_ADDR=$($BINARY keys show group_admin   --keyring-backend $KEYRING -a)
MEMBER1_ADDR=$($BINARY keys show group_member1 --keyring-backend $KEYRING -a)
MEMBER2_ADDR=$($BINARY keys show group_member2 --keyring-backend $KEYRING -a)
OPERATOR_ADDR=$($BINARY keys show tr_operator  --keyring-backend $KEYRING -a)

log "  Addresses:"
log "    group_admin:   $ADMIN_ADDR"
log "    group_member1: $MEMBER1_ADDR"
log "    group_member2: $MEMBER2_ADDR"
log "    tr_operator:   $OPERATOR_ADDR"

# Fund group_admin (needs funds to create group and submit proposal)
log "  Funding group_admin with $FUND_AMOUNT..."
$BINARY tx bank send $FUNDER $ADMIN_ADDR $FUND_AMOUNT \
    --chain-id $CHAIN_ID --keyring-backend $KEYRING --fees $FEES -y --broadcast-mode sync -o json | jq -r '.txhash'
wait_tx

# Fund group_member1 (needs funds to vote)
log "  Funding group_member1 with $FUND_AMOUNT..."
$BINARY tx bank send $FUNDER $MEMBER1_ADDR $FUND_AMOUNT \
    --chain-id $CHAIN_ID --keyring-backend $KEYRING --fees $FEES -y --broadcast-mode sync -o json | jq -r '.txhash'
wait_tx

# Fund group_member2 (needs funds to vote)
log "  Funding group_member2 with $FUND_AMOUNT..."
$BINARY tx bank send $FUNDER $MEMBER2_ADDR $FUND_AMOUNT \
    --chain-id $CHAIN_ID --keyring-backend $KEYRING --fees $FEES -y --broadcast-mode sync -o json | jq -r '.txhash'
wait_tx

# Fund operator (needs enough for trust deposit + gas)
log "  Funding tr_operator with $OPERATOR_FUND..."
$BINARY tx bank send $FUNDER $OPERATOR_ADDR $OPERATOR_FUND \
    --chain-id $CHAIN_ID --keyring-backend $KEYRING --fees $FEES -y --broadcast-mode sync -o json | jq -r '.txhash'
wait_tx

# ═════════════════════════════════════════════════════════════════════════════
# STEP 2: Create group with policy (threshold=2, voting=60s)
# ═════════════════════════════════════════════════════════════════════════════
log "==========================================="
log "  STEP 2: Create group with decision policy"
log "==========================================="

MEMBERS_FILE=$(mktemp /tmp/group_members.XXXXXX.json)
cat > "$MEMBERS_FILE" <<EOF
{
    "members": [
        { "address": "$ADMIN_ADDR",   "weight": "1", "metadata": "group admin" },
        { "address": "$MEMBER1_ADDR", "weight": "1", "metadata": "group member 1" },
        { "address": "$MEMBER2_ADDR", "weight": "1", "metadata": "group member 2" }
    ]
}
EOF

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

$BINARY tx group create-group-with-policy \
    "$ADMIN_ADDR" \
    "verana-trust-registry-group" \
    "trust registry group policy" \
    "$MEMBERS_FILE" \
    "$POLICY_FILE" \
    --group-policy-as-admin \
    --chain-id $CHAIN_ID --keyring-backend $KEYRING --fees $FEES --gas auto --gas-adjustment 1.5 \
    -y --broadcast-mode sync -o json | jq -r '.txhash'
wait_tx

# Get group policy address
GROUP_ID=$($BINARY q group groups -o json | jq -r '.groups[-1].id')
GROUP_POLICY_ADDR=$($BINARY q group group-policies-by-group "$GROUP_ID" -o json | jq -r '.group_policies[0].address')

log "  Group ID:             $GROUP_ID"
log "  Group Policy Address: $GROUP_POLICY_ADDR"

if [ -z "$GROUP_POLICY_ADDR" ] || [ "$GROUP_POLICY_ADDR" = "null" ]; then
    log "ERROR: Failed to get group policy address!"
    exit 1
fi

# ═════════════════════════════════════════════════════════════════════════════
# STEP 3: Fund the group policy account
# ═════════════════════════════════════════════════════════════════════════════
log "==========================================="
log "  STEP 3: Fund group policy account"
log "==========================================="

log "  Funding group policy with $FUND_AMOUNT..."
$BINARY tx bank send $FUNDER $GROUP_POLICY_ADDR $FUND_AMOUNT \
    --chain-id $CHAIN_ID --keyring-backend $KEYRING --fees $FEES -y --broadcast-mode sync -o json | jq -r '.txhash'
wait_tx

POLICY_BALANCE=$($BINARY q bank balances $GROUP_POLICY_ADDR -o json | jq -r '.balances[] | select(.denom=="uvna") | .amount // "0"')
log "  Group policy balance: ${POLICY_BALANCE}uvna"

# ═════════════════════════════════════════════════════════════════════════════
# STEP 4: Submit group proposal — MsgGrantOperatorAuthorization
# ═════════════════════════════════════════════════════════════════════════════
log "==========================================="
log "  STEP 4: Submit proposal to grant operator authorization"
log "==========================================="

PROPOSAL_FILE=$(mktemp /tmp/grant_operator_proposal.XXXXXX.json)
cat > "$PROPOSAL_FILE" <<EOF
{
    "group_policy_address": "$GROUP_POLICY_ADDR",
    "proposers": ["$ADMIN_ADDR"],
    "metadata": "Grant operator authorization to tr_operator for MsgCreateTrustRegistry",
    "messages": [
        {
            "@type": "/verana.de.v1.MsgGrantOperatorAuthorization",
            "authority": "$GROUP_POLICY_ADDR",
            "operator": "",
            "grantee": "$OPERATOR_ADDR",
            "msg_types": ["/verana.tr.v1.MsgCreateTrustRegistry"],
            "with_feegrant": false
        }
    ],
    "exec": 0,
    "title": "Grant TR Operator Authorization",
    "summary": "Authorize tr_operator to create trust registries on behalf of the group"
}
EOF

log "  Proposal JSON:"
cat "$PROPOSAL_FILE"
log ""

$BINARY tx group submit-proposal "$PROPOSAL_FILE" \
    --from group_admin \
    --chain-id $CHAIN_ID --keyring-backend $KEYRING --fees $FEES --gas auto --gas-adjustment 1.5 \
    -y --broadcast-mode sync -o json | jq -r '.txhash'
wait_tx

# Get the proposal ID
PROPOSAL_ID=$($BINARY q group proposals-by-group-policy "$GROUP_POLICY_ADDR" -o json | jq -r '.proposals[-1].id')
log "  Proposal ID: $PROPOSAL_ID"

if [ -z "$PROPOSAL_ID" ] || [ "$PROPOSAL_ID" = "null" ]; then
    log "ERROR: Failed to get proposal ID!"
    # Try to debug
    $BINARY q group proposals-by-group-policy "$GROUP_POLICY_ADDR" -o json | jq .
    exit 1
fi

# ═════════════════════════════════════════════════════════════════════════════
# STEP 5: Vote YES on the proposal (need 2 of 3 for threshold)
#         The second vote uses --exec 1 (EXEC_TRY) to auto-execute once
#         threshold is met (min_execution_period=0s allows this).
# ═════════════════════════════════════════════════════════════════════════════
log "==========================================="
log "  STEP 5: Vote on proposal (2 of 3 members)"
log "==========================================="

log "  group_admin voting YES..."
$BINARY tx group vote "$PROPOSAL_ID" "$ADMIN_ADDR" VOTE_OPTION_YES "" \
    --from group_admin \
    --chain-id $CHAIN_ID --keyring-backend $KEYRING --fees $FEES \
    -y --broadcast-mode sync -o json | jq -r '.txhash'
wait_tx

log "  group_member1 voting YES (with --exec 1 to auto-execute)..."
$BINARY tx group vote "$PROPOSAL_ID" "$MEMBER1_ADDR" VOTE_OPTION_YES "" \
    --from group_member1 \
    --exec 1 \
    --chain-id $CHAIN_ID --keyring-backend $KEYRING --fees $FEES --gas auto --gas-adjustment 1.5 \
    -y --broadcast-mode sync -o json | jq -r '.txhash'
wait_tx

# The proposal should be auto-executed now (pruned from state after success).
# Try to query it — if "not found", it was executed and pruned (success).
log "  Checking proposal status..."
PROPOSAL_QUERY=$($BINARY q group proposal "$PROPOSAL_ID" -o json 2>&1) || true

if echo "$PROPOSAL_QUERY" | grep -q "not found"; then
    log "  Proposal was executed and pruned (this is expected)"
else
    PROPOSAL_STATUS=$(echo "$PROPOSAL_QUERY" | jq -r '.proposal.status // "unknown"')
    EXEC_RESULT=$(echo "$PROPOSAL_QUERY" | jq -r '.proposal.executor_result // "unknown"')
    log "  Proposal status: $PROPOSAL_STATUS"
    log "  Executor result: $EXEC_RESULT"

    if [ "$PROPOSAL_STATUS" = "PROPOSAL_STATUS_ACCEPTED" ] && [ "$EXEC_RESULT" != "PROPOSAL_EXECUTOR_RESULT_SUCCESS" ]; then
        log "  Proposal accepted but not yet executed. Executing now..."
        $BINARY tx group exec "$PROPOSAL_ID" \
            --from group_admin \
            --chain-id $CHAIN_ID --keyring-backend $KEYRING --fees $FEES --gas auto --gas-adjustment 1.5 \
            -y --broadcast-mode sync -o json | jq -r '.txhash'
        wait_tx
    fi
fi

# ═════════════════════════════════════════════════════════════════════════════
# STEP 8: Create trust registry using the operator
# ═════════════════════════════════════════════════════════════════════════════
log "==========================================="
log "  STEP 8: Create trust registry using operator"
log "==========================================="

log "  Operator ($OPERATOR_ADDR) creating trust registry..."
log "  Authority (group policy): $GROUP_POLICY_ADDR"

$BINARY tx tr create-trust-registry \
    "$GROUP_POLICY_ADDR" \
    "did:example:e2e-test-trust-registry" \
    "en" \
    "https://example.com/governance-framework-v1" \
    "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26" \
    --from tr_operator \
    --chain-id $CHAIN_ID --keyring-backend $KEYRING --fees $FEES --gas auto --gas-adjustment 1.5 \
    -y --broadcast-mode sync -o json | jq -r '.txhash'
wait_tx

# ═════════════════════════════════════════════════════════════════════════════
# STEP 9: Verify the trust registry was created
# ═════════════════════════════════════════════════════════════════════════════
log "==========================================="
log "  STEP 9: Verify trust registry"
log "==========================================="

TR_LIST=$($BINARY q tr list-trust-registries -o json)
echo "$TR_LIST" | jq .

TR_COUNT=$(echo "$TR_LIST" | jq '.trust_registries | length')
log "  Total trust registries: $TR_COUNT"

if [ "$TR_COUNT" -gt 0 ]; then
    LAST_TR=$(echo "$TR_LIST" | jq '.trust_registries[-1]')
    TR_ID=$(echo "$LAST_TR" | jq -r '.id')
    TR_DID=$(echo "$LAST_TR" | jq -r '.did')
    TR_CONTROLLER=$(echo "$LAST_TR" | jq -r '.controller')
    TR_LANGUAGE=$(echo "$LAST_TR" | jq -r '.language')

    log ""
    log "==========================================="
    log "  SUCCESS!"
    log "==========================================="
    log ""
    log "  Trust Registry ID:    $TR_ID"
    log "  DID:                  $TR_DID"
    log "  Controller:           $TR_CONTROLLER"
    log "  Language:              $TR_LANGUAGE"
    log ""
    log "  Controller matches group policy: $([ "$TR_CONTROLLER" = "$GROUP_POLICY_ADDR" ] && echo 'YES' || echo 'NO')"
    log ""
else
    log "ERROR: No trust registries found!"
    log "  Check the transaction logs above for errors."
fi

# Cleanup
rm -f "$MEMBERS_FILE" "$POLICY_FILE" "$PROPOSAL_FILE"

log ""
log "Done!"
