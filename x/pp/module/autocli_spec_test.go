package participant_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	permission "github.com/verana-labs/verana/x/pp/module"
)

// TestAutocli_CreateRootPermission_HasNoNonSpecFlags locks in the spec shape
// of the `create-root-perm` command. Before this PR, PR #280 added a proto
// field `permission_type` (and `vs_operator`) to MsgCreateRootParticipant and
// `permission_type` to MsgRenewParticipantOP based on a misread of VPR spec v4
// draft 13. The autocli declaration never exposed them as explicit flags, so
// the `veranad` CLI silently sent the proto3 zero value and devnet stored
// root permissions with `type: UNSPECIFIED`.
//
// This test fails if either proto field is reintroduced or surfaces as a
// named flag.
//
// Spec anchors:
//   - [MOD-PERM-MSG-7-1] parameters of CreateRootParticipant
//   - [MOD-PERM-MSG-2-1] parameters of RenewParticipantOP
func TestAutocli_CreateRootPermission_HasNoNonSpecFlags(t *testing.T) {
	opts := permission.AppModule{}.AutoCLIOptions()
	require.NotNil(t, opts)
	require.NotNil(t, opts.Tx)

	for _, cmd := range opts.Tx.RpcCommandOptions {
		if cmd.RpcMethod != "CreateRootParticipant" {
			continue
		}
		_, hasPermType := cmd.FlagOptions["permission_type"]
		require.False(t, hasPermType,
			"spec [MOD-PERM-MSG-7-1] does not define permission_type; CLI flag must not exist")
		_, hasVsOperator := cmd.FlagOptions["vs_operator"]
		require.False(t, hasVsOperator,
			"spec [MOD-PERM-MSG-7-1] does not define vs_operator; CLI flag must not exist")
		require.Equal(t,
			"create-root-participant [schema-id] [did] [validation-fees] [issuance-fees] [verification-fees]",
			cmd.Use,
			"create-root-perm Use string must match spec [MOD-PERM-MSG-7-1] parameters")
		return
	}
	t.Fatalf("CreateRootParticipant RpcMethod not found in autocli declaration")
}

func TestAutocli_RenewParticipantOP_HasNoRoleFlag(t *testing.T) {
	opts := permission.AppModule{}.AutoCLIOptions()
	require.NotNil(t, opts)
	require.NotNil(t, opts.Tx)

	for _, cmd := range opts.Tx.RpcCommandOptions {
		if cmd.RpcMethod != "RenewParticipantOP" {
			continue
		}
		_, hasPermType := cmd.FlagOptions["permission_type"]
		require.False(t, hasPermType,
			"spec [MOD-PERM-MSG-2-1] does not define permission_type; CLI flag must not exist")
		return
	}
	t.Fatalf("RenewParticipantOP RpcMethod not found in autocli declaration")
}
