package trustdeposit_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "github.com/verana-labs/verana-blockchain/testutil/keeper"
	"github.com/verana-labs/verana-blockchain/testutil/nullify"
	trustdeposit "github.com/verana-labs/verana-blockchain/x/td/module"
	"github.com/verana-labs/verana-blockchain/x/td/types"
)

func TestGenesis(t *testing.T) {
	// Initialize test addresses
	addr1 := sdk.AccAddress([]byte("test_address1")).String()
	addr2 := sdk.AccAddress([]byte("test_address2")).String()

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		TrustDeposits: []types.TrustDepositRecord{
			{
				Account:   addr1,
				Share:     100,
				Amount:    1000,
				Claimable: 50,
			},
			{
				Account:   addr2,
				Share:     200,
				Amount:    2000,
				Claimable: 100,
			},
		},
	}

	k, ctx := keepertest.TrustdepositKeeper(t)
	trustdeposit.InitGenesis(ctx, k, genesisState)
	got := trustdeposit.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	// Use nullify to ignore fields that are updated by the keeper (timestamps, etc.)
	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// Verify params
	require.Equal(t, genesisState.Params, got.Params)

	// Verify trust deposits (may need to sort since map iteration is non-deterministic)
	require.ElementsMatch(t, genesisState.TrustDeposits, got.TrustDeposits)

	// Verify that trust deposits were correctly stored in the keeper
	for _, td := range genesisState.TrustDeposits {
		stored, err := k.TrustDeposit.Get(ctx, td.Account)
		require.NoError(t, err)
		require.Equal(t, td.Account, stored.Account)
		require.Equal(t, td.Share, stored.Share)
		require.Equal(t, td.Amount, stored.Amount)
		require.Equal(t, td.Claimable, stored.Claimable)
	}
}

// TestEmptyGenesis tests the initialization with empty genesis state
func TestEmptyGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:        types.DefaultParams(),
		TrustDeposits: []types.TrustDepositRecord{},
	}

	k, ctx := keepertest.TrustdepositKeeper(t)
	trustdeposit.InitGenesis(ctx, k, genesisState)
	exported := trustdeposit.ExportGenesis(ctx, k)

	require.Equal(t, genesisState.Params, exported.Params)
	require.Empty(t, exported.TrustDeposits)
}

// TestImportExportGenesisWithTrustDeposits tests that import/export preserves state
func TestImportExportGenesisWithTrustDeposits(t *testing.T) {
	// Create keeper and context
	k, ctx := keepertest.TrustdepositKeeper(t)

	// Create trust deposits directly with the keeper
	addr1 := sdk.AccAddress([]byte("test_address1")).String()
	addr2 := sdk.AccAddress([]byte("test_address2")).String()

	td1 := types.TrustDeposit{
		Account:   addr1,
		Share:     100,
		Amount:    1000,
		Claimable: 50,
	}

	td2 := types.TrustDeposit{
		Account:   addr2,
		Share:     200,
		Amount:    2000,
		Claimable: 100,
	}

	// Save trust deposits
	require.NoError(t, k.TrustDeposit.Set(ctx, addr1, td1))
	require.NoError(t, k.TrustDeposit.Set(ctx, addr2, td2))

	// Export genesis
	exported := trustdeposit.ExportGenesis(ctx, k)

	// Verify exported genesis contains the trust deposits
	require.Len(t, exported.TrustDeposits, 2)

	// Initialize a new keeper with the exported genesis
	newK, newCtx := keepertest.TrustdepositKeeper(t)
	trustdeposit.InitGenesis(newCtx, newK, *exported)

	// Verify the trust deposits were correctly imported
	storedTd1, err := newK.TrustDeposit.Get(newCtx, addr1)
	require.NoError(t, err)
	require.Equal(t, td1, storedTd1)

	storedTd2, err := newK.TrustDeposit.Get(newCtx, addr2)
	require.NoError(t, err)
	require.Equal(t, td2, storedTd2)
}
