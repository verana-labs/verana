package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana/x/diddirectory/types"
)

func TestGenesisState_Validate(t *testing.T) {
	now := time.Now().UTC()
	tomorrow := now.AddDate(0, 0, 1)
	oneYearLater := now.AddDate(1, 0, 0)

	validDID := types.DIDDirectory{
		Did:        "did:example:123456789abcdefghi",
		Controller: "cosmos1controller",
		Created:    now,
		Modified:   now,
		Exp:        oneYearLater,
		Deposit:    5000,
	}

	// Set up test cases
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
		errMsg   string
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state with one DID",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					validDID,
				},
			},
			valid: true,
		},
		{
			desc: "valid genesis state with multiple DIDs",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					validDID,
					{
						Did:        "did:example:another",
						Controller: "cosmos1controller2",
						Created:    now,
						Modified:   tomorrow, // Modified later
						Exp:        oneYearLater,
						Deposit:    10000,
					},
				},
			},
			valid: true,
		},
		{
			desc: "invalid params",
			genState: &types.GenesisState{
				Params: types.Params{
					DidDirectoryTrustDeposit: 0, // Invalid - should be positive
					DidDirectoryGracePeriod:  30,
				},
				DidDirectories: []types.DIDDirectory{},
			},
			valid:  false,
			errMsg: "invalid parameters",
		},
		{
			desc: "duplicate DID",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					validDID,
					validDID, // Duplicate
				},
			},
			valid:  false,
			errMsg: "duplicate DID Directory",
		},
		{
			desc: "empty DID",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					{
						Did:        "", // Empty DID
						Controller: "cosmos1controller",
						Created:    now,
						Modified:   now,
						Exp:        oneYearLater,
						Deposit:    5000,
					},
				},
			},
			valid:  false,
			errMsg: "empty DID at index",
		},
		{
			desc: "empty controller",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					{
						Did:        "did:example:valid",
						Controller: "", // Empty controller
						Created:    now,
						Modified:   now,
						Exp:        oneYearLater,
						Deposit:    5000,
					},
				},
			},
			valid:  false,
			errMsg: "empty controller",
		},
		{
			desc: "invalid DID format",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					{
						Did:        "invalid-did", // Invalid format
						Controller: "cosmos1controller",
						Created:    now,
						Modified:   now,
						Exp:        oneYearLater,
						Deposit:    5000,
					},
				},
			},
			valid:  false,
			errMsg: "invalid DID format",
		},
		{
			desc: "zero created timestamp",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					{
						Did:        "did:example:valid",
						Controller: "cosmos1controller",
						Created:    time.Time{}, // Zero timestamp
						Modified:   now,
						Exp:        oneYearLater,
						Deposit:    5000,
					},
				},
			},
			valid:  false,
			errMsg: "zero created timestamp",
		},
		{
			desc: "zero modified timestamp",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					{
						Did:        "did:example:valid",
						Controller: "cosmos1controller",
						Created:    now,
						Modified:   time.Time{}, // Zero timestamp
						Exp:        oneYearLater,
						Deposit:    5000,
					},
				},
			},
			valid:  false,
			errMsg: "zero modified timestamp",
		},
		{
			desc: "zero expiration timestamp",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					{
						Did:        "did:example:valid",
						Controller: "cosmos1controller",
						Created:    now,
						Modified:   now,
						Exp:        time.Time{}, // Zero timestamp
						Deposit:    5000,
					},
				},
			},
			valid:  false,
			errMsg: "zero expiration timestamp",
		},
		{
			desc: "modified before created",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					{
						Did:        "did:example:valid",
						Controller: "cosmos1controller",
						Created:    tomorrow,
						Modified:   now, // Before created
						Exp:        oneYearLater,
						Deposit:    5000,
					},
				},
			},
			valid:  false,
			errMsg: "modified timestamp before created timestamp",
		},
		{
			desc: "expiration before created",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					{
						Did:        "did:example:valid",
						Controller: "cosmos1controller",
						Created:    tomorrow,
						Modified:   tomorrow,
						Exp:        now, // Before created
						Deposit:    5000,
					},
				},
			},
			valid:  false,
			errMsg: "expiration before created timestamp",
		},
		{
			desc: "non-positive deposit",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				DidDirectories: []types.DIDDirectory{
					{
						Did:        "did:example:valid",
						Controller: "cosmos1controller",
						Created:    now,
						Modified:   now,
						Exp:        oneYearLater,
						Deposit:    0, // Zero deposit
					},
				},
			},
			valid:  false,
			errMsg: "non-positive deposit value",
		},
	}

	// Run all test cases
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
			}
		})
	}
}

func TestSanitizeGenesisState(t *testing.T) {
	now := time.Now().UTC()
	oneYearLater := now.AddDate(1, 0, 0)

	// Create a genesis state with DIDs in non-alphabetical order
	genState := &types.GenesisState{
		Params: types.DefaultParams(),
		DidDirectories: []types.DIDDirectory{
			{
				Did:        "did:example:zzz",
				Controller: "cosmos1controller1",
				Created:    now,
				Modified:   now,
				Exp:        oneYearLater,
				Deposit:    5000,
			},
			{
				Did:        "did:example:aaa",
				Controller: "cosmos1controller2",
				Created:    now,
				Modified:   now,
				Exp:        oneYearLater,
				Deposit:    5000,
			},
			{
				Did:        "did:example:mmm",
				Controller: "cosmos1controller3",
				Created:    now,
				Modified:   now,
				Exp:        oneYearLater,
				Deposit:    5000,
			},
		},
	}

	// Sanitize the genesis state
	sanitized := types.SanitizeGenesisState(genState)

	// Verify that the DIDs are now in alphabetical order
	require.Equal(t, "did:example:aaa", sanitized.DidDirectories[0].Did)
	require.Equal(t, "did:example:mmm", sanitized.DidDirectories[1].Did)
	require.Equal(t, "did:example:zzz", sanitized.DidDirectories[2].Did)
}
