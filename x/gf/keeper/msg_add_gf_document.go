package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	cerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

// subjectKind is an internal tag for which owner the request targets.
type subjectKind int

const (
	subjectEcosystem subjectKind = iota
	subjectCorporation
)

// resolvedSubject holds the validated owner of a GF operation.
type resolvedSubject struct {
	kind          subjectKind
	ecosystemID   uint64
	corporationID uint64
	language      string
	activeVersion uint32
}

// resolveSubject implements the spec's "Define subject as ..." block for
// MOD-GF-MSG-1-2-1 and MOD-GF-MSG-2-2-1, on top of AUTHZ-CHECK-5 which
// resolves the signing account → Corporation by policy_address → co.id.
func (k Keeper) resolveSubject(ctx context.Context, signingCorp string, ecosystemID uint64) (resolvedSubject, error) {
	// AUTHZ-CHECK-5 surface: resolve the signing corporation account.
	coView, ok := k.corporationKeeper().ResolveByPolicyAddress(ctx, signingCorp)
	if !ok {
		return resolvedSubject{}, cerrors.Wrapf(types.ErrSubjectNotFound, "no Corporation registered for signing account %s", signingCorp)
	}

	if ecosystemID != 0 {
		eco, ok := k.ecosystemKeeper().GetEcosystemView(ctx, ecosystemID)
		if !ok {
			return resolvedSubject{}, cerrors.Wrapf(types.ErrSubjectNotFound, "ecosystem %d", ecosystemID)
		}
		if eco.CorporationID != coView.Id {
			return resolvedSubject{}, types.ErrSubjectNotControlled
		}
		return resolvedSubject{
			kind:          subjectEcosystem,
			ecosystemID:   eco.Id,
			language:      eco.Language,
			activeVersion: eco.ActiveVersion,
		}, nil
	}

	return resolvedSubject{
		kind:          subjectCorporation,
		corporationID: coView.Id,
		language:      coView.Language,
		activeVersion: coView.ActiveVersion,
	}, nil
}

// maxVersionFor returns the highest known GFV.version for the subject, or 0 if none.
// Also returns whether a GFV with `targetVersion` already exists and its id.
func (k Keeper) maxVersionFor(ctx context.Context, sub resolvedSubject, targetVersion uint32) (maxV uint32, hasTarget bool, gfvID uint64, err error) {
	switch sub.kind {
	case subjectEcosystem:
		iter, e := k.GFVersionByEcosystem.Iterate(ctx, collections.NewPrefixedPairRange[uint64, uint32](sub.ecosystemID))
		if e != nil {
			err = e
			return
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			key, e := iter.Key()
			if e != nil {
				err = e
				return
			}
			v := key.K2()
			if v > maxV {
				maxV = v
			}
			if v == targetVersion {
				hasTarget = true
				id, e := iter.Value()
				if e != nil {
					err = e
					return
				}
				gfvID = id
			}
		}
	case subjectCorporation:
		iter, e := k.GFVersionByCorporation.Iterate(ctx, collections.NewPrefixedPairRange[uint64, uint32](sub.corporationID))
		if e != nil {
			err = e
			return
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			key, e := iter.Key()
			if e != nil {
				err = e
				return
			}
			v := key.K2()
			if v > maxV {
				maxV = v
			}
			if v == targetVersion {
				hasTarget = true
				id, e := iter.Value()
				if e != nil {
					err = e
					return
				}
				gfvID = id
			}
		}
	}
	return
}

