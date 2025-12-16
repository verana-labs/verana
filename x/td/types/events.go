package types

const (
	EventTypeSlashTrustDeposit        = "slash_trust_deposit"
	EventTypeRepaySlashedTrustDeposit = "repay_slashed_trust_deposit"
	EventTypeReclaimTrustDepositYield = "reclaim_trust_deposit_yield"
	EventTypeReclaimTrustDeposit      = "reclaim_trust_deposit"
	EventTypeAdjustTrustDeposit       = "adjust_trust_deposit"
	EventTypeYieldDistribution        = "yield_distribution"
	EventTypeYieldTransfer            = "yield_transfer"
)

const (
	AttributeKeyAccount              = "account"
	AttributeKeyAmount               = "amount"
	AttributeKeySlashCount           = "slash_count"
	AttributeKeyRepaidBy             = "repaid_by"
	AttributeKeyTimestamp            = "timestamp"
	AttributeKeyClaimedYield         = "claimed_yield"
	AttributeKeySharesReduced       = "shares_reduced"
	AttributeKeyClaimedAmount        = "claimed_amount"
	AttributeKeyBurnedAmount        = "burned_amount"
	AttributeKeyTransferAmount       = "transfer_amount"
	AttributeKeyAugend              = "augend"
	AttributeKeyAdjustmentType      = "adjustment_type"
	AttributeKeyNewAmount           = "new_amount"
	AttributeKeyNewShare            = "new_share"
	AttributeKeyNewClaimable        = "new_claimable"
	AttributeKeyYIPIncomingBalance  = "yip_incoming_balance"
	AttributeKeyYIPIncomingBalanceDec = "yip_incoming_balance_dec"
	AttributeKeyYIPBalanceBefore   = "yip_balance_before"
	AttributeKeyAllowance            = "allowance"
	AttributeKeyTrustDepositBalance = "trust_deposit_balance"
	AttributeKeyTransferAmountDec    = "transfer_amount_dec"
)
