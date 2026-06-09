package keeper_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/cs/keeper"
)

// [MOD-CS-MSG-1] json_schema is stored preserving the submitter's property
// order (the spec defines NO JCS / alphabetical canonicalization). The off-chain
// indexer relies on the documented field order, so storage MUST NOT sort keys.
func TestCreateCredentialSchema_PreservesPropertyOrder(t *testing.T) {
	k, ms, mockTrk, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	authority := sdk.AccAddress([]byte("order_authority_____")).String()
	operator := sdk.AccAddress([]byte("order_operator______")).String()
	trID := mockTrk.CreateMockEcosystem(authority, "did:example:order")

	// Properties in deliberately NON-alphabetical order.
	schema := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "OrderCredential",
  "description": "order test",
  "type": "object",
  "properties": {
    "zebra": { "type": "string" },
    "apple": { "type": "string" },
    "mango": { "type": "string" }
  }
}`
	msg := keeper.CreateMsgWithValidityPeriods(authority, operator, trID, schema, 365, 365, 180, 180, 180, 2, 2, 2, 1, "tu", "sha256")
	require.NoError(t, msg.ValidateBasic())

	resp, err := ms.CreateCredentialSchema(ctx, msg)
	require.NoError(t, err)

	stored, err := k.CredentialSchema.Get(sdkCtx, resp.Id)
	require.NoError(t, err)

	// Submitted order zebra < apple < mango MUST be preserved; alphabetical
	// canonicalization would reorder to apple < mango < zebra.
	zi := strings.Index(stored.JsonSchema, `"zebra"`)
	ai := strings.Index(stored.JsonSchema, `"apple"`)
	mi := strings.Index(stored.JsonSchema, `"mango"`)
	require.True(t, zi >= 0 && ai >= 0 && mi >= 0, "all properties present")
	require.Less(t, zi, ai, "zebra must precede apple (order preserved, not alphabetized)")
	require.Less(t, ai, mi, "apple must precede mango")

	// Canonical $id is colon-form and injected first.
	require.Contains(t, stored.JsonSchema, ":cs:")
}
