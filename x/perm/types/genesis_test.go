package types_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana/x/perm/types"
)

func TestGenesisState_Validate(t *testing.T) {
	nowTime := time.Now()
	futureTime := nowTime.Add(24 * time.Hour)
	creatorAddr := sdk.AccAddress([]byte("test_creator")).String()

	validPerm1 := types.Permission{
		Id:             1,
		Type:           types.PermissionType_ECOSYSTEM,
		Did:            "did:example:12345",
		Grantee:        creatorAddr,
		Created:        &nowTime,
		CreatedBy:      creatorAddr,
		Modified:       &nowTime,
		SchemaId:       1,
		Country:        "US",
		EffectiveFrom:  &nowTime,
		EffectiveUntil: &futureTime,
	}

	validPerm2 := types.Permission{
		Id:              2,
		Type:            types.PermissionType_ISSUER,
		Did:             "did:example:67890",
		Grantee:         creatorAddr,
		Created:         &nowTime,
		CreatedBy:       creatorAddr,
		Modified:        &nowTime,
		SchemaId:        1,
		Country:         "CA",
		EffectiveFrom:   &nowTime,
		EffectiveUntil:  &futureTime,
		ValidatorPermId: 1,
	}

	validSession := types.PermissionSession{
		Id:          "test-session-id",
		Controller:  creatorAddr,
		AgentPermId: 2,
		Created:     &nowTime,
		Modified:    &nowTime,
		Authz: []*types.SessionAuthz{
			{
				ExecutorPermId:    1,
				BeneficiaryPermId: 2,
			},
		},
	}

	tests := []struct {
		desc        string
		genState    *types.GenesisState
		valid       bool
		errorString string
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state with permissions and sessions",
			genState: &types.GenesisState{
				Params:             types.DefaultParams(),
				Permissions:        []types.Permission{validPerm1, validPerm2},
				PermissionSessions: []types.PermissionSession{validSession},
				NextPermissionId:   3,
			},
			valid: true,
		},
		{
			desc: "invalid params",
			genState: &types.GenesisState{
				Params: types.Params{
					ValidationTermRequestedTimeoutDays: 0, // Invalid - must be positive
				},
				Permissions:        []types.Permission{},
				PermissionSessions: []types.PermissionSession{},
				NextPermissionId:   1,
			},
			valid:       false,
			errorString: "validation term requested timeout days must be positive",
		},
		{
			desc: "duplicate perm IDs",
			genState: &types.GenesisState{
				Params:             types.DefaultParams(),
				Permissions:        []types.Permission{validPerm1, validPerm1}, // Duplicate ID
				PermissionSessions: []types.PermissionSession{},
				NextPermissionId:   3,
			},
			valid:       false,
			errorString: "duplicate perm ID",
		},
		{
			desc: "missing perm ID",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Permissions: []types.Permission{
					{
						Id:       0, // Invalid ID
						Type:     types.PermissionType_ISSUER,
						Grantee:  creatorAddr,
						Created:  &nowTime,
						Modified: &nowTime,
					},
				},
				PermissionSessions: []types.PermissionSession{},
				NextPermissionId:   1,
			},
			valid:       false,
			errorString: "perm ID cannot be 0",
		},
		{
			desc: "invalid validator reference",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Permissions: []types.Permission{
					{
						Id:              1,
						Type:            types.PermissionType_ISSUER,
						Grantee:         creatorAddr,
						Created:         &nowTime,
						Modified:        &nowTime,
						ValidatorPermId: 999, // Non-existent validator
					},
				},
				PermissionSessions: []types.PermissionSession{},
				NextPermissionId:   2,
			},
			valid:       false,
			errorString: "validator perm ID 999 not found",
		},
		{
			desc: "next perm ID too low",
			genState: &types.GenesisState{
				Params:             types.DefaultParams(),
				Permissions:        []types.Permission{validPerm1, validPerm2},
				PermissionSessions: []types.PermissionSession{},
				NextPermissionId:   1, // Should be > 2
			},
			valid:       false,
			errorString: "next_permission_id (1) must be greater than",
		},
		{
			desc: "missing session reference",
			genState: &types.GenesisState{
				Params:      types.DefaultParams(),
				Permissions: []types.Permission{validPerm1},
				PermissionSessions: []types.PermissionSession{
					{
						Id:          "test-session-id",
						Controller:  creatorAddr,
						AgentPermId: 999, // Non-existent perm
						Created:     &nowTime,
						Modified:    &nowTime,
					},
				},
				NextPermissionId: 2,
			},
			valid:       false,
			errorString: "agent perm ID 999 not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				if tc.errorString != "" {
					require.Contains(t, err.Error(), tc.errorString)
				}
			}
		})
	}
}
