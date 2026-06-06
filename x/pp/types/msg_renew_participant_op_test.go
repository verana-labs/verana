package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/pp/types"
)

// TestMsgRenewParticipantOP_ValidateBasic exercises the three spec-defined
// parameters per [MOD-PP-MSG-2-1] and [MOD-PP-MSG-2-2-1]: corporation,
// operator, id. No participant_type is validated because the spec does not
// include it as a parameter.
func TestMsgRenewParticipantOP_ValidateBasic(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("test_address________")).String()

	valid := func() *types.MsgRenewParticipantOP {
		return &types.MsgRenewParticipantOP{
			Corporation: validAddr,
			Operator:    validAddr,
			Id:          1,
		}
	}

	tests := []struct {
		name    string
		mutate  func(m *types.MsgRenewParticipantOP)
		wantErr string
	}{
		{"valid baseline", func(m *types.MsgRenewParticipantOP) {}, ""},
		{"empty corporation", func(m *types.MsgRenewParticipantOP) { m.Corporation = "" }, "invalid corporation address"},
		{"invalid corporation bech32", func(m *types.MsgRenewParticipantOP) { m.Corporation = "not-bech32" }, "invalid corporation address"},
		{"empty operator", func(m *types.MsgRenewParticipantOP) { m.Operator = "" }, "invalid operator address"},
		{"id = 0", func(m *types.MsgRenewParticipantOP) { m.Id = 0 }, "participant ID cannot be 0"},
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
