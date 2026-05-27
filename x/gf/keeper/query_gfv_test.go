package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

func TestQueryGetGovernanceFrameworkVersion_NilRequest(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.GetGovernanceFrameworkVersion(ctx, nil)
	require.Error(t, err)
}

func TestQueryGetGovernanceFrameworkVersion_NotFound(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 999})
	require.Error(t, err)
}

func TestQueryGetGovernanceFrameworkVersion_PreferredLanguageFallbackWhenNoMatch(t *testing.T) {
	corp := &mockCorporation{
		view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en"},
		found: true,
	}
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
	require.NoError(t, err)

	qs := keeper.NewQueryServerImpl(k)
	// Ask for "es" which doesn't exist — must fall back to all docs.
	resp, err := qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{
		Id:                1,
		PreferredLanguage: "es",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Version.Documents, "must fall back to all docs when preferred language is absent")
}

func TestQueryListGovernanceFrameworkVersions_NilRequest(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.ListGovernanceFrameworkVersions(ctx, nil)
	require.Error(t, err)
}

func TestQueryListGovernanceFrameworkVersions_MaxSizeCap(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{
		CorporationId:   1,
		ResponseMaxSize: 2000, // exceeds 1024 limit
	})
	require.Error(t, err)
}

func TestQueryListGovernanceFrameworkVersions_EmptyResultForUnknownCorp(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{
		CorporationId: 12345,
	})
	require.NoError(t, err)
	require.Empty(t, resp.Versions)
}

func TestQueryListGovernanceFrameworkVersions_EcosystemPath(t *testing.T) {
	corp := &mockCorporation{
		view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en"},
		found: true,
	}
	eco := &mockEcosystem{
		view:  types.EcosystemView{Id: 7, CorporationID: 1, Language: "en", ActiveVersion: 0},
		found: true,
	}
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 7, 1))
	require.NoError(t, err)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{
		EcosystemId: 7,
	})
	require.NoError(t, err)
	require.Len(t, resp.Versions, 1)
	require.Equal(t, uint64(7), resp.Versions[0].EcosystemId)
}

func TestQueryListGovernanceFrameworkVersions_ActiveOnlySubjectNotFound(t *testing.T) {
	// Corporation lookup by id fails — active_only path must return NotFound.
	corp := &mockCorporation{found: false}
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{
		CorporationId: 1,
		ActiveOnly:    true,
	})
	require.Error(t, err)
}

func TestQueryListGovernanceFrameworkVersions_ActiveOnlyEcosystemSubjectNotFound(t *testing.T) {
	eco := &mockEcosystem{found: false}
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, &mockCorporation{})
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{
		EcosystemId: 7,
		ActiveOnly:  true,
	})
	require.Error(t, err)
}

func TestQueryGetGovernanceFrameworkVersion(t *testing.T) {
	t.Run("MOD-GF-QRY-1: returns GFV with docs, preferred language filter applied", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
		ms := keeper.NewMsgServerImpl(k)
		// Setup: add 2 docs (en, fr) under v1.
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.NoError(t, err)
		frMsg := validMsg(testCorp, testOperator, 0, 1)
		frMsg.DocLanguage = "fr"
		frMsg.DocUrl = "https://example.com/gf-v1-fr.html"
		_, err = ms.AddGovernanceFrameworkDocument(ctx, frMsg)
		require.NoError(t, err)

		qs := keeper.NewQueryServerImpl(k)
		// No filter — both docs.
		resp, err := qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 1})
		require.NoError(t, err)
		require.Len(t, resp.Version.Documents, 2)

		// Preferred language "fr" — only the fr doc.
		respFR, err := qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 1, PreferredLanguage: "fr"})
		require.NoError(t, err)
		require.Len(t, respFR.Version.Documents, 1)
		require.Equal(t, "fr", respFR.Version.Documents[0].Language)
	})
}

func TestQueryListGovernanceFrameworkVersions(t *testing.T) {
	t.Run("MOD-GF-QRY-2: exactly one of ecosystem_id/corporation must be set", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		qs := keeper.NewQueryServerImpl(k)
		_, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{})
		require.Error(t, err)
		_, err = qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{
			EcosystemId: 1,
			CorporationId: 1,
		})
		require.Error(t, err)
	})

	t.Run("MOD-GF-QRY-2-3: results ordered by ascending version, active_only respected", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
		ctx = ctx.WithBlockTime(time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC))
		ms := keeper.NewMsgServerImpl(k)

		// Add v1 + activate, then add v2 (not activated).
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.NoError(t, err)
		_, err = ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
		})
		require.NoError(t, err)
		corp.view.ActiveVersion = 1 // simulate ecosystem-side bump
		_, err = ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 2))
		require.NoError(t, err)

		qs := keeper.NewQueryServerImpl(k)
		all, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{CorporationId: 1})
		require.NoError(t, err)
		require.Len(t, all.Versions, 2)
		require.Equal(t, uint32(1), all.Versions[0].Version)
		require.Equal(t, uint32(2), all.Versions[1].Version)

		activeOnly, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{
			CorporationId: 1,
			ActiveOnly:  true,
		})
		require.NoError(t, err)
		require.Len(t, activeOnly.Versions, 1)
		require.Equal(t, uint32(1), activeOnly.Versions[0].Version)
	})
}
