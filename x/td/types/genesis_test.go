package types_test

import (
	"testing"

	"cosmossdk.io/math"

	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana/x/td/types"
)

func TestGenesisState_Validate(t *testing.T) {
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
						CorporationId: 1,
						Share:         math.LegacyNewDec(100),
						Deposit:       1000,
						Refunded:      500,
					},
					{
						CorporationId: 2,
						Share:         math.LegacyNewDec(200),
						Deposit:       2000,
						Refunded:      1000,
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
			desc: "invalid corporation_id (zero)",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TrustDeposits: []types.TrustDepositRecord{
					{
						CorporationId: 0, // Invalid: zero corporation_id
						Share:         math.LegacyNewDec(100),
						Deposit:       1000,
						Refunded:      500,
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicate corporation_id",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TrustDeposits: []types.TrustDepositRecord{
					{
						CorporationId: 1,
						Share:         math.LegacyNewDec(100),
						Deposit:       1000,
						Refunded:      500,
					},
					{
						CorporationId: 1, // Duplicate corporation_id
						Share:         math.LegacyNewDec(200),
						Deposit:       2000,
						Refunded:      1000,
					},
				},
			},
			valid: false,
		},
		{
			desc: "refunded exceeds deposit",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				TrustDeposits: []types.TrustDepositRecord{
					{
						CorporationId: 1,
						Share:         math.LegacyNewDec(100),
						Deposit:       1000,
						Refunded:      1500, // Invalid: refunded > deposit
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
