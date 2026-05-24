package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/gf/types"
)

func TestGenesisValidate(t *testing.T) {
	t.Run("default genesis is valid", func(t *testing.T) {
		require.NoError(t, types.DefaultGenesis().Validate())
	})

	t.Run("rejects GFV with both ecosystem_id and corporation set", func(t *testing.T) {
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Versions: []types.GovernanceFrameworkVersion{
				{Id: 1, EcosystemId: 1, CorporationId: 1, Version: 1},
			},
		}
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidSubject)
	})

	t.Run("rejects GFV with neither ecosystem_id nor corporation set", func(t *testing.T) {
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Versions: []types.GovernanceFrameworkVersion{
				{Id: 1, Version: 1},
			},
		}
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidSubject)
	})

	t.Run("rejects GFD with dangling gfv_id", func(t *testing.T) {
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Versions: []types.GovernanceFrameworkVersion{
				{Id: 1, CorporationId: 1, Version: 1},
			},
			Documents: []types.GovernanceFrameworkDocument{
				{Id: 1, GfvId: 999, Language: "en"},
			},
		}
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidVersion)
	})
}
