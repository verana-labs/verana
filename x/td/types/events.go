package types

const (
	EventTypeRepaySlashedTrustDeposit = "repay_slashed_trust_deposit"
	EventTypeReclaimTrustDepositYield = "reclaim_trust_deposit_yield"
	EventTypeReclaimTrustDeposit      = "reclaim_trust_deposit"
)

const (
	AttributeKeyAccount        = "account"
	AttributeKeyAmount         = "amount"
	AttributeKeyRepaidBy       = "repaid_by"
	AttributeKeyTimestamp      = "timestamp"
	AttributeKeyClaimedYield   = "claimed_yield"
	AttributeKeySharesReduced  = "shares_reduced"
	AttributeKeyClaimedAmount  = "claimed_amount"
	AttributeKeyBurnedAmount   = "burned_amount"
	AttributeKeyTransferAmount = "transfer_amount"
)
