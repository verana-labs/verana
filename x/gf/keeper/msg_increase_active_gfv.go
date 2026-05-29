package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	cerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

// IncreaseActiveGovernanceFrameworkVersion implements MOD-GF-MSG-2.
func (ms msgServer) IncreaseActiveGovernanceFrameworkVersion(goCtx context.Context, msg *types.MsgIncreaseActiveGovernanceFrameworkVersion) (*types.MsgIncreaseActiveGovernanceFrameworkVersionResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// AUTHZ-CHECK-1
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, sdk.MsgTypeURL(msg), ctx.BlockTime()); err != nil {
		return nil, err
	}

	sub, err := ms.resolveSubject(ctx, msg.Corporation, msg.EcosystemId)
	if err != nil {
		return nil, err
	}

	nextVersion := sub.activeVersion + 1
	// Lookup next GFV via secondary index.
	var gfvID uint64
	switch sub.kind {
	case subjectEcosystem:
		gfvID, err = ms.GFVersionByEcosystem.Get(ctx, collections.Join(sub.ecosystemID, nextVersion))
	case subjectCorporation:
		gfvID, err = ms.GFVersionByCorporation.Get(ctx, collections.Join(sub.corporationID, nextVersion))
	}
	if err != nil {
		return nil, cerrors.Wrapf(types.ErrNoActivatableVersion, "no GFV for next version %d", nextVersion)
	}
	gfv, err := ms.GFVersion.Get(ctx, gfvID)
	if err != nil {
		return nil, fmt.Errorf("fetch gfv %d: %w", gfvID, err)
	}

	// Spec MOD-GF-MSG-2-2-1: a document for subject.language MUST exist on this version.
	hasDefaultLang := false
	if err := ms.GFDocument.Walk(ctx, nil, func(_ uint64, doc types.GovernanceFrameworkDocument) (bool, error) {
		if doc.GfvId == gfv.Id && doc.Language == sub.language {
			hasDefaultLang = true
			return true, nil
		}
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("walk gfd: %w", err)
	}
	if !hasDefaultLang {
		return nil, types.ErrMissingDefaultLang
	}

	// Execute MOD-GF-MSG-2-3.
	now := ctx.BlockTime()
	gfv.ActiveSince = now
	if err := ms.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
		return nil, fmt.Errorf("persist gfv: %w", err)
	}

	switch sub.kind {
	case subjectEcosystem:
		if err := ms.ecosystemKeeper().SetEcosystemActiveVersion(ctx, sub.ecosystemID, nextVersion); err != nil {
			return nil, fmt.Errorf("update ecosystem active version: %w", err)
		}
	case subjectCorporation:
		if err := ms.corporationKeeper().SetActiveVersion(ctx, sub.corporationID, nextVersion); err != nil {
			return nil, fmt.Errorf("update corporation active version: %w", err)
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeIncreaseGFActive,
		sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
		sdk.NewAttribute(types.AttributeKeyEcosystemID, fmt.Sprintf("%d", msg.EcosystemId)),
		sdk.NewAttribute(types.AttributeKeyGFVersionID, fmt.Sprintf("%d", gfv.Id)),
		sdk.NewAttribute(types.AttributeKeyVersion, fmt.Sprintf("%d", nextVersion)),
	))

	return &types.MsgIncreaseActiveGovernanceFrameworkVersionResponse{}, nil
}
