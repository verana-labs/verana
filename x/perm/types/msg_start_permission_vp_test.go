package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/perm/types"
)

// TestMsgStartPermissionVP_ValidateBasic exercises the mandatory fields and
// enum whitelist per spec [MOD-PERM-MSG-1-1] and [MOD-PERM-MSG-1-2-1]. Valid
// `type` values are {ISSUER_GRANTOR, VERIFIER_GRANTOR, ISSUER, VERIFIER, HOLDER}.
// UNSPECIFIED and ECOSYSTEM MUST be rejected because root perms are only created
// via MsgCreateRootPermission, never via StartPermissionVP.
func TestMsgStartPermissionVP_ValidateBasic(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("test_address________")).String()
	validDid := "did:example:123456789abcdefghi"

	valid := func() *types.MsgStartPermissionVP {
		return &types.MsgStartPermissionVP{
			Corporation:     validAddr,
			Operator:        validAddr,
			Type:            types.PermissionType_ISSUER,
			ValidatorPermId: 1,
			Did:             validDid,
		}
	}

	tests := []struct {
		name    string
		mutate  func(m *types.MsgStartPermissionVP)
		wantErr string
	}{
		{"valid ISSUER", func(m *types.MsgStartPermissionVP) {}, ""},
		{"valid VERIFIER", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_VERIFIER }, ""},
		{"valid ISSUER_GRANTOR", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_ISSUER_GRANTOR }, ""},
		{"valid VERIFIER_GRANTOR", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_VERIFIER_GRANTOR }, ""},
		{"valid HOLDER", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_HOLDER }, ""},
		{"type UNSPECIFIED rejected", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_UNSPECIFIED }, "perm type must be one of"},
		{"type ECOSYSTEM rejected", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_ECOSYSTEM }, "perm type must be one of"},
		{"validator_perm_id = 0", func(m *types.MsgStartPermissionVP) { m.ValidatorPermId = 0 }, "validator perm ID cannot be 0"},
		{"empty did", func(m *types.MsgStartPermissionVP) { m.Did = "" }, "did is required"},
		{"malformed did", func(m *types.MsgStartPermissionVP) { m.Did = "garbage" }, "invalid DID format"},
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
