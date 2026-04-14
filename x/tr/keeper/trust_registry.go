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

func (ms msgServer) createTrustRegistryEntries(ctx sdk.Context, msg *types.MsgCreateTrustRegistry, now time.Time) (types.TrustRegistry, types.GovernanceFrameworkVersion, error) {
	nextTrId, err := ms.Keeper.GetNextID(ctx, "tr")
	if err != nil {
		return types.TrustRegistry{}, types.GovernanceFrameworkVersion{}, fmt.Errorf("failed to generate trust registry ID: %w", err)
	}

	// Spec v4: MsgCreateTrustRegistry no longer bundles an initial governance
	// framework document, but an empty v1 is seeded internally so active_version=1
	// and subsequent MsgAddGovernanceFrameworkDocument calls can attach documents.
	tr := types.TrustRegistry{
		Id:            nextTrId,
		Did:           msg.Did,
		Corporation:   msg.Corporation, // Corporation is the controlling group account of the trust registry
		Created:       now,
		Modified:      now,
		Archived:      nil,
		Aka:           msg.Aka,
		ActiveVersion: 1,
		Language:      msg.Language,
	}

	nextGfvId, err := ms.Keeper.GetNextID(ctx, "gfv")
	if err != nil {
		return types.TrustRegistry{}, types.GovernanceFrameworkVersion{}, fmt.Errorf("failed to generate governance framework version ID: %w", err)
	}

	gfv := types.GovernanceFrameworkVersion{
		Id:          nextGfvId,
		TrId:        tr.Id,
		Created:     now,
		Version:     1,
		ActiveSince: now,
	}

	return tr, gfv, nil
}

func (ms msgServer) persistEntries(ctx sdk.Context, tr types.TrustRegistry, gfv types.GovernanceFrameworkVersion) error {
	if err := ms.TrustRegistry.Set(ctx, tr.Id, tr); err != nil {
		return fmt.Errorf("failed to persist TrustRegistry: %w", err)
	}

	if err := ms.TrustRegistryDIDIndex.Set(ctx, tr.Did, tr.Id); err != nil {
		return fmt.Errorf("failed to persist DID index: %w", err)
	}

	if err := ms.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
		return fmt.Errorf("failed to persist GovernanceFrameworkVersion: %w", err)
	}

	return nil
}
