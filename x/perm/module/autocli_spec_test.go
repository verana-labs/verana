package permission_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	permission "github.com/verana-labs/verana/x/perm/module"
)

// TestAutocli_CreateRootPermission_HasNoNonSpecFlags locks in the spec shape
// of the `create-root-perm` command. Before this PR, PR #280 added a proto
// field `permission_type` (and `vs_operator`) to MsgCreateRootPermission and
// `permission_type` to MsgRenewPermissionVP based on a misread of VPR spec v4
// draft 13. The autocli declaration never exposed them as explicit flags, so
// the `veranad` CLI silently sent the proto3 zero value and devnet stored
// root permissions with `type: UNSPECIFIED`.
//
// This test fails if either proto field is reintroduced or surfaces as a
// named flag.
//
// Spec anchors:
//   - [MOD-PERM-MSG-7-1] parameters of CreateRootPermission
//   - [MOD-PERM-MSG-2-1] parameters of RenewPermissionVP
func TestAutocli_CreateRootPermission_HasNoNonSpecFlags(t *testing.T) {
	opts := permission.AppModule{}.AutoCLIOptions()
	require.NotNil(t, opts)
	require.NotNil(t, opts.Tx)

	for _, cmd := range opts.Tx.RpcCommandOptions {
		if cmd.RpcMethod != "CreateRootPermission" {
			continue
		}
		_, hasPermType := cmd.FlagOptions["permission_type"]
		require.False(t, hasPermType,
			"spec [MOD-PERM-MSG-7-1] does not define permission_type; CLI flag must not exist")
		_, hasVsOperator := cmd.FlagOptions["vs_operator"]
		require.False(t, hasVsOperator,
			"spec [MOD-PERM-MSG-7-1] does not define vs_operator; CLI flag must not exist")
		require.Equal(t,
			"create-root-perm [schema-id] [did] [validation-fees] [issuance-fees] [verification-fees]",
			cmd.Use,
			"create-root-perm Use string must match spec [MOD-PERM-MSG-7-1] parameters")
		return
	}
	t.Fatalf("CreateRootPermission RpcMethod not found in autocli declaration")
}

func TestAutocli_RenewPermissionVP_HasNoPermissionTypeFlag(t *testing.T) {
	opts := permission.AppModule{}.AutoCLIOptions()
	require.NotNil(t, opts)
	require.NotNil(t, opts.Tx)

	for _, cmd := range opts.Tx.RpcCommandOptions {
		if cmd.RpcMethod != "RenewPermissionVP" {
			continue
		}
		_, hasPermType := cmd.FlagOptions["permission_type"]
		require.False(t, hasPermType,
			"spec [MOD-PERM-MSG-2-1] does not define permission_type; CLI flag must not exist")
		return
	}
	t.Fatalf("RenewPermissionVP RpcMethod not found in autocli declaration")
}
