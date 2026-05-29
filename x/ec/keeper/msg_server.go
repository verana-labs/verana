package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/ec/types"
)

type msgServer struct{ Keeper }

func NewMsgServerImpl(keeper Keeper) types.MsgServer { return &msgServer{Keeper: keeper} }

var _ types.MsgServer = msgServer{}

// CreateEcosystem implements MOD-ES-MSG-1.
//
// Order of operations (atomic — any error rolls back the whole tx):
//  1. ValidateBasic.
//  2. AUTHZ-CHECK: delegationKeeper.CheckOperatorAuthorization for the
//     (corporation, operator, msgTypeURL) tuple.
//  3. AUTHZ-CHECK-5: resolve msg.Corporation policy_address → co.id; reject
//     if no Corporation is registered for the signing account.
//  4. DID consistency precondition (MOD-ES-MSG-1-2-1): for every existing
//     Ecosystem entry with the same did, its corporation_id MUST equal
//     co.Id; else abort. (NOT a DID-uniqueness check — same did may be
//     shared across ecosystems controlled by the same Corporation.)
//  5. Allocate ec.id, persist Ecosystem + (did, corp_id) consistency index.
//  6. Call gfKeeper.CreateInitialGFVersionForEcosystem (seeds v1 GFV + GFD).
//  7. Emit create_ecosystem event.
func (ms msgServer) CreateEcosystem(goCtx context.Context, msg *types.MsgCreateEcosystem) (*types.MsgCreateEcosystemResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, sdk.MsgTypeURL(msg), now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	co, ok := ms.coKeeper.ResolveByPolicyAddress(ctx, msg.Corporation)
	if !ok {
		return nil, errors.Wrap(types.ErrCorporationNotRegistered, msg.Corporation)
	}

	if err := ms.assertDIDConsistent(ctx, msg.Did, co.Id, 0); err != nil {
		return nil, err
	}

	ecID, err := ms.GetNextID(ctx, "ec")
	if err != nil {
		return nil, err
	}
	ec := types.Ecosystem{
		Id:            ecID,
		Did:           msg.Did,
		CorporationId: co.Id,
		Created:       now,
		Modified:      now,
		Archived:      false,
		Language:      msg.Language,
		ActiveVersion: 1,
	}
	if err := ms.Ecosystem.Set(ctx, ec.Id, ec); err != nil {
		return nil, fmt.Errorf("persist ecosystem: %w", err)
	}
	if err := ms.EcosystemByDIDCorp.Set(ctx, collections.Join(msg.Did, co.Id), ec.Id); err != nil {
		return nil, fmt.Errorf("persist (did,corp) index: %w", err)
	}

	if err := ms.gfKeeper.CreateInitialGFVersionForEcosystem(ctx, ec.Id, msg.Language, msg.DocUrl, msg.DocDigestSri); err != nil {
		return nil, fmt.Errorf("seed initial GF version: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreateEcosystem,
		sdk.NewAttribute(types.AttributeKeyEcosystemID, fmt.Sprintf("%d", ec.Id)),
		sdk.NewAttribute(types.AttributeKeyCorporationID, fmt.Sprintf("%d", co.Id)),
		sdk.NewAttribute(types.AttributeKeyDID, msg.Did),
		sdk.NewAttribute(types.AttributeKeyLanguage, msg.Language),
	))

	return &types.MsgCreateEcosystemResponse{EcosystemId: ec.Id}, nil
}

