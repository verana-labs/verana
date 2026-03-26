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

	for _, oa := range genState.OperatorAuthorizations {
		key := collections.Join(oa.Authority, oa.Operator)
		if err := k.OperatorAuthorizations.Set(ctx, key, oa); err != nil {
			return fmt.Errorf("failed to set operator authorization: %w", err)
		}
	}

	for _, fg := range genState.FeeGrants {
		key := collections.Join(fg.Grantor, fg.Grantee)
		if err := k.FeeGrants.Set(ctx, key, fg); err != nil {
			return fmt.Errorf("failed to set fee grant: %w", err)
		}
	}

	for _, vsoa := range genState.VsOperatorAuthorizations {
		key := collections.Join(vsoa.Authority, vsoa.VsOperator)
		if err := k.VSOperatorAuthorizations.Set(ctx, key, vsoa); err != nil {
			return fmt.Errorf("failed to set vs operator authorization: %w", err)
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

	// Export operator authorizations
	oaList := []types.OperatorAuthorization{}
	if err := k.OperatorAuthorizations.Walk(ctx, nil, func(key collections.Pair[string, string], val types.OperatorAuthorization) (bool, error) {
		oaList = append(oaList, val)
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to export operator authorizations: %w", err)
	}
	sort.Slice(oaList, func(i, j int) bool {
		if oaList[i].Authority != oaList[j].Authority {
			return oaList[i].Authority < oaList[j].Authority
		}
		return oaList[i].Operator < oaList[j].Operator
	})
	genesis.OperatorAuthorizations = oaList

	// Export fee grants
	fgList := []types.FeeGrant{}
	if err := k.FeeGrants.Walk(ctx, nil, func(key collections.Pair[string, string], val types.FeeGrant) (bool, error) {
		fgList = append(fgList, val)
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to export fee grants: %w", err)
	}
	sort.Slice(fgList, func(i, j int) bool {
		if fgList[i].Grantor != fgList[j].Grantor {
			return fgList[i].Grantor < fgList[j].Grantor
		}
		return fgList[i].Grantee < fgList[j].Grantee
	})
	genesis.FeeGrants = fgList

	// Export VS operator authorizations
	vsoaList := []types.VSOperatorAuthorization{}
	if err := k.VSOperatorAuthorizations.Walk(ctx, nil, func(key collections.Pair[string, string], val types.VSOperatorAuthorization) (bool, error) {
		vsoaList = append(vsoaList, val)
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to export vs operator authorizations: %w", err)
	}
	sort.Slice(vsoaList, func(i, j int) bool {
		if vsoaList[i].Authority != vsoaList[j].Authority {
			return vsoaList[i].Authority < vsoaList[j].Authority
		}
		return vsoaList[i].VsOperator < vsoaList[j].VsOperator
	})
	genesis.VsOperatorAuthorizations = vsoaList

	return genesis, nil
}
