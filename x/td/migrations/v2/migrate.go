package v2

import (
	"time"

	"cosmossdk.io/core/store"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/verana-labs/verana/x/td/types"
)

// Keeper defines the interface required for migration.
// This interface allows the migration to work without importing the keeper package,
// breaking the cyclic dependency.
type Keeper interface {
	// GetStoreService returns the store service to access raw store
	GetStoreService() store.KVStoreService
	// GetCodec returns the codec for unmarshaling
	GetCodec() codec.BinaryCodec
	// GetLogger returns the logger
	GetLogger() interface {
		Info(msg string, keyvals ...interface{})
	}
}

// MigrateStore performs in-place store migrations from v1 to v2.
// The migration converts Share from uint64 to LegacyDec.
//
// Strategy:
// 1. Access the raw store to read old data
// 2. Iterate over all trust deposit keys
// 3. For each entry, read raw bytes and unmarshal with old proto (uint64 Share)
// 4. Convert Share from uint64 to LegacyDec
// 5. Save using new proto definition
func MigrateStore(ctx sdk.Context, k Keeper) error {
	logger := k.GetLogger()
	logger.Info("Starting migration: converting Share from uint64 to LegacyDec")

	storeService := k.GetStoreService()
	cdc := k.GetCodec()

	// Get the raw KVStore
	kvStore := storeService.OpenKVStore(ctx)

	// The trust_deposit map uses prefix 1 (from types.TrustDepositKey)
	// We need to iterate over all keys with this prefix
	prefix := []byte{0x01} // collections.NewPrefix(1) = []byte{0x01}

	// Calculate end prefix: increment the last byte
	endPrefix := make([]byte, len(prefix))
	copy(endPrefix, prefix)
	if len(endPrefix) > 0 {
		endPrefix[len(endPrefix)-1]++
	}

	// Iterate over all keys with the trust_deposit prefix
	iterator, err := kvStore.Iterator(prefix, endPrefix)
	if err != nil {
		return err
	}
	defer iterator.Close()

	migratedCount := 0
	skippedCount := 0
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		value := iterator.Value()

		// First, try to unmarshal with the new proto definition (LegacyDec Share)
		// This handles the case where the migration was partially run or data is already migrated
		var newTD types.TrustDeposit
		valueCodec := codec.CollValue[types.TrustDeposit](cdc)
		newTD, err := valueCodec.Decode(value)
		if err == nil {
			// Data is already in the new format, skip it
			skippedCount++
			continue
		}

		// Try to unmarshal with old proto definition (uint64 Share)
		oldTD, err := unmarshalOldTrustDeposit(value, cdc)
		if err != nil {
			logger.Info("Failed to unmarshal trust deposit (neither old nor new format), skipping", "key", string(key), "error", err)
			continue
		}

		// Convert Share from uint64 to LegacyDec
		newShare := math.LegacyNewDec(int64(oldTD.Share))

		// Convert timestamps from timestamppb.Timestamp to *time.Time
		var lastSlashed *time.Time
		var lastRepaid *time.Time
		if oldTD.LastSlashed != nil {
			t := oldTD.LastSlashed.AsTime()
			lastSlashed = &t
		}
		if oldTD.LastRepaid != nil {
			t := oldTD.LastRepaid.AsTime()
			lastRepaid = &t
		}

		// Create new TrustDeposit with LegacyDec Share using the actual types
		newTD = types.TrustDeposit{
			Account:        oldTD.Account,
			Share:          newShare, // Now LegacyDec
			Amount:         oldTD.Amount,
			Claimable:      oldTD.Claimable,
			SlashedDeposit: oldTD.SlashedDeposit,
			RepaidDeposit:  oldTD.RepaidDeposit,
			LastSlashed:    lastSlashed,
			LastRepaid:     lastRepaid,
			SlashCount:     oldTD.SlashCount,
			LastRepaidBy:   oldTD.LastRepaidBy,
		}

		// Marshal the new structure using the same encoder that collections.Map uses
		// This ensures the LegacyDec custom type is properly encoded
		// CollValue wraps the codec to handle custom types correctly
		newBz, err := valueCodec.Encode(newTD)
		if err != nil {
			logger.Info("Failed to encode new trust deposit", "key", string(key), "error", err)
			continue
		}

		// Save the migrated data
		kvStore.Set(key, newBz)
		migratedCount++

		logger.Info("Migrated trust deposit", "account", oldTD.Account, "old_share", oldTD.Share, "new_share", newShare.String())
	}

	logger.Info("Migration completed", "migrated_count", migratedCount, "skipped_count", skippedCount)
	return nil
}

// unmarshalOldTrustDeposit unmarshals raw bytes using the old proto definition with uint64 Share.
// This function uses proto.Unmarshal with a message that matches the old structure.
func unmarshalOldTrustDeposit(bz []byte, cdc codec.BinaryCodec) (*TrustDepositV1, error) {
	// Use proto.Unmarshal with a message that matches the old structure
	msg := &OldTrustDepositProto{}

	err := proto.Unmarshal(bz, msg)
	if err != nil {
		return nil, err
	}

	// Convert to our struct
	return &TrustDepositV1{
		Account:        msg.Account,
		Share:          msg.Share, // This is uint64 in old proto
		Amount:         msg.Amount,
		Claimable:      msg.Claimable,
		SlashedDeposit: msg.SlashedDeposit,
		RepaidDeposit:  msg.RepaidDeposit,
		LastSlashed:    msg.LastSlashed,
		LastRepaid:     msg.LastRepaid,
		SlashCount:     msg.SlashCount,
		LastRepaidBy:   msg.LastRepaidBy,
	}, nil
}

// OldTrustDepositProto is a proto message that matches the old structure with uint64 Share.
// This is used for unmarshaling old data.
// It implements proto.Message interface.
type OldTrustDepositProto struct {
	Account        string                 `protobuf:"bytes,1,opt,name=account,proto3" json:"account,omitempty"`
	Share          uint64                 `protobuf:"varint,2,opt,name=share,proto3" json:"share,omitempty"` // OLD: uint64
	Amount         uint64                 `protobuf:"varint,3,opt,name=amount,proto3" json:"amount,omitempty"`
	Claimable      uint64                 `protobuf:"varint,4,opt,name=claimable,proto3" json:"claimable,omitempty"`
	SlashedDeposit uint64                 `protobuf:"varint,5,opt,name=slashed_deposit,json=slashedDeposit,proto3" json:"slashed_deposit,omitempty"`
	RepaidDeposit  uint64                 `protobuf:"varint,6,opt,name=repaid_deposit,json=repaidDeposit,proto3" json:"repaid_deposit,omitempty"`
	LastSlashed    *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=last_slashed,json=lastSlashed,proto3" json:"last_slashed,omitempty"`
	LastRepaid     *timestamppb.Timestamp `protobuf:"bytes,8,opt,name=last_repaid,json=lastRepaid,proto3" json:"last_repaid,omitempty"`
	SlashCount     uint64                 `protobuf:"varint,9,opt,name=slash_count,json=slashCount,proto3" json:"slash_count,omitempty"`
	LastRepaidBy   string                 `protobuf:"bytes,10,opt,name=last_repaid_by,json=lastRepaidBy,proto3" json:"last_repaid_by,omitempty"`
}

// Implement proto.Message interface
func (m *OldTrustDepositProto) Reset() { *m = OldTrustDepositProto{} }
func (m *OldTrustDepositProto) String() string {
	if m == nil {
		return "OldTrustDepositProto<nil>"
	}
	return "OldTrustDepositProto"
}
func (*OldTrustDepositProto) ProtoMessage() {}
