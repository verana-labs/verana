package keeper_test

import (
	"context"
	"testing"
	"time"

	cerrors "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

const (
	testCorp     = "cosmos1corp00000000000000000000000000000abc"
	testOperator = "cosmos1op0000000000000000000000000000000abc"
	testDigest   = "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26"
	testURL      = "https://example.com/gf-v1.html"
)

// mockDelegation implements types.DelegationKeeper.
type mockDelegation struct{ err error }

func (m mockDelegation) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	return m.err
}

// mockEcosystem implements types.EcosystemKeeper.
type mockEcosystem struct {
	view  types.EcosystemView
	found bool
	setFn func(uint64, uint32) error
}

func (m *mockEcosystem) GetEcosystemView(_ context.Context, _ uint64) (types.EcosystemView, bool) {
	return m.view, m.found
}
func (m *mockEcosystem) SetEcosystemActiveVersion(_ context.Context, id uint64, v uint32) error {
	if m.setFn != nil {
		return m.setFn(id, v)
	}
	return nil
}

// mockCorporation implements types.CorporationKeeper.
type mockCorporation struct {
	view  types.CorporationView
	found bool
	setFn func(uint64, uint32) error
}

func (m *mockCorporation) ResolveByPolicyAddress(_ context.Context, _ string) (types.CorporationView, bool) {
	return m.view, m.found
}
func (m *mockCorporation) GetByID(_ context.Context, _ uint64) (types.CorporationView, bool) {
	return m.view, m.found
}
func (m *mockCorporation) SetActiveVersion(_ context.Context, id uint64, v uint32) error {
	if m.setFn != nil {
		return m.setFn(id, v)
	}
	return nil
}

func validMsg(corp, op string, ecoID uint64, version uint32) *types.MsgAddGovernanceFrameworkDocument {
	return &types.MsgAddGovernanceFrameworkDocument{
		Corporation:  corp,
		Operator:     op,
		EcosystemId:  ecoID,
		DocLanguage:  "en",
		DocUrl:       testURL,
		DocDigestSri: testDigest,
		Version:      version,
	}
}

