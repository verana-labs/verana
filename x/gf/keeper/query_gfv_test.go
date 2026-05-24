package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

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
		require.Equal(t, int32(1), all.Versions[0].Version)
		require.Equal(t, int32(2), all.Versions[1].Version)

		activeOnly, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{
			CorporationId: 1,
			ActiveOnly:  true,
		})
		require.NoError(t, err)
		require.Len(t, activeOnly.Versions, 1)
		require.Equal(t, int32(1), activeOnly.Versions[0].Version)
	})
}
