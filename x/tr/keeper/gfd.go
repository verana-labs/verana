package keeper

import (
	"errors"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/tr/types"
)

func (ms msgServer) validateAddGovernanceFrameworkDocumentParams(ctx sdk.Context, msg *types.MsgAddGovernanceFrameworkDocument) error {
	// Direct lookup of trust registry by ID
	tr, err := ms.TrustRegistry.Get(ctx, msg.TrId)
	if err != nil {
		return fmt.Errorf("trust registry with ID %d does not exist: %w", msg.TrId, err)
	}

	// Check corporation - corporation must match the trust registry corporation
	if tr.Corporation != msg.Corporation {
		return errors.New("corporation is not the controller of the trust registry")
	}

	// Check version validity
	var maxVersion int32
	var hasVersion bool
	err = ms.GFVersion.Walk(ctx, nil, func(id uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
		if gfv.TrId == msg.TrId {
			if gfv.Version == msg.Version {
				hasVersion = true
			}
			if gfv.Version > maxVersion {
				maxVersion = gfv.Version
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("error checking versions: %w", err)
	}

	// Spec v4: version must either refer to an existing GF version OR be the next
	// sequential version (maxVersion+1). Adding documents to older (strictly
	// inactive) versions is rejected, but documents MAY be added to the current
	// active version (allows language variants on a fresh/initial version).
	nextVersion := maxVersion + 1

	if !hasVersion && msg.Version != nextVersion {
		return fmt.Errorf("invalid version: must be %d", nextVersion)
	}

	if msg.Version < tr.ActiveVersion {
		return fmt.Errorf("invalid version: must be at least %d (current active version)", tr.ActiveVersion)
	}

	// Validate language tag
	if !isValidLanguageTag(msg.Language) {
		return errors.New("invalid language tag (must conform to rfc1766)")
	}

	return nil
}

func (ms msgServer) executeAddGovernanceFrameworkDocument(ctx sdk.Context, msg *types.MsgAddGovernanceFrameworkDocument) error {
	// Find or create governance framework version
	var gfv types.GovernanceFrameworkVersion
	maxVersion := int32(0)
	err := ms.GFVersion.Walk(ctx, nil, func(key uint64, version types.GovernanceFrameworkVersion) (bool, error) {
		if version.TrId == msg.TrId {
			if version.Version > maxVersion {
				maxVersion = version.Version
			}
			if version.Version == msg.Version {
				gfv = version
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk governance framework versions: %w", err)
	}

	// Create new version if needed
	if gfv.Id == 0 {
		nextGfvId, err := ms.GetNextID(ctx, "gfv")
		if err != nil {
			return fmt.Errorf("failed to generate governance framework version ID: %w", err)
		}
		gfv = types.GovernanceFrameworkVersion{
			Id:          nextGfvId,
			TrId:        msg.TrId,
			Created:     ctx.BlockTime(),
			Version:     msg.Version,
			ActiveSince: time.Time{}, // Zero time as per spec - not active yet
		}
		if err := ms.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
			return fmt.Errorf("failed to persist governance framework version: %w", err)
		}
	}

	// Create document
	nextGfdId, err := ms.GetNextID(ctx, "gfd")
	if err != nil {
		return fmt.Errorf("failed to generate governance framework document ID: %w", err)
	}

	gfd := types.GovernanceFrameworkDocument{
		Id:        nextGfdId,
		GfvId:     gfv.Id,
		Created:   ctx.BlockTime(),
		Language:  msg.Language,
		Url:       msg.Url,
		DigestSri: msg.DigestSri,
	}

	if err := ms.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return fmt.Errorf("failed to persist governance framework document: %w", err)
	}

	return nil
}