func TestAddGovernanceFrameworkDocument(t *testing.T) {
	t.Run("MOD-GF-MSG-1: happy path adds GFV+GFD to Corporation subject", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
		ctx = ctx.WithBlockTime(now)

		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.NoError(t, err)

		var gfvCount int
		_ = k.GFVersion.Walk(ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
			gfvCount++
			require.Equal(t, uint64(1), gfv.CorporationId)
			require.Zero(t, gfv.EcosystemId)
			require.Equal(t, uint32(1), gfv.Version)
			require.True(t, gfv.ActiveSince.IsZero())
			return false, nil
		})
		require.Equal(t, 1, gfvCount)
	})

	t.Run("MOD-GF-MSG-1-2-1: AUTHZ-CHECK-1 failure aborts", func(t *testing.T) {
		corp := &mockCorporation{view: types.CorporationView{Id: 1, PolicyAddress: testCorp}, found: true}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t,
			mockDelegation{err: cerrors.Wrap(sdkerrors.ErrUnauthorized, "unauthorized")},
			eco, corp,
		)
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.Error(t, err)
	})

	t.Run("MOD-GF-MSG-1-2-1: ecosystem not controlled by signer aborts", func(t *testing.T) {
		// AUTHZ-CHECK-5 resolves signing corp to co.id = 1; ecosystem #1's CorporationID = 999 (different) → not controlled.
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		eco := &mockEcosystem{
			view:  types.EcosystemView{Id: 1, CorporationID: 999, Language: "en"},
			found: true,
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 1, 1))
		require.ErrorIs(t, err, types.ErrSubjectNotControlled)
	})

	t.Run("MOD-GF-MSG-1-2-1: version must be > active_version", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 2},
			found: true,
		}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)
		// Set up a v1 first so MaxV becomes 1 and v2 attempt fails on active_version check.
		_, _ = ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		_, _ = ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 2))
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 2))
		require.ErrorIs(t, err, types.ErrInvalidVersion)
	})

	t.Run("MOD-GF-MSG-1-3: replaces existing GFD for same language", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.NoError(t, err)

		updated := validMsg(testCorp, testOperator, 0, 1)
		updated.DocUrl = "https://example.com/gf-v1-updated.html"
		_, err = ms.AddGovernanceFrameworkDocument(ctx, updated)
		require.NoError(t, err)

		var gfdCount int
		_ = k.GFDocument.Walk(ctx, nil, func(_ uint64, d types.GovernanceFrameworkDocument) (bool, error) {
			gfdCount++
			require.Equal(t, "https://example.com/gf-v1-updated.html", d.Url)
			return false, nil
		})
		require.Equal(t, 1, gfdCount, "GFD count must be 1 — same-language doc must be replaced, not appended")
	})

	t.Run("MOD-GF-MSG-1-2-1: ecosystem not found aborts", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en"},
			found: true,
		}
		eco := &mockEcosystem{found: false} // GetEcosystemView returns (zero, false)
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 42, 1))
		require.ErrorIs(t, err, types.ErrSubjectNotFound)
	})

	t.Run("MOD-GF-MSG-1-2-1: corporation not registered aborts (StubCorporationKeeper behaviour)", func(t *testing.T) {
		// found: false — simulates the StubCorporationKeeper that returns
		// (zero, false) from ResolveByPolicyAddress until MOD-CO (#303) lands.
		corp := &mockCorporation{found: false}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.ErrorIs(t, err, types.ErrSubjectNotFound)
	})

	t.Run("MOD-GF-MSG-1-1: ValidateBasic rejects malformed Msg before AUTHZ-CHECK", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en"},
			found: true,
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
		ms := keeper.NewMsgServerImpl(k)
		bad := validMsg(testCorp, testOperator, 0, 1)
		bad.DocUrl = "not a url"
		_, err := ms.AddGovernanceFrameworkDocument(ctx, bad)
		require.ErrorIs(t, err, types.ErrInvalidURL)
	})

	t.Run("MOD-GF-MSG-1-3: target version exists (in-place update of existing GFV)", func(t *testing.T) {
		// Verify the maxVersionFor `hasTarget=true` branch: passing the same
		// version twice does NOT mint a new GFV id; it reuses the existing one
		// and just upserts the document.
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en"},
			found: true,
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
		ms := keeper.NewMsgServerImpl(k)

		// Add v1 with "en" doc.
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.NoError(t, err)
		// Add a SECOND v1 with "fr" doc — should reuse the same GFV.
		frMsg := validMsg(testCorp, testOperator, 0, 1)
		frMsg.DocLanguage = "fr"
		frMsg.DocUrl = "https://example.com/v1-fr.html"
		_, err = ms.AddGovernanceFrameworkDocument(ctx, frMsg)
		require.NoError(t, err)

		var gfvCount, gfdCount int
		_ = k.GFVersion.Walk(ctx, nil, func(_ uint64, _ types.GovernanceFrameworkVersion) (bool, error) {
			gfvCount++
			return false, nil
		})
		_ = k.GFDocument.Walk(ctx, nil, func(_ uint64, _ types.GovernanceFrameworkDocument) (bool, error) {
			gfdCount++
			return false, nil
		})
		require.Equal(t, 1, gfvCount, "v1 must be reused, not duplicated")
		require.Equal(t, 2, gfdCount, "two language docs must coexist under the same GFV")
	})

	t.Run("MOD-GF-MSG-1-3: ecosystem subject creates GFV with ecosystem_id set", func(t *testing.T) {
		// AUTHZ-CHECK-5 resolves signing corporation to co.id = 1; ecosystem #7 is controlled by corp id 1.
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		eco := &mockEcosystem{
			view:  types.EcosystemView{Id: 7, CorporationID: 1, Language: "en"},
			found: true,
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 7, 1))
		require.NoError(t, err)

		_ = k.GFVersion.Walk(ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
			require.Equal(t, uint64(7), gfv.EcosystemId)
			require.Equal(t, uint64(0), gfv.CorporationId)
			return false, nil
		})
	})
}
