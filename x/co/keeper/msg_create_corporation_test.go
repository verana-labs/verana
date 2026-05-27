package keeper_test

import (
	"errors"
	"testing"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/co/keeper"
	"github.com/verana-labs/verana/x/co/types"
)

func anyDecisionPolicy(t *testing.T) *cdctypes.Any {
	t.Helper()
	a, err := cdctypes.NewAnyWithValue(&group.ThresholdDecisionPolicy{
		Threshold: "1",
		Windows:   &group.DecisionPolicyWindows{VotingPeriod: 0},
	})
	require.NoError(t, err)
	return a
}

// Valid bech32 fixtures used by the keeper tests (deterministically generated).
const (
	tkSigner = "cosmos1hfyt5r4f3rnu5gqrgfwr4446zgn00gdj0nn7dx"
	tkMember = "cosmos1jpjwc8r9y5xqpj7q3w9c2qhda0ednjcmxvtna4"
	tkCorp   = "cosmos14wcc52lpsxwuxxhqjxrhvuumhm0xr6z247un93"
	tkOp     = "cosmos1fvz0kp4jfseea3zyduu78dd5yqcwrarwtxthjn"
	tkPolicy = "cosmos1uyy0v2yu28dayjljvwe5h8w6pa8l8mwlx6589y"
)

func validCreateMsg(t *testing.T) *types.MsgCreateCorporation {
	t.Helper()
	return &types.MsgCreateCorporation{
		Signer:         tkSigner,
		Members:        []types.Member{{Address: tkMember, Weight: "1"}},
		DecisionPolicy: anyDecisionPolicy(t),
		Did:            "did:example:1",
		Language:       "en",
		DocUrl:         "https://example.com/cgf.pdf",
		DocDigestSri:   "sha256-aGVsbG8=",
	}
}

func TestCreateCorporation_Happy(t *testing.T) {
	grp := &mockGroup{policy: tkPolicy}
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, gf)
	ms := keeper.NewMsgServerImpl(k)

	msg := validCreateMsg(t)
	resp, err := ms.CreateCorporation(ctx, msg)
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp.CorporationId)
	require.Equal(t, tkPolicy, resp.PolicyAddress)

	// Group call shape: group_policy_as_admin and members converted correctly.
	require.Equal(t, 1, grp.callsCnt)
	require.True(t, grp.gotReq.GroupPolicyAsAdmin)
	require.Equal(t, msg.Signer, grp.gotReq.Admin)
	require.Len(t, grp.gotReq.Members, 1)
	require.Equal(t, tkMember, grp.gotReq.Members[0].Address)
	require.Equal(t, "1", grp.gotReq.Members[0].Weight)

	// GF seed called with the right args.
	require.Equal(t, 1, gf.createCalls)
	require.Equal(t, uint64(1), gf.createArgs.corpID)
	require.Equal(t, "en", gf.createArgs.language)
	require.Equal(t, msg.DocUrl, gf.createArgs.docURL)
	require.Equal(t, msg.DocDigestSri, gf.createArgs.docDigestSRI)

	// Corporation persisted with active_version=1 and reverse indexes set.
	co, err := k.Corporation.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, uint32(1), co.ActiveVersion)
	require.Equal(t, tkPolicy, co.PolicyAddress)
	require.Equal(t, msg.Did, co.Did)

	id, err := k.CorporationByPolicyAddr.Get(ctx, tkPolicy)
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)
	id, err = k.CorporationByDID.Get(ctx, msg.Did)
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)

	// Event emitted.
	evs := ctx.EventManager().Events()
	var seen bool
	for _, e := range evs {
		if e.Type == types.EventTypeCreateCorporation {
			seen = true
		}
	}
	require.True(t, seen, "create_corporation event must be emitted")
}

func TestCreateCorporation_ValidateBasicError(t *testing.T) {
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.CreateCorporation(ctx, &types.MsgCreateCorporation{}) // empty: bails out
	require.Error(t, err)
}

func TestCreateCorporation_GroupFailureIsPropagated(t *testing.T) {
	grp := &mockGroup{err: errors.New("group boom")}
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, gf)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "group boom")
	// Corporation must NOT be persisted on group failure.
	_, err = k.Corporation.Get(ctx, 1)
	require.Error(t, err)
	require.Equal(t, 0, gf.createCalls, "GF seed must not run when group creation failed")
}

func TestCreateCorporation_DuplicatePolicyAddress(t *testing.T) {
	grp := &mockGroup{policy: tkPolicy}
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, gf)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)

	// Second call returns the SAME policy_address (so it's pre-bound) but a
	// different DID; uniqueness check on policy_address must fire.
	msg2 := validCreateMsg(t)
	msg2.Did = "did:example:2"
	_, err = ms.CreateCorporation(ctx, msg2)
	require.ErrorIs(t, err, types.ErrPolicyAddressAlreadyBound)
}

func TestCreateCorporation_DuplicateDID(t *testing.T) {
	gf := &mockGF{}
	// First call uses default policy, second call returns a different policy.
	grp := &mockGroup{policy: tkPolicy}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, gf)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)

	grp.policy = "cosmos1z7z43tkdpkzddxw5t3krh8jjx8amghmhacdj6u" // policy-test-2
	_, err = ms.CreateCorporation(ctx, validCreateMsg(t))      // same DID, different policy
	require.ErrorIs(t, err, types.ErrDIDAlreadyExists)
}

// TestCreateCorporation_DIDConflictAbortsBeforeGroupCall pins the
// precondition-first ordering: a duplicate-DID submission must short-circuit
// without invoking x/group. This is the spec-mandated ordering and avoids
// wasted group-creation work.
func TestCreateCorporation_DIDConflictAbortsBeforeGroupCall(t *testing.T) {
	grp := &mockGroup{policy: tkPolicy}
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, gf)
	ms := keeper.NewMsgServerImpl(k)

	// Seed one Corporation.
	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)
	require.Equal(t, 1, grp.callsCnt, "first call hit x/group")

	// Second call with same DID — must abort with ErrDIDAlreadyExists BEFORE
	// reaching x/group. callsCnt MUST stay at 1.
	_, err = ms.CreateCorporation(ctx, validCreateMsg(t))
	require.ErrorIs(t, err, types.ErrDIDAlreadyExists)
	require.Equal(t, 1, grp.callsCnt, "x/group must not be called when did is already bound")
	require.Equal(t, 1, gf.createCalls, "GF seed must not be called for the failed second attempt")
}

func TestCreateCorporation_GFSeedFailureBubblesUp(t *testing.T) {
	grp := &mockGroup{policy: tkPolicy}
	gf := &mockGF{createErr: errors.New("gf boom")}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, gf)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "gf boom")
	// Group + corp index/state were created before the GF call. The msg handler
	// returns the error → the SDK rolls back the entire tx in production. In
	// this unit harness no rollback runs, so we only assert the error surfaced.
}
