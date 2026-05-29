package keeper

import (
	"context"
	"fmt"
	"sort"

	"cosmossdk.io/collections"
	cerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

// CreateInitialGFVersionForCorporation seeds version 1 of the Corporation's
// Governance Framework: one GovernanceFrameworkVersion + one
// GovernanceFrameworkDocument (in the Corporation's primary `language`).
//
// Called by MOD-CO MSG-1 (CreateCorporation) immediately after the Corporation
// row is persisted. There is no on-chain signer at this layer (keeper-to-keeper
// call), so AUTHZ is skipped — the parent MOD-CO MSG-1 has already validated
// the inputs via MsgCreateCorporation.ValidateBasic.
//
// Idempotency: aborts if (corpID, v=1) is already indexed. That can only
// happen on a buggy double-call from MOD-CO; it is NOT a legitimate retry path.
func (k Keeper) CreateInitialGFVersionForCorporation(
	ctx context.Context,
	corpID uint64,
	language, docURL, docDigestSRI string,
) error {
	if corpID == 0 {
		return cerrors.Wrap(types.ErrInvalidSubject, "corporation_id must be > 0")
	}
	if has, err := k.GFVersionByCorporation.Has(ctx, collections.Join(corpID, uint32(1))); err != nil {
		return fmt.Errorf("check existing gfv: %w", err)
	} else if has {
		return cerrors.Wrapf(types.ErrInvalidVersion, "initial GFV for corporation %d already exists", corpID)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	gfvID, err := k.GetNextID(sdkCtx, "gfv")
	if err != nil {
		return err
	}
	gfv := types.GovernanceFrameworkVersion{
		Id:            gfvID,
		CorporationId: corpID,
		Created:       now,
		Version:       1,
		ActiveSince:   now, // MOD-CO MSG-1 seed: v1 is born active per spec.
	}
	if err := k.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
		return fmt.Errorf("persist gfv: %w", err)
	}
	if err := k.GFVersionByCorporation.Set(ctx, collections.Join(corpID, uint32(1)), gfv.Id); err != nil {
		return fmt.Errorf("persist gfv corp index: %w", err)
	}

	gfdID, err := k.GetNextID(sdkCtx, "gfd")
	if err != nil {
		return err
	}
	gfd := types.GovernanceFrameworkDocument{
		Id:        gfdID,
		GfvId:     gfv.Id,
		Created:   now,
		Language:  language,
		Url:       docURL,
		DigestSri: docDigestSRI,
	}
	if err := k.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return fmt.Errorf("persist gfd: %w", err)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeAddGFDocument,
		sdk.NewAttribute(types.AttributeKeyCorporation, fmt.Sprintf("%d", corpID)),
		sdk.NewAttribute(types.AttributeKeyGFVersionID, fmt.Sprintf("%d", gfv.Id)),
		sdk.NewAttribute(types.AttributeKeyGFDocID, fmt.Sprintf("%d", gfd.Id)),
		sdk.NewAttribute(types.AttributeKeyVersion, "1"),
		sdk.NewAttribute(types.AttributeKeyLanguage, language),
	))

	return nil
}

// CreateInitialGFVersionForEcosystem seeds version 1 of the Ecosystem's
// Governance Framework: one GovernanceFrameworkVersion + one
// GovernanceFrameworkDocument (in the Ecosystem's primary `language`).
//
// Called by MOD-ES MSG-1 (CreateEcosystem) immediately after the Ecosystem
// row is persisted. Mirror of CreateInitialGFVersionForCorporation; the
// resulting GFV is the ecosystem-owned half of the XOR (EcosystemId set,
// CorporationId zero), enforced by x/gf genesis at-rest invariants.
//
// AUTHZ is skipped (keeper-to-keeper). Idempotency: aborts if (ecID, v=1) is
// already indexed.
func (k Keeper) CreateInitialGFVersionForEcosystem(
	ctx context.Context,
	ecID uint64,
	language, docURL, docDigestSRI string,
) error {
	if ecID == 0 {
		return cerrors.Wrap(types.ErrInvalidSubject, "ecosystem_id must be > 0")
	}
	if has, err := k.GFVersionByEcosystem.Has(ctx, collections.Join(ecID, uint32(1))); err != nil {
		return fmt.Errorf("check existing gfv: %w", err)
	} else if has {
		return cerrors.Wrapf(types.ErrInvalidVersion, "initial GFV for ecosystem %d already exists", ecID)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	gfvID, err := k.GetNextID(sdkCtx, "gfv")
	if err != nil {
		return err
	}
	gfv := types.GovernanceFrameworkVersion{
		Id:          gfvID,
		EcosystemId: ecID,
		Created:     now,
		Version:     1,
		ActiveSince: now, // MOD-ES MSG-1 seed: v1 is born active per spec.
	}
	if err := k.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
		return fmt.Errorf("persist gfv: %w", err)
	}
	if err := k.GFVersionByEcosystem.Set(ctx, collections.Join(ecID, uint32(1)), gfv.Id); err != nil {
		return fmt.Errorf("persist gfv eco index: %w", err)
	}

	gfdID, err := k.GetNextID(sdkCtx, "gfd")
	if err != nil {
		return err
	}
	gfd := types.GovernanceFrameworkDocument{
		Id:        gfdID,
		GfvId:     gfv.Id,
		Created:   now,
		Language:  language,
		Url:       docURL,
		DigestSri: docDigestSRI,
	}
	if err := k.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return fmt.Errorf("persist gfd: %w", err)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeAddGFDocument,
		sdk.NewAttribute(types.AttributeKeyEcosystemID, fmt.Sprintf("%d", ecID)),
		sdk.NewAttribute(types.AttributeKeyGFVersionID, fmt.Sprintf("%d", gfv.Id)),
		sdk.NewAttribute(types.AttributeKeyGFDocID, fmt.Sprintf("%d", gfd.Id)),
		sdk.NewAttribute(types.AttributeKeyVersion, "1"),
		sdk.NewAttribute(types.AttributeKeyLanguage, language),
	))

	return nil
}

