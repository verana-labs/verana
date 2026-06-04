package participant_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	participant "github.com/verana-labs/verana/x/pp/module"
)

// TestAutocli_CreateRootParticipant_HasNoNonSpecFlags locks in the spec shape
// of the `create-root-participant` command. Before this PR, PR #280 added a proto
// field `participant_type` (and `vs_operator`) to MsgCreateRootParticipant and
// `participant_type` to MsgRenewParticipantOP based on a misread of VPR spec v4
// draft 13. The autocli declaration never exposed them as explicit flags, so
// the `veranad` CLI silently sent the proto3 zero value and devnet stored
// root participants with `type: UNSPECIFIED`.
//
// This test fails if either proto field is reintroduced or surfaces as a
// named flag.
//
// Spec anchors:
//   - [MOD-PP-MSG-7-1] parameters of CreateRootParticipant
//   - [MOD-PP-MSG-2-1] parameters of RenewParticipantOP
func TestAutocli_CreateRootParticipant_HasNoNonSpecFlags(t *testing.T) {
	opts := participant.AppModule{}.AutoCLIOptions()
	require.NotNil(t, opts)
	require.NotNil(t, opts.Tx)

	for _, cmd := range opts.Tx.RpcCommandOptions {
		if cmd.RpcMethod != "CreateRootParticipant" {
			continue
		}
		_, hasParticipantType := cmd.FlagOptions["participant_type"]
		require.False(t, hasParticipantType,
			"spec [MOD-PP-MSG-7-1] does not define participant_type; CLI flag must not exist")
		_, hasVsOperator := cmd.FlagOptions["vs_operator"]
		require.False(t, hasVsOperator,
			"spec [MOD-PP-MSG-7-1] does not define vs_operator; CLI flag must not exist")
		require.Equal(t,
			"create-root-participant [schema-id] [did] [validation-fees] [issuance-fees] [verification-fees]",
			cmd.Use,
			"create-root-participant Use string must match spec [MOD-PP-MSG-7-1] parameters")
		return
	}
	t.Fatalf("CreateRootParticipant RpcMethod not found in autocli declaration")
}

func TestAutocli_RenewParticipantOP_HasNoRoleFlag(t *testing.T) {
	opts := participant.AppModule{}.AutoCLIOptions()
	require.NotNil(t, opts)
	require.NotNil(t, opts.Tx)

	for _, cmd := range opts.Tx.RpcCommandOptions {
		if cmd.RpcMethod != "RenewParticipantOP" {
			continue
		}
		_, hasParticipantType := cmd.FlagOptions["participant_type"]
		require.False(t, hasParticipantType,
			"spec [MOD-PP-MSG-2-1] does not define participant_type; CLI flag must not exist")
		return
	}
	t.Fatalf("RenewParticipantOP RpcMethod not found in autocli declaration")
}
