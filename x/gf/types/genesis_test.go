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

	t.Run("rejects GFD with invalid BCP47 language", func(t *testing.T) {
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Versions: []types.GovernanceFrameworkVersion{
				{Id: 1, CorporationId: 1, Version: 1},
			},
			Documents: []types.GovernanceFrameworkDocument{
				{Id: 1, GfvId: 1, Language: "1bad"},
			},
		}
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidLanguage)
	})

	t.Run("rejects GFV with version below 1", func(t *testing.T) {
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Versions: []types.GovernanceFrameworkVersion{
				{Id: 1, CorporationId: 1, Version: 0},
			},
		}
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidVersion)
	})

	t.Run("accepts valid mixed eco + corp GFVs with valid GFDs", func(t *testing.T) {
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Versions: []types.GovernanceFrameworkVersion{
				{Id: 1, EcosystemId: 7, Version: 1},
				{Id: 2, CorporationId: 9, Version: 1},
			},
			Documents: []types.GovernanceFrameworkDocument{
				{Id: 1, GfvId: 1, Language: "en"},
				{Id: 2, GfvId: 2, Language: "fr"},
			},
		}
		require.NoError(t, gs.Validate())
	})
}
