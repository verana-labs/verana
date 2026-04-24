package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/perm/types"
)

// TestMsgSelfCreatePermission_ValidateBasic exercises the mandatory fields
// and the narrow enum whitelist per spec [MOD-PERM-MSG-14-1] and
// [MOD-PERM-MSG-14-2-1]. Valid `type` values are ISSUER or VERIFIER ONLY;
// all other enum values MUST be rejected.
func TestMsgSelfCreatePermission_ValidateBasic(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("test_address________")).String()
	validDid := "did:example:123456789abcdefghi"

	valid := func() *types.MsgSelfCreatePermission {
		return &types.MsgSelfCreatePermission{
			Corporation:     validAddr,
			Operator:        validAddr,
			Type:            types.PermissionType_ISSUER,
			ValidatorPermId: 1,
			Did:             validDid,
		}
	}

	tests := []struct {
		name    string
		mutate  func(m *types.MsgSelfCreatePermission)
		wantErr string
	}{
		{"valid ISSUER", func(m *types.MsgSelfCreatePermission) {}, ""},
		{"valid VERIFIER", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_VERIFIER }, ""},
		{"type UNSPECIFIED rejected", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_UNSPECIFIED }, "type must be ISSUER or VERIFIER"},
		{"type ECOSYSTEM rejected", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_ECOSYSTEM }, "type must be ISSUER or VERIFIER"},
		{"type ISSUER_GRANTOR rejected", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_ISSUER_GRANTOR }, "type must be ISSUER or VERIFIER"},
		{"type VERIFIER_GRANTOR rejected", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_VERIFIER_GRANTOR }, "type must be ISSUER or VERIFIER"},
		{"type HOLDER rejected", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_HOLDER }, "type must be ISSUER or VERIFIER"},
		{"validator_perm_id = 0", func(m *types.MsgSelfCreatePermission) { m.ValidatorPermId = 0 }, "validator_perm_id is mandatory"},
		{"empty did", func(m *types.MsgSelfCreatePermission) { m.Did = "" }, "did is mandatory"},
		{"malformed did", func(m *types.MsgSelfCreatePermission) { m.Did = "nope" }, "invalid DID syntax"},
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
