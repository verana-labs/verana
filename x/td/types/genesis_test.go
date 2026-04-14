package types_test

import (
	"testing"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana/x/td/types"
)

func TestGenesisState_Validate(t *testing.T) {
	// Setup valid addresses for testing
	validAddr1 := sdk.AccAddress([]byte("test_address1")).String()
	validAddr2 := sdk.AccAddress([]byte("test_address2")).String()

	// Create a custom invalid param for testing
	invalidParams := types.DefaultParams()
	invalidShareValue, _ := math.LegacyNewDecFromStr("0.0") // Zero is invalid
	invalidParams.TrustDepositShareValue = invalidShareValue

	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state with trust deposits",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TrustDeposits: []types.TrustDepositRecord{
					{
						Corporation: validAddr1,
						Share:     math.LegacyNewDec(100),
						Deposit:   1000,
						Claimable: 500,
					},
					{
						Corporation: validAddr2,
						Share:     math.LegacyNewDec(200),
						Deposit:   2000,
						Claimable: 1000,
					},
				},
			},
			valid: true,
		},
		{
			desc: "invalid parameter",
			genState: &types.GenesisState{
				Params:        invalidParams,
				TrustDeposits: []types.TrustDepositRecord{},
			},
			valid: false,
		},
		{
			desc: "invalid account address",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TrustDeposits: []types.TrustDepositRecord{
					{
						Corporation: "invalid_address", // Invalid: not a valid bech32 address
						Share:       math.LegacyNewDec(100),
						Deposit:     1000,
						Claimable:   500,
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicate account",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TrustDeposits: []types.TrustDepositRecord{
					{
						Corporation: validAddr1,
						Share:     math.LegacyNewDec(100),
						Deposit:   1000,
						Claimable: 500,
					},
					{
						Corporation: validAddr1, // Duplicate account
						Share:     math.LegacyNewDec(200),
						Deposit:   2000,
						Claimable: 1000,
					},
				},
			},
			valid: false,
		},
		{
			desc: "claimable exceeds amount",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TrustDeposits: []types.TrustDepositRecord{
					{
						Corporation: validAddr1,
						Share:     math.LegacyNewDec(100),
						Deposit:   1000,
						Claimable: 1500, // Invalid: claimable > deposit
					},
				},
			},
			valid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
