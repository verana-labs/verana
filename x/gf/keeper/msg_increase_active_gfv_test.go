package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

func TestIncreaseActiveGovernanceFrameworkVersion(t *testing.T) {
	t.Run("MOD-GF-MSG-2: happy path activates next version for Corporation", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		var newActive int32
		corp.setFn = func(_ uint64, v int32) error { newActive = v; return nil }

		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
		ctx = ctx.WithBlockTime(now)

		// Setup: add a v1 GFV+GFD via MSG-1 (active_version stays 0).
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.NoError(t, err)

		// Now bump active_version 0 → 1.
		_, err = ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
			EcosystemId: 0,
		})
		require.NoError(t, err)
		require.Equal(t, int32(1), newActive)

		// GFV active_since should now be set.
		_ = k.GFVersion.Walk(ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
			require.False(t, gfv.ActiveSince.IsZero())
			require.Equal(t, now, gfv.ActiveSince)
			return false, nil
		})
	})

	t.Run("MOD-GF-MSG-2-2-1: aborts when no next-version GFV exists", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		// No MSG-1 first — nothing to activate.
		_, err := ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
		})
		require.ErrorIs(t, err, types.ErrNoActivatableVersion)
	})

	t.Run("MOD-GF-MSG-2-2-1: aborts when default-language doc missing", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		// MSG-1 adds an "fr" doc — default language is "en".
		_, err := ms.AddGovernanceFrameworkDocument(ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation:  testCorp,
			Operator:     testOperator,
			EcosystemId:  0,
			DocLanguage:  "fr",
			DocUrl:       testURL,
			DocDigestSri: testDigest,
			Version:      1,
		})
		require.NoError(t, err)

		_, err = ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
		})
		require.ErrorIs(t, err, types.ErrMissingDefaultLang)
	})
}
