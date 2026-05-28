package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/co/types"
)

func validCorp(id uint64, addr, did string) types.Corporation {
	return types.Corporation{
		Id:            id,
		PolicyAddress: addr,
		Did:           did,
		Created:       time.Unix(0, 0),
		Modified:      time.Unix(0, 0),
		Language:      "en",
		ActiveVersion: 1,
	}
}

func TestGenesis_Default(t *testing.T) {
	gs := types.DefaultGenesis()
	require.NotNil(t, gs)
	require.NoError(t, gs.Validate())
	require.Empty(t, gs.Corporations)
}

func TestGenesis_Validate(t *testing.T) {
	good := types.GenesisState{
		Params: types.DefaultParams(),
		Corporations: []types.Corporation{
			validCorp(1, "cosmos1aaa", "did:example:1"),
			validCorp(2, "cosmos1bbb", "did:example:2"),
		},
	}
	require.NoError(t, good.Validate())

	cases := map[string]types.GenesisState{
		"id zero": {Params: types.DefaultParams(), Corporations: []types.Corporation{validCorp(0, "cosmos1a", "did:example:1")}},
		"dup id": {Params: types.DefaultParams(), Corporations: []types.Corporation{
			validCorp(1, "cosmos1a", "did:example:1"),
			validCorp(1, "cosmos1b", "did:example:2"),
		}},
		"dup addr": {Params: types.DefaultParams(), Corporations: []types.Corporation{
			validCorp(1, "cosmos1a", "did:example:1"),
			validCorp(2, "cosmos1a", "did:example:2"),
		}},
		"dup did": {Params: types.DefaultParams(), Corporations: []types.Corporation{
			validCorp(1, "cosmos1a", "did:example:1"),
			validCorp(2, "cosmos1b", "did:example:1"),
		}},
		"empty addr": {Params: types.DefaultParams(), Corporations: []types.Corporation{
			func() types.Corporation { c := validCorp(1, "", "did:example:1"); return c }(),
		}},
		"bad did": {Params: types.DefaultParams(), Corporations: []types.Corporation{
			validCorp(1, "cosmos1a", "not-a-did"),
		}},
		"bad lang": {Params: types.DefaultParams(), Corporations: []types.Corporation{
			func() types.Corporation { c := validCorp(1, "cosmos1a", "did:example:1"); c.Language = "!!"; return c }(),
		}},
		"zero created": {Params: types.DefaultParams(), Corporations: []types.Corporation{
			func() types.Corporation { c := validCorp(1, "cosmos1a", "did:example:1"); c.Created = time.Time{}; return c }(),
		}},
		"zero modified": {Params: types.DefaultParams(), Corporations: []types.Corporation{
			func() types.Corporation { c := validCorp(1, "cosmos1a", "did:example:1"); c.Modified = time.Time{}; return c }(),
		}},
		"modified before created": {Params: types.DefaultParams(), Corporations: []types.Corporation{
			func() types.Corporation {
				c := validCorp(1, "cosmos1a", "did:example:1")
				c.Created = time.Unix(100, 0)
				c.Modified = time.Unix(50, 0)
				return c
			}(),
		}},
	}
	for name, gs := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, gs.Validate())
		})
	}
}

// TestGenesis_Validate_TimestampErrorKind pins the error kind for the new
// timestamp-invariant cases so a future refactor can't silently swap it.
func TestGenesis_Validate_TimestampErrorKind(t *testing.T) {
	mk := func(mut func(*types.Corporation)) types.GenesisState {
		c := validCorp(1, "cosmos1a", "did:example:1")
		mut(&c)
		return types.GenesisState{Params: types.DefaultParams(), Corporations: []types.Corporation{c}}
	}
	require.ErrorIs(t, mk(func(c *types.Corporation) { c.Created = time.Time{} }).Validate(), types.ErrInvalidTimestamp)
	require.ErrorIs(t, mk(func(c *types.Corporation) { c.Modified = time.Time{} }).Validate(), types.ErrInvalidTimestamp)
	require.ErrorIs(t, mk(func(c *types.Corporation) {
		c.Created = time.Unix(100, 0)
		c.Modified = time.Unix(50, 0)
	}).Validate(), types.ErrInvalidTimestamp)
}
