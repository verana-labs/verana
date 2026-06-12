package keeper

import (
	"context"
	"fmt"
	"sort"

	"cosmossdk.io/collections"

	"github.com/verana-labs/verana/x/de/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	if err := k.Params.Set(ctx, genState.Params); err != nil {
		return err
	}

	var maxOAID uint64
	for _, oa := range genState.OperatorAuthorizations {
		if err := k.OperatorAuthorizations.Set(ctx, oa.Id, oa); err != nil {
			return fmt.Errorf("failed to set operator authorization: %w", err)
		}
		if err := k.OperatorAuthorizationByCorpOp.Set(ctx, collections.Join(oa.CorporationId, oa.Operator), oa.Id); err != nil {
			return fmt.Errorf("failed to set operator authorization index: %w", err)
		}
		if oa.Id > maxOAID {
			maxOAID = oa.Id
		}
	}
	if maxOAID > 0 {
		if err := k.OperatorAuthorizationSeq.Set(ctx, maxOAID); err != nil {
			return fmt.Errorf("failed to seed operator authorization sequence: %w", err)
		}
	}

	for _, fg := range genState.FeeGrants {
		key := collections.Join(fg.GrantorCorporationId, fg.Grantee)
		if err := k.FeeGrants.Set(ctx, key, fg); err != nil {
			return fmt.Errorf("failed to set fee grant: %w", err)
		}
	}

	var maxVSOAID uint64
	for _, vsoa := range genState.VsOperatorAuthorizations {
		if err := k.VSOperatorAuthorizations.Set(ctx, vsoa.Id, vsoa); err != nil {
			return fmt.Errorf("failed to set vs operator authorization: %w", err)
		}
		if err := k.VSOAByCorpOp.Set(ctx, collections.Join(vsoa.CorporationId, vsoa.VsOperator), vsoa.Id); err != nil {
			return fmt.Errorf("failed to set vs operator authorization index: %w", err)
		}
		for _, rec := range vsoa.Records {
			if err := k.VSOAByParticipant.Set(ctx, rec.ParticipantId, vsoa.Id); err != nil {
				return fmt.Errorf("failed to set participant index: %w", err)
			}
		}
		if vsoa.Id > maxVSOAID {
			maxVSOAID = vsoa.Id
		}
	}
	if maxVSOAID > 0 {
		if err := k.VSOASeq.Set(ctx, maxVSOAID); err != nil {
			return fmt.Errorf("failed to seed vs operator authorization sequence: %w", err)
		}
	}

	return nil
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	var err error

	genesis := types.DefaultGenesis()
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Operator authorizations (keyed by id).
	oaList := []types.OperatorAuthorization{}
	if err := k.OperatorAuthorizations.Walk(ctx, nil, func(_ uint64, val types.OperatorAuthorization) (bool, error) {
		oaList = append(oaList, val)
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to export operator authorizations: %w", err)
	}
	sort.Slice(oaList, func(i, j int) bool { return oaList[i].Id < oaList[j].Id })
	genesis.OperatorAuthorizations = oaList

	// Fee grants (composite key).
	fgList := []types.FeeGrant{}
	if err := k.FeeGrants.Walk(ctx, nil, func(_ collections.Pair[uint64, string], val types.FeeGrant) (bool, error) {
		fgList = append(fgList, val)
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to export fee grants: %w", err)
	}
	sort.Slice(fgList, func(i, j int) bool {
		if fgList[i].GrantorCorporationId != fgList[j].GrantorCorporationId {
			return fgList[i].GrantorCorporationId < fgList[j].GrantorCorporationId
		}
		return fgList[i].Grantee < fgList[j].Grantee
	})
	genesis.FeeGrants = fgList

	// VS operator authorizations (keyed by id).
	vsoaList := []types.VSOperatorAuthorization{}
	if err := k.VSOperatorAuthorizations.Walk(ctx, nil, func(_ uint64, val types.VSOperatorAuthorization) (bool, error) {
		vsoaList = append(vsoaList, val)
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to export vs operator authorizations: %w", err)
	}
	sort.Slice(vsoaList, func(i, j int) bool { return vsoaList[i].Id < vsoaList[j].Id })
	genesis.VsOperatorAuthorizations = vsoaList

	return genesis, nil
}
