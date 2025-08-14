package trustregistry_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/testutil/nullify"
	trustregistry "github.com/verana-labs/verana/x/trustregistry/module"
	"github.com/verana-labs/verana/x/trustregistry/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.TrustregistryKeeper(t)
	trustregistry.InitGenesis(ctx, k, genesisState)
	got := trustregistry.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
