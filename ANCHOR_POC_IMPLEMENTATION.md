# Anchor-Based Trust Deposit POC Implementation Guide

> **Issue Reference**: [#185 - Anchor-Based Control, Authorization, and Trust Deposit Architecture](https://github.com/verana-labs/verana/issues/185)
> 
> **Cosmos SDK Version**: v0.53.4

---

## POC Objectives

| # | Objective | Status |
|---|-----------|--------|
| 1 | Anchor-Linked Trust Deposit | ✅ Implemented |
| 2 | VS Registration with Hot Keys | ✅ Implemented |
| 3 | Operator Spending from Anchor | ✅ Implemented |
| 4 | Per-VS Spend Limits | ✅ Implemented |

---

## Architecture Overview

```
SETUP PHASE:
═════════════

    ┌────────────────┐     ┌────────────────┐
    │ Admin Account 1│     │ Admin Account 2│
    └───────┬────────┘     └───────┬────────┘
            │                      │
            └──────────┬───────────┘
                       │ create group with policy
                       ▼
              ┌────────────────────┐
              │  Group Policy Acct │  ◄── This IS the anchor_id
              │  (verana1anchor...)│     (real on-chain account)
              └─────────┬──────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
        ▼               ▼               ▼
   ┌─────────┐    ┌──────────┐    ┌──────────┐
   │ Register│    │ Register │    │ Register │
   │ Anchor  │    │   VS 1   │    │   VS 2   │
   └────┬────┘    └────┬─────┘    └────┬─────┘
        │              │               │
        ▼              ▼               ▼
 ┌──────────────┐ ┌────────────┐  ┌────────────┐
 │ Anchor Entry │ │ VS Op Key 1│  │ VS Op Key 2│
 │ ──────────── │ │ verana1op1 │  │ verana1op2 │
 │ anchor_id    │ └─────┬──────┘  └─────┬──────┘
 │ group_id: 1  │       │               │
 └──────────────┘       │ set allowance │
                        ▼               ▼
                 ┌────────────┐  ┌────────────┐
                 │ Allowance 1│  │ Allowance 2│
                 │ limit: 1M  │  │ limit: 500K│
                 └────────────┘  └────────────┘


OPERATION PHASE (Trust Deposit Accumulation):
═════════════════════════════════════════════

    VS Operator 1 performs operation          VS Operator 2 performs operation
    (e.g., register DID)                      (e.g., create Permission)
           │                                         │
           ▼                                         ▼
    ┌─────────────────┐                       ┌─────────────────┐
    │ Operation       │                       │ Operation       │
    │ creator: op1    │                       │ creator: op2    │
    └────────┬────────┘                       └────────┬────────┘
             │                                         │
             │ 1. Resolve op1 → anchor_id              │ 1. Resolve op2 → anchor_id
             │ 2. Check allowance (for debits)         │ 2. Check allowance (for debits)
             │ 3. AdjustAnchorTrustDeposit             │ 3. AdjustAnchorTrustDeposit
             │                                         │
             └──────────────────┬──────────────────────┘
                                │
                                ▼
                      ┌──────────────────┐
                      │   TrustDeposit   │
                      │   (anchor-based) │
                      │   ────────────── │
                      │   anchor_id: ... │
                      │   amount updates │
                      └──────────────────┘


KEY INSIGHT:
════════════
  • VS operators (hot keys) sign transactions
  • Operators hold minimal funds (just for gas)
  • Trust deposits accumulate from operations (NOT funded directly)
  • Positive adjustments: increase TD (e.g., from DID registration)
  • Negative adjustments: decrease TD (checked against allowance)
  • Each operator has spending limits for debits
  • Anchor = Group Policy Account = controls everything via group proposals
```

---

## Data Structures

### Proto Definitions (`proto/verana/td/v1/anchor.proto`)

```protobuf
message Anchor {
  string anchor_id = 1;  // Group policy address (primary key)
  uint64 group_id = 2;   // x/group group ID
  google.protobuf.Timestamp created = 3;
  string metadata = 4;
}

message VerifiableService {
  string anchor_id = 1;         // Parent anchor
  string operator_account = 2;  // Hot key address (primary key)
  bool active = 4;
  string metadata = 5;
}

message OperatorAllowance {
  string anchor_id = 1;
  string operator_account = 2;
  uint64 allowance_limit = 3;   // Max per period
  uint64 spent = 4;             // Current period spend
  uint64 reset_period = 5;      // Period in seconds
}
```

---

## Core Keeper Functions (`x/td/keeper/anchor_poc.go`)

### Registration Functions
| Function | Purpose |
|----------|---------|
| `RegisterAnchor(anchorID, groupID, metadata)` | Register group policy as Anchor |
| `RegisterVerifiableService(anchorID, operator, metadata)` | Register hot key for Anchor |
| `SetOperatorAllowance(anchorID, operator, limit, period)` | Set spending limits |

### Trust Deposit Functions
| Function | Purpose |
|----------|---------|
| `AdjustAnchorTrustDeposit(anchorID, augend, operator)` | Main function - handles +/- adjustments |
| `DebitAnchorTrustDeposit(anchorID, amount, operator, reason)` | Convenience wrapper for debits |

### Resolution Functions
| Function | Purpose |
|----------|---------|
| `IsAnchor(anchorID)` | Check if address is registered anchor |
| `GetAnchorForOperator(operatorAccount)` | Resolve operator → anchor_id |
| `IsVerifiableService(operatorAccount)` | Check if address is VS operator |

---

## Transaction Messages (`proto/verana/td/v1/tx.proto`)

```protobuf
service Msg {
  rpc RegisterAnchor(MsgRegisterAnchor) returns (MsgRegisterAnchorResponse);
  rpc RegisterVerifiableService(MsgRegisterVerifiableService) returns (MsgRegisterVerifiableServiceResponse);
  rpc SetOperatorAllowance(MsgSetOperatorAllowance) returns (MsgSetOperatorAllowanceResponse);
}
```

**Authorization**: All anchor operations require `msg.Creator == msg.AnchorId`. Since the anchor is a group policy account, transactions must be submitted via group proposals.

---

## CLI Commands

```bash
# Register anchor (requires group proposal)
veranad tx td register-anchor --anchor-id <addr> --group-id <id> --metadata "..."

# Register VS operator (requires group proposal)
veranad tx td register-verifiable-service --anchor-id <addr> --operator-account <addr>

# Set operator allowance (requires group proposal)
veranad tx td set-operator-allowance --anchor-id <addr> --operator-account <addr> --allowance-limit 1000000 --reset-period 86400
```

---

## Testing

Run POC unit tests:
```bash
go test ./x/td/keeper/... -run "Test(Anchor|Verifiable|Operator|DirectAnchor|Allowance)" -v
```

---

## Future Work (Post-POC)

1. **x/authz Integration**: Grant VS operators permission to execute specific messages
2. **Module Integration**: Update `x/dd`, `x/perm` to call `AdjustAnchorTrustDeposit`
3. **Query Endpoints**: Add queries for anchors, VS operators, allowances
4. **Genesis Support**: Export/import anchor state
5. **Migration**: Migrate existing account-based TDs to anchor-based