// ListVersionsByCorporation returns the Corporation's GovernanceFrameworkVersion
// list (each enriched with documents) for the CorporationWithGF response shape
// in MOD-CO queries.
//
// `activeVersion` is the Corporation's current active_version, supplied by
// MOD-CO (which owns that field). When `activeOnly` is true only that entry is
// returned. `preferredLang` filters documents per the same rules as
// (querier).collectDocs.
func (k Keeper) ListVersionsByCorporation(
	ctx context.Context,
	corpID uint64,
	activeVersion uint32,
	activeOnly bool,
	preferredLang string,
) ([]types.GovernanceFrameworkVersionWithDocs, error) {
	if corpID == 0 {
		return nil, cerrors.Wrap(types.ErrInvalidSubject, "corporation_id must be > 0")
	}

	var gfvIDs []uint64
	iter, err := k.GFVersionByCorporation.Iterate(ctx, collections.NewPrefixedPairRange[uint64, uint32](corpID))
	if err != nil {
		return nil, fmt.Errorf("iterate gfv index: %w", err)
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		id, err := iter.Value()
		if err != nil {
			return nil, fmt.Errorf("iter value: %w", err)
		}
		gfvIDs = append(gfvIDs, id)
	}

	out := make([]types.GovernanceFrameworkVersionWithDocs, 0, len(gfvIDs))
	q := querier{Keeper: k}
	for _, id := range gfvIDs {
		gfv, err := k.GFVersion.Get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("fetch gfv %d: %w", id, err)
		}
		if activeOnly && gfv.Version != activeVersion {
			continue
		}
		docs, err := q.collectDocs(ctx, gfv.Id, preferredLang)
		if err != nil {
			return nil, fmt.Errorf("collect docs: %w", err)
		}
		out = append(out, types.GovernanceFrameworkVersionWithDocs{
			Id:            gfv.Id,
			EcosystemId:   gfv.EcosystemId,
			CorporationId: gfv.CorporationId,
			Created:       gfv.Created,
			Version:       gfv.Version,
			ActiveSince:   gfv.ActiveSince,
			Documents:     docs,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

// ListVersionsByEcosystem mirrors ListVersionsByCorporation but reads the
// GFVersionByEcosystem index. Used by MOD-ES QRY-1/QRY-2 to inline nested
// governance framework versions and their documents.
func (k Keeper) ListVersionsByEcosystem(
	ctx context.Context,
	ecID uint64,
	activeVersion uint32,
	activeOnly bool,
	preferredLang string,
) ([]types.GovernanceFrameworkVersionWithDocs, error) {
	if ecID == 0 {
		return nil, cerrors.Wrap(types.ErrInvalidSubject, "ecosystem_id must be > 0")
	}

	var gfvIDs []uint64
	iter, err := k.GFVersionByEcosystem.Iterate(ctx, collections.NewPrefixedPairRange[uint64, uint32](ecID))
	if err != nil {
		return nil, fmt.Errorf("iterate gfv index: %w", err)
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		id, err := iter.Value()
		if err != nil {
			return nil, fmt.Errorf("iter value: %w", err)
		}
		gfvIDs = append(gfvIDs, id)
	}

	out := make([]types.GovernanceFrameworkVersionWithDocs, 0, len(gfvIDs))
	q := querier{Keeper: k}
	for _, id := range gfvIDs {
		gfv, err := k.GFVersion.Get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("fetch gfv %d: %w", id, err)
		}
		if activeOnly && gfv.Version != activeVersion {
			continue
		}
		docs, err := q.collectDocs(ctx, gfv.Id, preferredLang)
		if err != nil {
			return nil, fmt.Errorf("collect docs: %w", err)
		}
		out = append(out, types.GovernanceFrameworkVersionWithDocs{
			Id:            gfv.Id,
			EcosystemId:   gfv.EcosystemId,
			CorporationId: gfv.CorporationId,
			Created:       gfv.Created,
			Version:       gfv.Version,
			ActiveSince:   gfv.ActiveSince,
			Documents:     docs,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}