// UpdateEcosystem implements MOD-ES-MSG-2.
//
// Spec MSG-2-3 literally reads "set ecosystem.did = did; ecosystem.modified =
// now". This implementation short-circuits on a no-op did rotation — same as
// MOD-CO's UpdateCorporation: AUTHZ-CHECK still runs so unauthorized
// operators can't even attempt a no-op write, but persist + event are
// skipped and `modified` is NOT bumped. Deliberate deviation from spec
// literal for ergonomics; matches MOD-CO precedent.
func (ms msgServer) UpdateEcosystem(goCtx context.Context, msg *types.MsgUpdateEcosystem) (*types.MsgUpdateEcosystemResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, sdk.MsgTypeURL(msg), now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	co, ok := ms.coKeeper.ResolveByPolicyAddress(ctx, msg.Corporation)
	if !ok {
		return nil, errors.Wrap(types.ErrCorporationNotRegistered, msg.Corporation)
	}

	ec, err := ms.Ecosystem.Get(ctx, msg.Id)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return nil, errors.Wrapf(types.ErrEcosystemNotFound, "id %d", msg.Id)
		}
		return nil, fmt.Errorf("get ecosystem: %w", err)
	}
	if ec.CorporationId != co.Id {
		return nil, errors.Wrapf(types.ErrUnauthorizedOperator, "ecosystem %d controlled by corporation %d, signer is corporation %d", ec.Id, ec.CorporationId, co.Id)
	}

	if ec.Did == msg.Did {
		return &types.MsgUpdateEcosystemResponse{}, nil
	}

	if err := ms.assertDIDConsistent(ctx, msg.Did, co.Id, ec.Id); err != nil {
		return nil, err
	}

	if err := ms.EcosystemByDIDCorp.Remove(ctx, collections.Join(ec.Did, co.Id)); err != nil {
		return nil, fmt.Errorf("remove old (did,corp) index: %w", err)
	}
	if err := ms.EcosystemByDIDCorp.Set(ctx, collections.Join(msg.Did, co.Id), ec.Id); err != nil {
		return nil, fmt.Errorf("set new (did,corp) index: %w", err)
	}

	ec.Did = msg.Did
	ec.Modified = now
	if err := ms.Ecosystem.Set(ctx, ec.Id, ec); err != nil {
		return nil, fmt.Errorf("persist ecosystem: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateEcosystem,
		sdk.NewAttribute(types.AttributeKeyEcosystemID, fmt.Sprintf("%d", ec.Id)),
		sdk.NewAttribute(types.AttributeKeyCorporationID, fmt.Sprintf("%d", co.Id)),
		sdk.NewAttribute(types.AttributeKeyDID, ec.Did),
	))

	return &types.MsgUpdateEcosystemResponse{}, nil
}

// ArchiveEcosystem implements MOD-ES-MSG-3.
func (ms msgServer) ArchiveEcosystem(goCtx context.Context, msg *types.MsgArchiveEcosystem) (*types.MsgArchiveEcosystemResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, sdk.MsgTypeURL(msg), now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	co, ok := ms.coKeeper.ResolveByPolicyAddress(ctx, msg.Corporation)
	if !ok {
		return nil, errors.Wrap(types.ErrCorporationNotRegistered, msg.Corporation)
	}

	ec, err := ms.Ecosystem.Get(ctx, msg.Id)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return nil, errors.Wrapf(types.ErrEcosystemNotFound, "id %d", msg.Id)
		}
		return nil, fmt.Errorf("get ecosystem: %w", err)
	}
	if ec.CorporationId != co.Id {
		return nil, errors.Wrapf(types.ErrUnauthorizedOperator, "ecosystem %d controlled by corporation %d, signer is corporation %d", ec.Id, ec.CorporationId, co.Id)
	}

	if ec.Archived == msg.Archive {
		return nil, errors.Wrapf(types.ErrAlreadyInTargetArchiveState, "ecosystem %d archived=%t", ec.Id, ec.Archived)
	}

	ec.Archived = msg.Archive
	ec.Modified = now
	if err := ms.Ecosystem.Set(ctx, ec.Id, ec); err != nil {
		return nil, fmt.Errorf("persist ecosystem: %w", err)
	}

	status := "archived"
	if !msg.Archive {
		status = "unarchived"
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeArchiveEcosystem,
		sdk.NewAttribute(types.AttributeKeyEcosystemID, fmt.Sprintf("%d", ec.Id)),
		sdk.NewAttribute(types.AttributeKeyCorporationID, fmt.Sprintf("%d", co.Id)),
		sdk.NewAttribute(types.AttributeKeyArchiveStatus, status),
	))

	return &types.MsgArchiveEcosystemResponse{}, nil
}

// assertDIDConsistent enforces the per-Ecosystem (did, corporation_id)
// consistency invariant by iterating the (did, *) prefix in
// EcosystemByDIDCorp. selfID, if non-zero, is the id of the row being
// updated and is excluded from the check (so a no-op rotation to a did the
// caller already owns does not trip the invariant). On any conflicting
// owner found, returns ErrDIDOwnershipConflict.
func (k Keeper) assertDIDConsistent(ctx context.Context, did string, ownerCorpID uint64, selfID uint64) error {
	rng := collections.NewPrefixedPairRange[string, uint64](did)
	iter, err := k.EcosystemByDIDCorp.Iterate(ctx, rng)
	if err != nil {
		return fmt.Errorf("iterate (did,corp) index: %w", err)
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return fmt.Errorf("iter key: %w", err)
		}
		ecID, err := iter.Value()
		if err != nil {
			return fmt.Errorf("iter value: %w", err)
		}
		if selfID != 0 && ecID == selfID {
			continue
		}
		if key.K2() != ownerCorpID {
			return errors.Wrapf(types.ErrDIDOwnershipConflict, "did %q controlled by corporation %d", did, key.K2())
		}
	}
	return nil
}
