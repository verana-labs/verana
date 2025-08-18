package types

// Event types and attribute keys for the DID Directory module
const (
	EventTypeAddDID    = "add_did"
	EventTypeRenewDID  = "renew_did"
	EventTypeRemoveDID = "remove_did"
	EventTouchDID      = "touch_did"

	// Attribute keys
	AttributeKeyDID        = "did"
	AttributeKeyController = "controller"
	AttributeKeyExpiration = "expiration"
	AttributeKeyDeposit    = "deposit"
	AttributeKeyYears      = "years"
)
