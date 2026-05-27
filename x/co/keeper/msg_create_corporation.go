package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/group"

	"github.com/verana-labs/verana/x/co/types"
)

// CreateCorporation implements MOD-CO-MSG-1.
//
// Order of operations (atomic — any error rolls back the whole tx):
//  1. ValidateBasic (signer, members, decision_policy, did, language, urls).
//  2. DID uniqueness precondition: msg.did MUST NOT collide with any existing
//     Corporation. Checked before x/group work so a duplicate-DID submission
//     fails fast without wasting group-creation cycles (spec: precondition
//     checks precede execution).
//  3. Allocate x/group Group + GroupPolicy in one call (CreateGroupWithPolicy
//     with group_policy_as_admin=true). The returned group_policy_address is
//     the on-chain identity of the Corporation.
//  4. policy_address uniqueness gate (defence-in-depth: x/group returns a
//     deterministic address, so this should never fire in practice).
//  5. Allocate co.id, persist Corporation + reverse indexes.
//  6. Call gfKeeper.CreateInitialGFVersionForCorporation to seed v1 GF.
//  7. Emit create_corporation event.
func (ms msgServer) CreateCorporation(goCtx context.Context, msg *types.MsgCreateCorporation) (*types.MsgCreateCorporationResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// DID uniqueness precondition — runs BEFORE x/group call.
	if has, err := ms.CorporationByDID.Has(ctx, msg.Did); err != nil {
		return nil, fmt.Errorf("check did uniqueness: %w", err)
	} else if has {
		return nil, errors.Wrap(types.ErrDIDAlreadyExists, msg.Did)
	}

	groupMembers := make([]group.MemberRequest, len(msg.Members))
	for i, m := range msg.Members {
		groupMembers[i] = group.MemberRequest{
			Address:  m.Address,
			Weight:   m.Weight,
			Metadata: m.Metadata,
		}
	}

	groupReq := &group.MsgCreateGroupWithPolicy{
		Admin:               msg.Signer,
		Members:             groupMembers,
		GroupMetadata:       msg.GroupMetadata,
		GroupPolicyMetadata: msg.GroupPolicyMetadata,
		GroupPolicyAsAdmin:  true,
		DecisionPolicy:      msg.DecisionPolicy,
	}
	groupResp, err := ms.groupKeeper.CreateGroupWithPolicy(ctx, groupReq)
	if err != nil {
		return nil, fmt.Errorf("create group with policy: %w", err)
	}
	policyAddr := groupResp.GroupPolicyAddress

	if has, err := ms.CorporationByPolicyAddr.Has(ctx, policyAddr); err != nil {
		return nil, fmt.Errorf("check policy_address uniqueness: %w", err)
	} else if has {
		return nil, errors.Wrap(types.ErrPolicyAddressAlreadyBound, policyAddr)
	}

	coID, err := ms.GetNextID(ctx, "co")
	if err != nil {
		return nil, err
	}
	now := ctx.BlockTime()
	co := types.Corporation{
		Id:            coID,
		PolicyAddress: policyAddr,
		Did:           msg.Did,
		Created:       now,
		Modified:      now,
		Language:      msg.Language,
		ActiveVersion: 1,
	}
	if err := ms.Corporation.Set(ctx, co.Id, co); err != nil {
		return nil, fmt.Errorf("persist corporation: %w", err)
	}
	if err := ms.CorporationByPolicyAddr.Set(ctx, policyAddr, co.Id); err != nil {
		return nil, fmt.Errorf("persist policy_address index: %w", err)
	}
	if err := ms.CorporationByDID.Set(ctx, msg.Did, co.Id); err != nil {
		return nil, fmt.Errorf("persist did index: %w", err)
	}

	if err := ms.gfKeeper.CreateInitialGFVersionForCorporation(ctx, co.Id, msg.Language, msg.DocUrl, msg.DocDigestSri); err != nil {
		return nil, fmt.Errorf("seed initial GF version: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreateCorporation,
		sdk.NewAttribute(types.AttributeKeyCorporationID, fmt.Sprintf("%d", co.Id)),
		sdk.NewAttribute(types.AttributeKeyPolicyAddress, policyAddr),
		sdk.NewAttribute(types.AttributeKeyDID, msg.Did),
	))

	return &types.MsgCreateCorporationResponse{
		CorporationId: co.Id,
		PolicyAddress: policyAddr,
	}, nil
}
