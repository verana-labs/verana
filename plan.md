# Implementation Plan: Group-Based Operator Allowance Tracking

## Overview

This plan implements the Group-based architecture from [Issue #185](https://github.com/verana-labs/verana/issues/185) to enable per-operator allowance tracking when operators execute transactions on behalf of a Group.

---

## Part 1: Data Model

### Operator State (x/td module)

Based on Section 8.2 of Issue #185:

```
(group_id, operator_account) → OperatorAllowance {
    allowance: uint64           // Maximum allowed spend per period
    usage: uint64               // Current period usage
    reset_period_seconds: int64 // Duration in seconds (e.g., 86400 for daily)
    last_reset_at: timestamp    // When usage was last reset
    last_usage_at: timestamp    // When operator last used allowance
    active: bool                // Whether operator is active
}
```

### Proto Definition

```protobuf
// proto/verana/td/v1/operator.proto
message OperatorAllowance {
    string group = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
    string operator = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
    uint64 allowance = 3;           // Max spend per period
    uint64 usage = 4;               // Current period usage
    int64 reset_period_seconds = 5; // Reset period in seconds (0 = never reset)
    google.protobuf.Timestamp last_reset_at = 6 [(gogoproto.stdtime) = true, (gogoproto.nullable) = true];
    google.protobuf.Timestamp last_usage_at = 7 [(gogoproto.stdtime) = true, (gogoproto.nullable) = true];
    bool active = 8;
}
```

### Keeper Methods (x/td)

| Method | Description |
|--------|-------------|
| `SetOperatorAllowance(ctx, group, operator, allowance)` | Create/update operator |
| `GetOperatorAllowance(ctx, group, operator)` | Get operator info |
| `IsAuthorizedOperator(ctx, group, operator)` | Check if operator is active |
| `IncrementOperatorUsage(ctx, group, operator, amount)` | Track usage (fails if exceeds allowance) |
| `ResetOperatorUsage(ctx, group, operator)` | Reset usage counter |

---

## Part 2: Targeted Transaction Messages

Based on Fabrice's updated comment in Issue #185, these messages need the `group` field:

### Trust Registry (`x/tr`)

| Message | Add `group` field |
|---------|-------------------|
| MsgCreateTrustRegistry | ✅ |
| MsgAddGovernanceFrameworkDocument | ✅ |
| MsgIncreaseActiveGovernanceFrameworkVersion | ✅ |
| MsgUpdateTrustRegistry | ✅ |
| MsgArchiveTrustRegistry | ✅ |

### Credential Schema (`x/cs`)

| Message | Add `group` field |
|---------|-------------------|
| MsgCreateCredentialSchema | ✅ |
| MsgUpdateCredentialSchema | ✅ |
| MsgArchiveCredentialSchema | ✅ |

### Permission (`x/perm`)

| Message | Add `group` field | Notes |
|---------|-------------------|-------|
| MsgStartPermissionVP | ✅ | Controller is group |
| MsgRenewPermissionVP | ✅ | |
| MsgSetPermissionVPToValidated | ✅ | Validator's group |
| MsgCancelPermissionVPLastRequest | ✅ | |
| MsgCreateRootPermission | ✅ | Grantor is group |
| MsgCreatePermission | ✅ | Controller is group |
| MsgExtendPermission | ✅ | |
| MsgRevokePermission | ✅ | Must hold auth from ancestor |
| MsgCreateOrUpdatePermissionSession | ❌ | Operator signs directly (see note) |
| MsgSlashPermissionTrustDeposit | ❌ | Governance only |
| MsgRepayPermissionSlashedTrustDeposit | ✅ | |

> **Note on MsgCreateOrUpdatePermissionSession**: Per Fabrice's comment, the `operator` field on `Permission` defines who can call this. The operator signs directly - no group delegation needed.

### Trust Deposit (`x/td`)

| Message | Add `group` field |
|---------|-------------------|
| MsgReclaimTrustDepositYield | ✅ |
| MsgReclaimTrustDeposit | ✅ |
| MsgRepaySlashedTrustDeposit | ✅ |

> **Note**: DID Directory (`x/dd`) is NOT included in this implementation.

---

## Part 3: Proto Changes

Add `group` field to affected messages. Example:

```protobuf
message MsgCreateTrustRegistry {
    option (cosmos.msg.v1.signer) = "creator";
    
    string creator = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
    string group = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];  // NEW
    string did = 3;
    // ... other fields
}
```

---

## Part 4: Handler Logic (Detailed)

### When `--group` Flag is Used

**CLI Side:**
```bash
veranad tx tr create-trust-registry did:verana:123 --from operator1 --group verana1group...
```

1. CLI reads `--group` flag value
2. CLI constructs message with:
   - `msg.Creator` = operator address (from `--from`)
   - `msg.Group` = group address (from `--group`)
3. Transaction is signed by operator and broadcast

**Handler Side (when msg.Group is NOT empty):**

```
Step 1: Determine the Controller
─────────────────────────────────
• msg.Creator = operator (who signed the tx)
• msg.Group = group (who owns the resulting object)
• controller = msg.Group

Step 2: Validate Operator Authorization
───────────────────────────────────────
• Call: k.tdKeeper.IsAuthorizedOperator(ctx, msg.Group, msg.Creator)
• This checks if:
  - An OperatorAllowance record exists for (group, operator)
  - The operator is marked as active
• If NOT authorized → return ErrUnauthorizedOperator

Step 3: Check Operator Allowance (before operation)
──────────────────────────────────────────────────
• Get current allowance/usage: k.tdKeeper.GetOperatorAllowance(ctx, msg.Group, msg.Creator)
• Calculate estimated cost for this operation
• Check: if (current_usage + cost) > allowance → return ErrAllowanceExceeded

Step 4: Execute the Operation
────────────────────────────
• The operation uses msg.Group as the "actor"
• Any bank transfers use msg.Group's balance
• Any trust deposit operations use msg.Group's deposit
• Object is stored with controller = msg.Group

Step 5: Update Operator Usage
────────────────────────────
• Call: k.tdKeeper.IncrementOperatorUsage(ctx, msg.Group, msg.Creator, actualCost)
• This updates: OperatorAllowance.usage += actualCost
```

**Handler Side (when msg.Group is empty - default/existing behavior):**

```
• msg.Creator = operator (who signed the tx)
• controller = msg.Creator
• All operations use msg.Creator's balance
• No operator allowance checks
• Object is stored with controller = msg.Creator
```

### Example Flow: MsgCreateTrustRegistry

```
Operator: verana1operator...
Group: verana1group...
Allowance: 500 uvna
Current Usage: 100 uvna

1. Operator submits: veranad tx tr create-trust-registry --group verana1group...

2. Handler receives:
   - msg.Creator = verana1operator...
   - msg.Group = verana1group...

3. Authorization check:
   - IsAuthorizedOperator(verana1group..., verana1operator...) → true ✅

4. Allowance check:
   - Current: 100 uvna used of 500 uvna allowance
   - This operation costs: 50 uvna
   - 100 + 50 = 150 < 500 → OK ✅

5. Execute:
   - Trust deposit adjustment uses verana1group...'s deposit
   - TrustRegistry created with controller = verana1group...

6. Update usage:
   - IncrementOperatorUsage(verana1group..., verana1operator..., 50)
   - New usage: 150 uvna
```

### State After Operation

| Field | Value |
|-------|-------|
| TrustRegistry.Controller | verana1group... |
| TrustRegistry.CreatedBy | verana1operator... (optional - see Question 1) |
| OperatorAllowance.usage | 150 uvna |

---

## Part 5: Operator Management (x/td)

### How Operators are Added (via Group Proposal)

Operators are managed via **group proposals**. Only the group itself can add/remove operators.

**Flow:**
1. Group admin creates a proposal containing `MsgAddOperator`
2. Group members vote on the proposal
3. If approved, the group executes the proposal
4. `MsgAddOperator` is executed with `msg.Creator = group` (the group signs)

```bash
# proposal.json example:
{
  "group_policy_address": "verana1group...",
  "messages": [{
    "@type": "/verana.td.v1.MsgAddOperator",
    "creator": "verana1group...",
    "operator": "verana1operator...",
    "allowance": "1000000",
    "reset_period_seconds": "86400"
  }]
}
```

### Operator Management Messages

| Message | Purpose |
|---------|---------|
| MsgAddOperator | Add operator (allowance, reset_period_seconds) |
| MsgRemoveOperator | Remove operator from group |
| MsgUpdateOperatorAllowance | Update allowance/reset period |
| MsgResetOperatorUsage | Manually reset usage counter |

> **Note**: All messages require `msg.Creator` = group address (executed via group proposal).

### Usage Reset Logic (Automatic)

When operator executes a transaction:

```
1. Get operator record
2. Check if reset period elapsed:
   if current_time - last_reset_at >= reset_period_seconds:
       usage = 0
       last_reset_at = current_time
3. Check: usage + cost <= allowance (else reject)
4. After success:
   usage += cost
   last_usage_at = current_time
5. Emit EventOperatorUsage with amount_spent
```

### Example: Daily Reset

```
Allowance: 500 uvna | Reset Period: 86400s (24h) | Last Reset: Jan 22, 10:00

• Jan 22, 15:00 → 5h < 24h → no reset, usage += cost
• Jan 23, 12:00 → 26h >= 24h → reset to 0, then add cost
```

---

## Part 6: CLI Changes

Add `--group` flag to all affected transaction commands:

```bash
veranad tx tr create-trust-registry [did] --from operator --group verana1group...
veranad tx perm create-root-permission [args] --from operator --group verana1group...
```

---

## Implementation Order

> **Phase 1 Focus**: We will implement the complete flow for `MsgCreateTrustRegistry` first as a reference implementation, then extend to other messages.

### Phase 1: MsgCreateTrustRegistry (Current)

1. [x] Operator data model and keeper methods (x/td)
2. [x] Operator management messages (x/td) 
3. [x] Add `group` field to `MsgCreateTrustRegistry` proto
4. [x] Regenerate protos (`make proto-gen`)
5. [x] Update `MsgCreateTrustRegistry` handler
6. [x] Update `create-trust-registry` CLI command
7. [ ] Tests

### Phase 2: Extend to Other Messages (Later)

- Trust Registry: remaining messages
- Credential Schema: all messages
- Permission: targeted messages
- Trust Deposit: targeted messages

---

## Questions for Review

1. ~~Should we add `created_by` field to stored objects to track the operator?~~ (defer to Phase 2)
2. ~~Should operator usage reset on a schedule or only manually?~~ (Answered: automatic reset based on `reset_period_seconds`)

