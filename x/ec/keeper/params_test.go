package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/ec/types"
)

func TestGetParams(t *testing.T) {
	k, _, ctx := setupMsgServer(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
