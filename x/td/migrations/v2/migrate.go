package v2

import (
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

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
	// GetTrustDepositMap returns the TrustDeposit collections.Map for migration
	GetTrustDepositMap() collections.Map[string, types.TrustDeposit]
}

// MigrateStore performs in-place store migrations from v1 to v2.
// The migration converts Share from uint64 to LegacyDec.
//
// Strategy:
// 1. Use collections.Map to iterate over all trust deposits (ensures correct key encoding)
// 2. For each entry, try to read with new format first (already migrated)
// 3. If that fails, read raw bytes and unmarshal with old proto (uint64 Share)
// 4. Convert Share from uint64 to LegacyDec
// 5. Save using collections.Map.Set() (ensures correct encoding)
func MigrateStore(ctx sdk.Context, k Keeper) error {
	logger := k.GetLogger()
	logger.Info("Starting migration: converting Share from uint64 to LegacyDec")

	storeService := k.GetStoreService()
	cdc := k.GetCodec()
	trustDepositMap := k.GetTrustDepositMap()

	// Get the raw KVStore for reading old format data
	kvStore := storeService.OpenKVStore(ctx)

	// The trust_deposit map uses prefix 1 (from types.TrustDepositKey)
	// We need to iterate over all keys with this prefix to read old format
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
	valueCodec := codec.CollValue[types.TrustDeposit](cdc)

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		value := iterator.Value()

		// First, try to unmarshal with new proto definition (LegacyDec Share) using valueCodec
		// If this succeeds, the data is already in the new format
		var newTD types.TrustDeposit
		newTD, err := valueCodec.Decode(value)
		if err == nil {
			// Data is already in new format, verify it's readable via collections.Map
			// and skip if it's already properly stored
			_, getErr := trustDepositMap.Get(ctx, newTD.Account)
			if getErr == nil {
				// Already properly stored in collections, skip
				skippedCount++
				continue
			}
			// Data decodes correctly but might not be in collections yet
			// Re-save it using collections.Map to ensure proper storage
			if setErr := trustDepositMap.Set(ctx, newTD.Account, newTD); setErr == nil {
				skippedCount++
				continue
			}
		}

		// Try to unmarshal with old proto definition (uint64 Share)
		oldTD, err := unmarshalOldTrustDeposit(value, cdc)
		if err != nil {
			logger.Info("Failed to unmarshal trust deposit (neither old nor new format), skipping", "key", key, "error", err)
			continue
		}

		// Convert Share from uint64 to LegacyDec
		newShare := math.LegacyNewDec(int64(oldTD.Share))

		// Create new TrustDeposit with LegacyDec Share using the actual types
		// Note: LastSlashed and LastRepaid are already *time.Time, so we can use them directly
		newTD = types.TrustDeposit{
			Account:        oldTD.Account,
			Share:          newShare, // Now LegacyDec
			Amount:         oldTD.Amount,
			Claimable:      oldTD.Claimable,
			SlashedDeposit: oldTD.SlashedDeposit,
			RepaidDeposit:  oldTD.RepaidDeposit,
			LastSlashed:    oldTD.LastSlashed,
			LastRepaid:     oldTD.LastRepaid,
			SlashCount:     oldTD.SlashCount,
			LastRepaidBy:   oldTD.LastRepaidBy,
		}

		// Save using collections.Map.Set() - this ensures correct encoding
		// and uses the same codec as normal operations
		if err := trustDepositMap.Set(ctx, oldTD.Account, newTD); err != nil {
			logger.Info("Failed to save migrated trust deposit", "account", oldTD.Account, "error", err)
			continue
		}

		// Delete the old key from raw store to avoid conflicts
		// The new data is now stored via collections.Map with proper encoding
		kvStore.Delete(key)

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
	// Note: LastSlashed and LastRepaid are already *time.Time, so we can use them directly
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
	Account        string     `protobuf:"bytes,1,opt,name=account,proto3" json:"account,omitempty"`
	Share          uint64     `protobuf:"varint,2,opt,name=share,proto3" json:"share,omitempty"` // OLD: uint64
	Amount         uint64     `protobuf:"varint,3,opt,name=amount,proto3" json:"amount,omitempty"`
	Claimable      uint64     `protobuf:"varint,4,opt,name=claimable,proto3" json:"claimable,omitempty"`
	SlashedDeposit uint64     `protobuf:"varint,5,opt,name=slashed_deposit,json=slashedDeposit,proto3" json:"slashed_deposit,omitempty"`
	RepaidDeposit  uint64     `protobuf:"varint,6,opt,name=repaid_deposit,json=repaidDeposit,proto3" json:"repaid_deposit,omitempty"`
	LastSlashed    *time.Time `protobuf:"bytes,7,opt,name=last_slashed,json=lastSlashed,proto3,stdtime" json:"last_slashed,omitempty"`
	LastRepaid     *time.Time `protobuf:"bytes,8,opt,name=last_repaid,json=lastRepaid,proto3,stdtime" json:"last_repaid,omitempty"`
	SlashCount     uint64     `protobuf:"varint,9,opt,name=slash_count,json=slashCount,proto3" json:"slash_count,omitempty"`
	LastRepaidBy   string     `protobuf:"bytes,10,opt,name=last_repaid_by,json=lastRepaidBy,proto3" json:"last_repaid_by,omitempty"`
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
