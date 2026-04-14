package keeper

import (
	"fmt"
	"regexp"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/tr/types"
)

func isValidLanguageTag(lang string) bool {
	// RFC1766 primary tag must be exactly 2 letters
	if len(lang) != 2 {
		return false
	}
	// Must be lowercase letters only
	match, _ := regexp.MatchString(`^[a-z]{2}$`, lang)
	return match
}

func (ms msgServer) createTrustRegistryEntries(ctx sdk.Context, msg *types.MsgCreateTrustRegistry, now time.Time) (types.TrustRegistry, error) {
	// Generate next ID for trust registry
	nextTrId, err := ms.Keeper.GetNextID(ctx, "tr")
	if err != nil {
		return types.TrustRegistry{}, fmt.Errorf("failed to generate trust registry ID: %w", err)
	}

	// Spec v4: MsgCreateTrustRegistry no longer bundles an initial governance
	// framework version/document; those are added via MsgAddGovernanceFrameworkDocument.
	tr := types.TrustRegistry{
		Id:            nextTrId,
		Did:           msg.Did,
		Corporation:   msg.Corporation, // Corporation is the controlling group account of the trust registry
		Created:       now,
		Modified:      now,
		Archived:      nil,
		Aka:           msg.Aka,
		ActiveVersion: 0,
		Language:      msg.Language,
	}

	return tr, nil
}

func (ms msgServer) persistEntries(ctx sdk.Context, tr types.TrustRegistry) error {
	if err := ms.TrustRegistry.Set(ctx, tr.Id, tr); err != nil {
		return fmt.Errorf("failed to persist TrustRegistry: %w", err)
	}

	// Store DID -> ID index
	if err := ms.TrustRegistryDIDIndex.Set(ctx, tr.Did, tr.Id); err != nil {
		return fmt.Errorf("failed to persist DID index: %w", err)
	}

	return nil
}
