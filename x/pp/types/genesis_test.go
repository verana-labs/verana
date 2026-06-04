package types_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana/x/pp/types"
)

func TestGenesisState_Validate(t *testing.T) {
	nowTime := time.Now()
	futureTime := nowTime.Add(24 * time.Hour)
	creatorAddr := sdk.AccAddress([]byte("test_creator")).String()

	validParticipant1 := types.Participant{
		Id:                1,
		Role:              types.ParticipantRole_ECOSYSTEM,
		Did:               "did:example:12345",
		CorporationId:     uint64(1),
		Created:           &nowTime,
		Modified:          &nowTime,
		OpState:           types.OnboardingState_VALIDATED,
		OpLastStateChange: &nowTime,
		SchemaId:          1,
		EffectiveFrom:     &nowTime,
		EffectiveUntil:    &futureTime,
	}

	validParticipant2 := types.Participant{
		Id:                     2,
		Role:                   types.ParticipantRole_ISSUER,
		Did:                    "did:example:67890",
		CorporationId:          uint64(1),
		Created:                &nowTime,
		Modified:               &nowTime,
		OpState:                types.OnboardingState_VALIDATED,
		OpLastStateChange:      &nowTime,
		SchemaId:               1,
		EffectiveFrom:          &nowTime,
		EffectiveUntil:         &futureTime,
		ValidatorParticipantId: 1,
	}

	validSession := types.ParticipantSession{
		Id:            "test-session-id",
		CorporationId: uint64(1),
		VsOperator:    creatorAddr,
		Created:       &nowTime,
		Modified:      &nowTime,
		SessionRecords: []*types.ParticipantSessionRecord{
			{
				Id:                  1,
				IssuerParticipantId: 1,
				AgentParticipantId:  2,
				Created:             &nowTime,
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
			desc: "valid genesis state with participants and sessions",
			genState: &types.GenesisState{
				Params:              types.DefaultParams(),
				Participants:        []types.Participant{validParticipant1, validParticipant2},
				ParticipantSessions: []types.ParticipantSession{validSession},
				NextParticipantId:   3,
			},
			valid: true,
		},
		{
			desc: "invalid params",
			genState: &types.GenesisState{
				Params: types.Params{
					ValidationTermRequestedTimeoutDays: 0, // Invalid - must be positive
				},
				Participants:        []types.Participant{},
				ParticipantSessions: []types.ParticipantSession{},
				NextParticipantId:   1,
			},
			valid:       false,
			errorString: "validation term requested timeout days must be positive",
		},
		{
			desc: "duplicate participant IDs",
			genState: &types.GenesisState{
				Params:              types.DefaultParams(),
				Participants:        []types.Participant{validParticipant1, validParticipant1}, // Duplicate ID
				ParticipantSessions: []types.ParticipantSession{},
				NextParticipantId:   3,
			},
			valid:       false,
			errorString: "duplicate participant ID",
		},
		{
			desc: "missing participant ID",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Participants: []types.Participant{
					{
						Id:            0, // Invalid ID
						Role:          types.ParticipantRole_ISSUER,
						CorporationId: uint64(1),
						Created:       &nowTime,
						Modified:      &nowTime,
					},
				},
				ParticipantSessions: []types.ParticipantSession{},
				NextParticipantId:   1,
			},
			valid:       false,
			errorString: "participant ID cannot be 0",
		},
		{
			desc: "invalid validator reference",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				Participants: []types.Participant{
					{
						Id:                     1,
						Role:                   types.ParticipantRole_ISSUER,
						Did:                    "did:example:val",
						CorporationId:          uint64(1),
						Created:                &nowTime,
						Modified:               &nowTime,
						OpState:                types.OnboardingState_VALIDATED,
						OpLastStateChange:      &nowTime,
						ValidatorParticipantId: 999, // Non-existent validator
					},
				},
				ParticipantSessions: []types.ParticipantSession{},
				NextParticipantId:   2,
			},
			valid:       false,
			errorString: "validator participant ID 999 not found",
		},
		{
			desc: "next participant ID too low",
			genState: &types.GenesisState{
				Params:              types.DefaultParams(),
				Participants:        []types.Participant{validParticipant1, validParticipant2},
				ParticipantSessions: []types.ParticipantSession{},
				NextParticipantId:   1, // Should be > 2
			},
			valid:       false,
			errorString: "next_participant_id (1) must be greater than",
		},
		{
			desc: "missing session reference",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				Participants: []types.Participant{validParticipant1},
				ParticipantSessions: []types.ParticipantSession{
					{
						Id:            "test-session-id",
						CorporationId: uint64(1),
						VsOperator:    creatorAddr,
						Created:       &nowTime,
						Modified:      &nowTime,
						SessionRecords: []*types.ParticipantSessionRecord{
							{
								Id:                  1,
								IssuerParticipantId: 1,
								AgentParticipantId:  999, // Non-existent participant
								Created:             &nowTime,
							},
						},
					},
				},
				NextParticipantId: 2,
			},
			valid:       false,
			errorString: "agent participant ID 999 not found",
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
