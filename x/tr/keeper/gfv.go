package keeper

import (
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/tr/types"
)

func (ms msgServer) validateIncreaseActiveGovernanceFrameworkVersionParams(ctx sdk.Context, msg *types.MsgIncreaseActiveGovernanceFrameworkVersion) error {
	// Direct lookup by ID
	tr, err := ms.TrustRegistry.Get(ctx, msg.TrId)
	if err != nil {
		return fmt.Errorf("trust registry with ID %d does not exist: %w", msg.TrId, err)
	}

	if tr.Corporation != msg.Corporation {
		return errors.New("corporation is not the controller of the trust registry")
	}

	nextVersion := tr.ActiveVersion + 1

	// Use secondary index for O(1) lookup instead of full table scan.
	gfvId, err := ms.GFVersionByTR.Get(ctx, collections.Join(msg.TrId, nextVersion))
	if err != nil {
		return fmt.Errorf("no governance framework version found for version %d", nextVersion)
	}

	gfv, err := ms.GFVersion.Get(ctx, gfvId)
	if err != nil {
		return fmt.Errorf("failed to fetch governance framework version %d: %w", gfvId, err)
	}

	// Check for document in trust registry's language
	var hasDefaultLanguageDoc bool
	err = ms.GFDocument.Walk(ctx, nil, func(id uint64, gfd types.GovernanceFrameworkDocument) (bool, error) {
		if gfd.GfvId == gfv.Id && gfd.Language == tr.Language {
			hasDefaultLanguageDoc = true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("error checking documents: %w", err)
	}
	if !hasDefaultLanguageDoc {
		return errors.New("no document found for the default language of this version")
	}

	return nil
}

func (ms msgServer) executeIncreaseActiveGovernanceFrameworkVersion(ctx sdk.Context, msg *types.MsgIncreaseActiveGovernanceFrameworkVersion) error {
	// Direct lookup of trust registry by ID
	tr, err := ms.TrustRegistry.Get(ctx, msg.TrId)
	if err != nil {
		return fmt.Errorf("error finding trust registry: %w", err)
	}

	nextVersion := tr.ActiveVersion + 1

	// Use secondary index for O(1) lookup.
	gfvId, err := ms.GFVersionByTR.Get(ctx, collections.Join(msg.TrId, nextVersion))
	if err != nil {
		return fmt.Errorf("next version not found")
	}

	nextGfv, err := ms.GFVersion.Get(ctx, gfvId)
	if err != nil {
		return fmt.Errorf("failed to fetch governance framework version %d: %w", gfvId, err)
	}

	// Update version
	now := ctx.BlockTime()
	tr.ActiveVersion = nextVersion
	tr.Modified = now
	nextGfv.ActiveSince = now

	// Persist changes
	if err := ms.TrustRegistry.Set(ctx, tr.Id, tr); err != nil {
		return fmt.Errorf("failed to update trust registry: %w", err)
	}
	if err := ms.GFVersion.Set(ctx, nextGfv.Id, nextGfv); err != nil {
		return fmt.Errorf("failed to update governance framework version: %w", err)
	}

	return nil
}
