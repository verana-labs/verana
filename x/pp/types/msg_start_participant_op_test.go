package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/pp/types"
)

// TestMsgStartParticipantOP_ValidateBasic exercises the mandatory fields and
// enum whitelist per spec [MOD-PP-MSG-1-1] and [MOD-PP-MSG-1-2-1]. Valid
// `type` values are {ISSUER_GRANTOR, VERIFIER_GRANTOR, ISSUER, VERIFIER, HOLDER}.
// UNSPECIFIED and ECOSYSTEM MUST be rejected because root participants are only created
// via MsgCreateRootParticipant, never via StartParticipantOP.
func TestMsgStartParticipantOP_ValidateBasic(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("test_address________")).String()
	validDid := "did:example:123456789abcdefghi"

	valid := func() *types.MsgStartParticipantOP {
		return &types.MsgStartParticipantOP{
			Corporation:            validAddr,
			Operator:               validAddr,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: 1,
			Did:                    validDid,
		}
	}

	tests := []struct {
		name    string
		mutate  func(m *types.MsgStartParticipantOP)
		wantErr string
	}{
		{"valid ISSUER", func(m *types.MsgStartParticipantOP) {}, ""},
		{"valid VERIFIER", func(m *types.MsgStartParticipantOP) { m.Role = types.ParticipantRole_VERIFIER }, ""},
		{"valid ISSUER_GRANTOR", func(m *types.MsgStartParticipantOP) { m.Role = types.ParticipantRole_ISSUER_GRANTOR }, ""},
		{"valid VERIFIER_GRANTOR", func(m *types.MsgStartParticipantOP) { m.Role = types.ParticipantRole_VERIFIER_GRANTOR }, ""},
		{"valid HOLDER", func(m *types.MsgStartParticipantOP) { m.Role = types.ParticipantRole_HOLDER }, ""},
		{"type UNSPECIFIED rejected", func(m *types.MsgStartParticipantOP) { m.Role = types.ParticipantRole_UNSPECIFIED }, "participant type must be one of"},
		{"type ECOSYSTEM rejected", func(m *types.MsgStartParticipantOP) { m.Role = types.ParticipantRole_ECOSYSTEM }, "participant type must be one of"},
		{"validator_participant_id = 0", func(m *types.MsgStartParticipantOP) { m.ValidatorParticipantId = 0 }, "validator participant ID cannot be 0"},
		{"empty did", func(m *types.MsgStartParticipantOP) { m.Did = "" }, "did is required"},
		{"malformed did", func(m *types.MsgStartParticipantOP) { m.Did = "garbage" }, "invalid DID format"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := valid()
			tc.mutate(m)
			err := m.ValidateBasic()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}