// AddGovernanceFrameworkDocument implements MOD-GF-MSG-1.
func (ms msgServer) AddGovernanceFrameworkDocument(goCtx context.Context, msg *types.MsgAddGovernanceFrameworkDocument) (*types.MsgAddGovernanceFrameworkDocumentResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// AUTHZ-CHECK-1
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, sdk.MsgTypeURL(msg), ctx.BlockTime()); err != nil {
		return nil, err
	}

	// Resolve subject (Ecosystem or Corporation).
	sub, err := ms.resolveSubject(ctx, msg.Corporation, msg.EcosystemId)
	if err != nil {
		return nil, err
	}

	// Version checks per MOD-GF-MSG-1-2-1.
	maxV, hasTarget, existingGfvID, err := ms.maxVersionFor(ctx, sub, msg.Version)
	if err != nil {
		return nil, err
	}
	if !hasTarget && msg.Version != maxV+1 {
		return nil, cerrors.Wrapf(types.ErrInvalidVersion, "version must be %d", maxV+1)
	}
	if msg.Version <= sub.activeVersion {
		return nil, cerrors.Wrapf(types.ErrInvalidVersion, "must be greater than active_version %d", sub.activeVersion)
	}

	// Execute per MOD-GF-MSG-1-3.
	var gfv types.GovernanceFrameworkVersion
	if hasTarget {
		gfv, err = ms.GFVersion.Get(ctx, existingGfvID)
		if err != nil {
			return nil, fmt.Errorf("fetch gfv %d: %w", existingGfvID, err)
		}
	} else {
		nextID, err := ms.GetNextID(ctx, "gfv")
		if err != nil {
			return nil, err
		}
		gfv = types.GovernanceFrameworkVersion{
			Id:      nextID,
			Created: ctx.BlockTime(),
			Version: msg.Version,
		}
		if sub.kind == subjectEcosystem {
			gfv.EcosystemId = sub.ecosystemID
		} else {
			gfv.CorporationId = sub.corporationID
		}
		if err := ms.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
			return nil, fmt.Errorf("persist gfv: %w", err)
		}
		// Maintain secondary index.
		if sub.kind == subjectEcosystem {
			if err := ms.GFVersionByEcosystem.Set(ctx, collections.Join(sub.ecosystemID, msg.Version), gfv.Id); err != nil {
				return nil, fmt.Errorf("persist gfv eco index: %w", err)
			}
		} else {
			if err := ms.GFVersionByCorporation.Set(ctx, collections.Join(sub.corporationID, msg.Version), gfv.Id); err != nil {
				return nil, fmt.Errorf("persist gfv corp index: %w", err)
			}
		}
	}

	// Upsert the document for (gfv, language).
	var existingGFD types.GovernanceFrameworkDocument
	hasExisting := false
	if err := ms.GFDocument.Walk(ctx, nil, func(_ uint64, doc types.GovernanceFrameworkDocument) (bool, error) {
		if doc.GfvId == gfv.Id && doc.Language == msg.DocLanguage {
			existingGFD = doc
			hasExisting = true
			return true, nil
		}
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("walk gfd: %w", err)
	}

	var gfd types.GovernanceFrameworkDocument
	if hasExisting {
		gfd = existingGFD
		gfd.Url = msg.DocUrl
		gfd.DigestSri = msg.DocDigestSri
	} else {
		nextID, err := ms.GetNextID(ctx, "gfd")
		if err != nil {
			return nil, err
		}
		gfd = types.GovernanceFrameworkDocument{
			Id:        nextID,
			GfvId:     gfv.Id,
			Created:   ctx.BlockTime(),
			Language:  msg.DocLanguage,
			Url:       msg.DocUrl,
			DigestSri: msg.DocDigestSri,
		}
	}
	if err := ms.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return nil, fmt.Errorf("persist gfd: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeAddGFDocument,
		sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
		sdk.NewAttribute(types.AttributeKeyEcosystemID, fmt.Sprintf("%d", msg.EcosystemId)),
		sdk.NewAttribute(types.AttributeKeyGFVersionID, fmt.Sprintf("%d", gfv.Id)),
		sdk.NewAttribute(types.AttributeKeyGFDocID, fmt.Sprintf("%d", gfd.Id)),
		sdk.NewAttribute(types.AttributeKeyVersion, fmt.Sprintf("%d", msg.Version)),
		sdk.NewAttribute(types.AttributeKeyLanguage, msg.DocLanguage),
	))

	return &types.MsgAddGovernanceFrameworkDocumentResponse{}, nil
}
