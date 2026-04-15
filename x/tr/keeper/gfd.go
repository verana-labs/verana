package keeper

import (
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/collections"
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

	// Use secondary index to find the max version for this TR without a full table scan.
	// Iterate the prefix (trId, *) in reverse to get the highest version first.
	var maxVersion int32
	var hasVersion bool
	iter, err := ms.GFVersionByTR.Iterate(ctx, collections.NewPrefixedPairRange[uint64, int32](msg.TrId))
	if err != nil {
		return fmt.Errorf("error iterating GFVersionByTR index: %w", err)
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return fmt.Errorf("error reading GFVersionByTR key: %w", err)
		}
		v := key.K2()
		if v > maxVersion {
			maxVersion = v
		}
		if v == msg.Version {
			hasVersion = true
		}
	}

	// [MOD-TR-MSG-2-2-1] Spec draft 13: version MUST be greater than tr.active_version.
	// Documents can only be added to in-progress (future) versions; the active version
	// is immutable. This also implies the version either refers to an existing
	// in-progress GF version OR is the next sequential version (maxVersion+1).
	nextVersion := maxVersion + 1

	if !hasVersion && msg.Version != nextVersion {
		return fmt.Errorf("invalid version: must be %d", nextVersion)
	}

	if msg.Version <= tr.ActiveVersion {
		return fmt.Errorf("invalid version: must be greater than %d (current active version)", tr.ActiveVersion)
	}

	// Validate language tag
	if !types.IsValidBCP47(msg.Language) {
		return errors.New("invalid language tag (must be a valid BCP 47 tag)")
	}

	return nil
}

func (ms msgServer) executeAddGovernanceFrameworkDocument(ctx sdk.Context, msg *types.MsgAddGovernanceFrameworkDocument) error {
	// Use secondary index to find the GFV for this (tr_id, version) pair — O(1) lookup.
	gfvId, err := ms.GFVersionByTR.Get(ctx, collections.Join(msg.TrId, msg.Version))
	var gfv types.GovernanceFrameworkVersion
	if err == nil {
		// Found existing GFV
		gfv, err = ms.GFVersion.Get(ctx, gfvId)
		if err != nil {
			return fmt.Errorf("failed to fetch governance framework version %d: %w", gfvId, err)
		}
	} else {
		// Create new version
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
		// Maintain secondary index
		if err := ms.GFVersionByTR.Set(ctx, collections.Join(gfv.TrId, gfv.Version), gfv.Id); err != nil {
			return fmt.Errorf("failed to persist GFVersionByTR index: %w", err)
		}
	}

	// [MOD-TR-MSG-2-1 / 2-3] Spec: if a document already exists for this language
	// in this GF version, REPLACE the existing entry (in-place); otherwise create new.
	var existingGfd types.GovernanceFrameworkDocument
	var hasExisting bool
	if err := ms.GFDocument.Walk(ctx, nil, func(_ uint64, doc types.GovernanceFrameworkDocument) (bool, error) {
		if doc.GfvId == gfv.Id && doc.Language == msg.Language {
			existingGfd = doc
			hasExisting = true
			return true, nil
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("failed to walk governance framework documents: %w", err)
	}

	var gfd types.GovernanceFrameworkDocument
	if hasExisting {
		gfd = existingGfd
		gfd.Url = msg.Url
		gfd.DigestSri = msg.DigestSri
	} else {
		nextGfdId, err := ms.GetNextID(ctx, "gfd")
		if err != nil {
			return fmt.Errorf("failed to generate governance framework document ID: %w", err)
		}
		gfd = types.GovernanceFrameworkDocument{
			Id:        nextGfdId,
			GfvId:     gfv.Id,
			Created:   ctx.BlockTime(),
			Language:  msg.Language,
			Url:       msg.Url,
			DigestSri: msg.DigestSri,
		}
	}

	if err := ms.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return fmt.Errorf("failed to persist governance framework document: %w", err)
	}

	return nil
}
