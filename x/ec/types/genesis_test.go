package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/ec/types"
)

func validEco(id uint64, did string, corpID uint64) types.Ecosystem {
	return types.Ecosystem{
		Id:            id,
		Did:           did,
		CorporationId: corpID,
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
	require.Empty(t, gs.Ecosystems)
}

func TestGenesis_Validate_Happy(t *testing.T) {
	good := types.GenesisState{
		Params: types.DefaultParams(),
		Ecosystems: []types.Ecosystem{
			validEco(1, "did:example:1", 1),
			validEco(2, "did:example:2", 1),         // same corp, different did → OK
			validEco(3, "did:example:1", 1),         // same corp+same did → OK (shared DID across corp-controlled ecosystems)
			validEco(4, "did:example:other", 2),     // different corp → OK
		},
		Counters: []types.Counter{{EntityType: "ec", Value: 4}},
	}
	require.NoError(t, good.Validate())
}

func TestGenesis_Validate_Rejects(t *testing.T) {
	cases := map[string]types.GenesisState{
		"id zero": {Params: types.DefaultParams(), Ecosystems: []types.Ecosystem{validEco(0, "did:example:1", 1)}},
		"dup id": {Params: types.DefaultParams(), Ecosystems: []types.Ecosystem{
			validEco(1, "did:example:1", 1),
			validEco(1, "did:example:2", 1),
		}},
		"zero corp_id": {Params: types.DefaultParams(), Ecosystems: []types.Ecosystem{validEco(1, "did:example:1", 0)}},
		"bad did": {Params: types.DefaultParams(), Ecosystems: []types.Ecosystem{validEco(1, "not-a-did", 1)}},
		"bad lang": {Params: types.DefaultParams(), Ecosystems: []types.Ecosystem{
			func() types.Ecosystem { e := validEco(1, "did:example:1", 1); e.Language = "!!"; return e }(),
		}},
		"zero created": {Params: types.DefaultParams(), Ecosystems: []types.Ecosystem{
			func() types.Ecosystem { e := validEco(1, "did:example:1", 1); e.Created = time.Time{}; return e }(),
		}},
		"zero modified": {Params: types.DefaultParams(), Ecosystems: []types.Ecosystem{
			func() types.Ecosystem { e := validEco(1, "did:example:1", 1); e.Modified = time.Time{}; return e }(),
		}},
		"modified before created": {Params: types.DefaultParams(), Ecosystems: []types.Ecosystem{
			func() types.Ecosystem {
				e := validEco(1, "did:example:1", 1)
				e.Created = time.Unix(100, 0)
				e.Modified = time.Unix(50, 0)
				return e
			}(),
		}},
		"zero active_version": {Params: types.DefaultParams(), Ecosystems: []types.Ecosystem{
			func() types.Ecosystem { e := validEco(1, "did:example:1", 1); e.ActiveVersion = 0; return e }(),
		}},
		"(did,corp) consistency violation": {Params: types.DefaultParams(), Ecosystems: []types.Ecosystem{
			validEco(1, "did:example:shared", 1),
			validEco(2, "did:example:shared", 2), // SAME did, DIFFERENT corp_id → violates invariant
		}},
		"unknown counter type": {Params: types.DefaultParams(), Counters: []types.Counter{{EntityType: "tr", Value: 1}}},
		"ec counter below max id": {
			Params:     types.DefaultParams(),
			Ecosystems: []types.Ecosystem{validEco(5, "did:example:1", 1)},
			Counters:   []types.Counter{{EntityType: "ec", Value: 3}}, // 3 < 5 → reject
		},
	}
	for name, gs := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, gs.Validate())
		})
	}
}

func TestGenesis_Validate_TimestampErrorKind(t *testing.T) {
	mk := func(mut func(*types.Ecosystem)) types.GenesisState {
		e := validEco(1, "did:example:1", 1)
		mut(&e)
		return types.GenesisState{Params: types.DefaultParams(), Ecosystems: []types.Ecosystem{e}}
	}
	require.ErrorIs(t, mk(func(e *types.Ecosystem) { e.Created = time.Time{} }).Validate(), types.ErrInvalidTimestamp)
	require.ErrorIs(t, mk(func(e *types.Ecosystem) { e.Modified = time.Time{} }).Validate(), types.ErrInvalidTimestamp)
	require.ErrorIs(t, mk(func(e *types.Ecosystem) {
		e.Created = time.Unix(100, 0)
		e.Modified = time.Unix(50, 0)
	}).Validate(), types.ErrInvalidTimestamp)
}

func TestGenesis_Validate_DIDConflictErrorKind(t *testing.T) {
	gs := types.GenesisState{
		Params: types.DefaultParams(),
		Ecosystems: []types.Ecosystem{
			validEco(1, "did:example:shared", 1),
			validEco(2, "did:example:shared", 2),
		},
	}
	require.ErrorIs(t, gs.Validate(), types.ErrDIDOwnershipConflict)
}
