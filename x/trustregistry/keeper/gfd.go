package keeper

import (
	"errors"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana-blockchain/x/trustregistry/types"
)

func (ms msgServer) validateAddGovernanceFrameworkDocumentParams(ctx sdk.Context, msg *types.MsgAddGovernanceFrameworkDocument) error {
	// Direct lookup of trust registry by ID
	tr, err := ms.TrustRegistry.Get(ctx, msg.Id)
	if err != nil {
		return fmt.Errorf("trust registry with ID %d does not exist: %w", msg.Id, err)
	}

	// Check controller
	if tr.Controller != msg.Creator {
		return errors.New("creator is not the controller of the trust registry")
	}

	// Check version validity
	var maxVersion int32
	var hasVersion bool
	err = ms.GFVersion.Walk(ctx, nil, func(id uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
		if gfv.TrId == msg.Id {
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

	// Validate version according to spec
	if !hasVersion && msg.Version != maxVersion+1 {
		return fmt.Errorf("invalid version: must be %d or %d", maxVersion, maxVersion+1)
	}

	if msg.Version <= tr.ActiveVersion {
		return fmt.Errorf("version must be greater than the active version %d", tr.ActiveVersion)
	}

	// Validate language tag
	if !isValidLanguageTag(msg.DocLanguage) {
		return errors.New("invalid language tag (must conform to rfc1766)")
	}

	return nil
}

func (ms msgServer) executeAddGovernanceFrameworkDocument(ctx sdk.Context, msg *types.MsgAddGovernanceFrameworkDocument) error {
	// Find or create governance framework version
	var gfv types.GovernanceFrameworkVersion
	maxVersion := int32(0)
	err := ms.GFVersion.Walk(ctx, nil, func(key uint64, version types.GovernanceFrameworkVersion) (bool, error) {
		if version.TrId == msg.Id {
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
			TrId:        msg.Id,
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
		Id:       nextGfdId,
		GfvId:    gfv.Id,
		Created:  ctx.BlockTime(),
		Language: msg.DocLanguage,
		Url:      msg.DocUrl,
		Hash:     msg.DocHash,
	}

	if err := ms.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return fmt.Errorf("failed to persist governance framework document: %w", err)
	}

	return nil
}
