package v2

import (
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/gogoproto/proto"
)

// TrustDepositV1 represents the old version of TrustDeposit with uint64 Share.
// This struct matches the old proto definition before migration.
type TrustDepositV1 struct {
	Account        string
	Share          uint64 // OLD: uint64, will be converted to LegacyDec
	Amount         uint64
	Claimable      uint64
	SlashedDeposit uint64
	RepaidDeposit  uint64
	LastSlashed    *time.Time
	LastRepaid     *time.Time
	SlashCount     uint64
	LastRepaidBy   string
}

// ToV2 converts the old TrustDepositV1 to the new TrustDeposit format.
// It converts Share from uint64 to math.LegacyDec.
func (old *TrustDepositV1) ToV2() *TrustDepositV2 {
	return &TrustDepositV2{
		Account:        old.Account,
		Share:          math.LegacyNewDec(int64(old.Share)), // Convert uint64 to LegacyDec
		Amount:         old.Amount,
		Claimable:      old.Claimable,
		SlashedDeposit: old.SlashedDeposit,
		RepaidDeposit:  old.RepaidDeposit,
		LastSlashed:    old.LastSlashed,
		LastRepaid:     old.LastRepaid,
		SlashCount:     old.SlashCount,
		LastRepaidBy:   old.LastRepaidBy,
	}
}

// TrustDepositV2 represents the new version of TrustDeposit with LegacyDec Share.
// This matches the current proto definition.
type TrustDepositV2 struct {
	Account        string
	Share          math.LegacyDec // NEW: LegacyDec
	Amount         uint64
	Claimable      uint64
	SlashedDeposit uint64
	RepaidDeposit  uint64
	LastSlashed    *time.Time
	LastRepaid     *time.Time
	SlashCount     uint64
	LastRepaidBy   string
}

// UnmarshalOldTrustDeposit unmarshals raw bytes using the old proto definition.
// This function reads the old uint64 Share field.
func UnmarshalOldTrustDeposit(bz []byte, cdc interface {
	Unmarshal(bz []byte, ptr proto.Message) error
}) (*TrustDepositV1, error) {
	// Create a minimal proto message that matches the old structure
	// We'll manually parse the proto bytes
	old := &TrustDepositV1{}

	// For now, we'll use a simple approach: try to unmarshal with the old structure
	// In practice, you might need to use proto.Unmarshal directly with the old proto definition

	// Since we can't easily regenerate old pb.go files, we'll use a workaround:
	// Read the proto wire format manually or use a proto parser

	// For a more robust solution, you could:
	// 1. Keep the old pb.go files in a separate package
	// 2. Or manually parse the proto wire format
	// 3. Or use reflection to read the uint64 field

	// For now, let's use proto.Unmarshal with a message that has the old structure
	// We'll need to create a proto message type that matches the old definition

	// This is a placeholder - in practice, you'd need the actual old proto message type
	_ = bz
	_ = cdc
	_ = old

	return nil, nil // Placeholder - will be implemented below
}
