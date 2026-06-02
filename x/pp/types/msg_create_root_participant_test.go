package types_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/pp/types"
)

// TestMsgCreateRootParticipant_ValidateBasic exercises every mandatory-field
// rejection per spec [MOD-PP-MSG-7-1] and [MOD-PP-MSG-7-2-1]. Every case
// starts from a valid baseline and mutates exactly one field. This pattern
// surfaces the "field omitted, proto3 zero value" bug class that the Mohammad
// devnet report (2026-04-23) uncovered.
func TestMsgCreateRootParticipant_ValidateBasic(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("test_address________")).String()
	validDid := "did:example:123456789abcdefghi"

	valid := func() *types.MsgCreateRootParticipant {
		return &types.MsgCreateRootParticipant{
			Corporation:      validAddr,
			Operator:         validAddr,
			SchemaId:         1,
			Did:              validDid,
			ValidationFees:   0,
			IssuanceFees:     0,
			VerificationFees: 0,
		}
	}

	tests := []struct {
		name    string
		mutate  func(m *types.MsgCreateRootParticipant)
		wantErr string
	}{
		{"valid baseline", func(m *types.MsgCreateRootParticipant) {}, ""},
		{"empty corporation", func(m *types.MsgCreateRootParticipant) { m.Corporation = "" }, "invalid corporation address"},
		{"invalid corporation bech32", func(m *types.MsgCreateRootParticipant) { m.Corporation = "not-bech32" }, "invalid corporation address"},
		{"empty operator", func(m *types.MsgCreateRootParticipant) { m.Operator = "" }, "invalid operator address"},
		{"invalid operator bech32", func(m *types.MsgCreateRootParticipant) { m.Operator = "cosmos1garbage" }, "invalid operator address"},
		{"schema_id = 0", func(m *types.MsgCreateRootParticipant) { m.SchemaId = 0 }, "schema ID cannot be 0"},
		{"empty did", func(m *types.MsgCreateRootParticipant) { m.Did = "" }, "DID is required"},
		{"malformed did", func(m *types.MsgCreateRootParticipant) { m.Did = "not-a-did" }, "invalid DID format"},
		{"vsoa params without effective_until", func(m *types.MsgCreateRootParticipant) {
			m.VsOperator = validAddr
			m.VsOperatorAuthzMsgTypes = []string{types.MsgSetParticipantOPToValidatedTypeURL}
		}, "effective_until is required"},
		{"vsoa params with effective_until", func(m *types.MsgCreateRootParticipant) {
			m.VsOperator = validAddr
			m.VsOperatorAuthzMsgTypes = []string{types.MsgSetParticipantOPToValidatedTypeURL}
			exp := time.Unix(2000000000, 0).UTC()
			m.EffectiveUntil = &exp
		}, ""},
		{"vsoa msg_type not permitted for root", func(m *types.MsgCreateRootParticipant) {
			m.VsOperator = validAddr
			m.VsOperatorAuthzMsgTypes = []string{types.MsgCreateOrUpdateParticipantSessionTypeURL}
			exp := time.Unix(2000000000, 0).UTC()
			m.EffectiveUntil = &exp
		}, "not permitted for root participant"},
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
