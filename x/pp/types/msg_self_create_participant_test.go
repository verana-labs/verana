package types_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/pp/types"
)

// TestMsgSelfCreateParticipant_ValidateBasic exercises the mandatory fields
// and the narrow enum whitelist per spec [MOD-PP-MSG-14-1] and
// [MOD-PP-MSG-14-2-1]. Valid `type` values are ISSUER or VERIFIER ONLY;
// all other enum values MUST be rejected.
func TestMsgSelfCreateParticipant_ValidateBasic(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("test_address________")).String()
	validDid := "did:example:123456789abcdefghi"

	valid := func() *types.MsgSelfCreateParticipant {
		return &types.MsgSelfCreateParticipant{
			Corporation:            validAddr,
			Operator:               validAddr,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: 1,
			Did:                    validDid,
		}
	}

	tests := []struct {
		name    string
		mutate  func(m *types.MsgSelfCreateParticipant)
		wantErr string
	}{
		{"valid ISSUER", func(m *types.MsgSelfCreateParticipant) {}, ""},
		{"valid VERIFIER", func(m *types.MsgSelfCreateParticipant) { m.Role = types.ParticipantRole_VERIFIER }, ""},
		{"type UNSPECIFIED rejected", func(m *types.MsgSelfCreateParticipant) { m.Role = types.ParticipantRole_UNSPECIFIED }, "type must be ISSUER or VERIFIER"},
		{"type ECOSYSTEM rejected", func(m *types.MsgSelfCreateParticipant) { m.Role = types.ParticipantRole_ECOSYSTEM }, "type must be ISSUER or VERIFIER"},
		{"type ISSUER_GRANTOR rejected", func(m *types.MsgSelfCreateParticipant) { m.Role = types.ParticipantRole_ISSUER_GRANTOR }, "type must be ISSUER or VERIFIER"},
		{"type VERIFIER_GRANTOR rejected", func(m *types.MsgSelfCreateParticipant) { m.Role = types.ParticipantRole_VERIFIER_GRANTOR }, "type must be ISSUER or VERIFIER"},
		{"type HOLDER rejected", func(m *types.MsgSelfCreateParticipant) { m.Role = types.ParticipantRole_HOLDER }, "type must be ISSUER or VERIFIER"},
		{"validator_participant_id = 0", func(m *types.MsgSelfCreateParticipant) { m.ValidatorParticipantId = 0 }, "validator_participant_id is mandatory"},
		{"empty did", func(m *types.MsgSelfCreateParticipant) { m.Did = "" }, "did is mandatory"},
		{"malformed did", func(m *types.MsgSelfCreateParticipant) { m.Did = "nope" }, "invalid DID syntax"},
		{"vsoa params without effective_until", func(m *types.MsgSelfCreateParticipant) {
			m.VsOperator = validAddr
			m.VsOperatorAuthzMsgTypes = []string{types.MsgCreateOrUpdateParticipantSessionTypeURL}
		}, "effective_until is required"},
		{"vsoa params with effective_until", func(m *types.MsgSelfCreateParticipant) {
			m.VsOperator = validAddr
			m.VsOperatorAuthzMsgTypes = []string{types.MsgCreateOrUpdateParticipantSessionTypeURL}
			exp := time.Unix(2000000000, 0).UTC()
			m.EffectiveUntil = &exp
		}, ""},
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
