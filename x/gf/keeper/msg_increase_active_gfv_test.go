package keeper_test

import (
	"errors"
	"testing"
	"time"

	cerrors "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
		var newActive uint32
		corp.setFn = func(_ uint64, v uint32) error { newActive = v; return nil }

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
		require.Equal(t, uint32(1), newActive)

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

	t.Run("MOD-GF-MSG-2-1: ValidateBasic rejects missing corporation", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: "",
			Operator:    testOperator,
		})
		require.Error(t, err)
	})

	t.Run("MOD-GF-MSG-2-2-1: AUTHZ-CHECK-1 failure aborts before subject resolution", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en"},
			found: true,
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t,
			mockDelegation{err: cerrors.Wrap(sdkerrors.ErrUnauthorized, "unauthorized")},
			&mockEcosystem{}, corp,
		)
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
		})
		require.Error(t, err)
	})

	t.Run("MOD-GF-MSG-2-2-1: corporation not registered aborts (Stub case)", func(t *testing.T) {
		corp := &mockCorporation{found: false}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
		})
		require.ErrorIs(t, err, types.ErrSubjectNotFound)
	})

	t.Run("MOD-GF-MSG-2-2-1: ecosystem not controlled by signer aborts", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en"},
			found: true,
		}
		eco := &mockEcosystem{
			view:  types.EcosystemView{Id: 5, CorporationID: 999, Language: "en", ActiveVersion: 0},
			found: true,
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
			EcosystemId: 5,
		})
		require.ErrorIs(t, err, types.ErrSubjectNotControlled)
	})

	t.Run("MOD-GF-MSG-2-3: SetActiveVersion error from CorporationKeeper propagates", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
			setFn: func(_ uint64, _ uint32) error {
				return errors.New("forced setter failure")
			},
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
		ms := keeper.NewMsgServerImpl(k)
		ctx = ctx.WithBlockTime(time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC))

		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.NoError(t, err)

		_, err = ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "forced setter failure")
	})

	t.Run("MOD-GF-MSG-2-3: SetEcosystemActiveVersion error from EcosystemKeeper propagates", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en"},
			found: true,
		}
		eco := &mockEcosystem{
			view:  types.EcosystemView{Id: 7, CorporationID: 1, Language: "en", ActiveVersion: 0},
			found: true,
			setFn: func(_ uint64, _ uint32) error {
				return errors.New("forced eco setter failure")
			},
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)
		ctx = ctx.WithBlockTime(time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC))

		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 7, 1))
		require.NoError(t, err)

		_, err = ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
			EcosystemId: 7,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "forced eco setter failure")
	})
}
