package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/pp/types"
)

// The (did, corporation_id) consistency invariant (spec MOD-PP-MSG-1-2-1 /
// 7-2-1 / 14-2-1): all Participant entries sharing a did MUST belong to the
// same corporation. Enforced at create time in the three create paths.

func TestStartParticipantOP_DIDCorporationConsistency(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)

	corpA := sdk.AccAddress([]byte("did_start_corp_a____")).String()
	corpB := sdk.AccAddress([]byte("did_start_corp_b____")).String()
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)
	validatorID := vsoaValidator(t, k, sdkCtx, trkKeeper, corpA, now, past)

	// corpB already controls sharedDID.
	sharedDID := "did:example:start-shared"
	corpBID := trkKeeper.RegisterCorp(corpB)
	_, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: sharedDID,
		CorporationId: corpBID, Created: &now, Modified: &now,
		OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
	})
	require.NoError(t, err)
	delKeeper.Reset()

	// Bad path: corpA tries to start a participant under a did corpB controls.
	_, err = ms.StartParticipantOP(ctx, &types.MsgStartParticipantOP{
		Corporation: corpA, Operator: corpA, Role: types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorID, Did: sharedDID,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)

	// Happy path: a did not controlled by anyone else is accepted.
	_, err = ms.StartParticipantOP(ctx, &types.MsgStartParticipantOP{
		Corporation: corpA, Operator: corpA, Role: types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorID, Did: "did:example:start-fresh",
	})
	require.NoError(t, err)
}

func TestCreateRootParticipant_DIDCorporationConsistency(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)
	farFuture := now.Add(24 * time.Hour)

	corpA := sdk.AccAddress([]byte("did_root_corp_a_____")).String()
	corpB := sdk.AccAddress([]byte("did_root_corp_b_____")).String()
	ecoDID := "did:example:root-ecosystem"
	trID := trkKeeper.CreateMockEcosystem(corpA, ecoDID)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	// corpB already controls sharedDID.
	sharedDID := "did:example:root-shared"
	corpBID := trkKeeper.RegisterCorp(corpB)
	_, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: sharedDID,
		CorporationId: corpBID, Created: &now, Modified: &now,
		OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
	})
	require.NoError(t, err)

	// Bad path: corpA creates a root participant under a did corpB controls.
	_, err = ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
		Corporation: corpA, Operator: corpA, SchemaId: 1, Did: sharedDID,
		EffectiveFrom: &future, EffectiveUntil: &farFuture,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)

	// Happy path: corpA creates a root participant under its own ecosystem did.
	_, err = ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
		Corporation: corpA, Operator: corpA, SchemaId: 1, Did: ecoDID,
		EffectiveFrom: &future, EffectiveUntil: &farFuture,
	})
	require.NoError(t, err)
}

func TestSelfCreateParticipant_DIDCorporationConsistency(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)
	farFuture := now.Add(360 * 24 * time.Hour)

	corpA := sdk.AccAddress([]byte("did_self_corp_a_____")).String()
	corpB := sdk.AccAddress([]byte("did_self_corp_b_____")).String()
	ecoDID := "did:example:self-ecosystem"
	trID := trkKeeper.CreateMockEcosystem(corpA, ecoDID)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

	// Active ECOSYSTEM validator participant under corpA.
	validatorID, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: ecoDID,
		CorporationId: trkKeeper.RegisterCorp(corpA), Created: &now, Modified: &now,
		OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
	})
	require.NoError(t, err)

	// corpB already controls sharedDID.
	sharedDID := "did:example:self-shared"
	corpBID := trkKeeper.RegisterCorp(corpB)
	_, err = k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId: 1, Role: types.ParticipantRole_ISSUER, Did: sharedDID,
		CorporationId: corpBID, Created: &now, Modified: &now,
		OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
	})
	require.NoError(t, err)
	delKeeper.Reset()

	// Bad path: corpA self-creates under a did corpB controls.
	_, err = ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
		Corporation: corpA, Operator: corpA, Role: types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorID, Did: sharedDID,
		EffectiveFrom: &future, EffectiveUntil: &farFuture,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)

	// Happy path: a fresh did is accepted.
	_, err = ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
		Corporation: corpA, Operator: corpA, Role: types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorID, Did: "did:example:self-fresh",
		EffectiveFrom: &future, EffectiveUntil: &farFuture,
	})
	require.NoError(t, err)
}
