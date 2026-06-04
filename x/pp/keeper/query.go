package keeper

import (
	"context"
	errors2 "errors"
	"fmt"
	"sort"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/util/validation"
	credentialschematypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/pp/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) ListParticipants(goCtx context.Context, req *types.QueryListParticipantsRequest) (*types.QueryListParticipantsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PP-QRY-1-2] Checks
	// Validate response_max_size
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64 // Default value
	}
	if req.ResponseMaxSize < 1 || req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1,024")
	}

	var participants []types.Participant

	// [MOD-PP-QRY-1-3] Execution
	// Collect all matching participants
	err := k.Participant.Walk(ctx, nil, func(key uint64, participant types.Participant) (bool, error) {
		// Apply modified_after filter if provided
		if req.ModifiedAfter != nil && !participant.Modified.After(*req.ModifiedAfter) {
			return false, nil
		}

		participants = append(participants, participant)
		return len(participants) >= int(req.ResponseMaxSize), nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Sort by modified time ascending
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].Modified.Before(*participants[j].Modified)
	})

	return &types.QueryListParticipantsResponse{
		Participants: participants,
	}, nil
}

func (k Keeper) GetParticipant(goCtx context.Context, req *types.QueryGetParticipantRequest) (*types.QueryGetParticipantResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PP-QRY-2-2] Checks
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "participant ID cannot be 0")
	}

	// [MOD-PP-QRY-2-3] Execution
	participant, err := k.Participant.Get(ctx, req.Id)
	if err != nil {
		if errors2.Is(collections.ErrNotFound, err) {
			return nil, status.Error(codes.NotFound, "participant not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get participant: %v", err))
	}

	return &types.QueryGetParticipantResponse{
		Participant: participant,
	}, nil
}

func (k Keeper) GetParticipantSession(ctx context.Context, req *types.QueryGetParticipantSessionRequest) (*types.QueryGetParticipantSessionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID is required")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	session, err := k.ParticipantSession.Get(sdkCtx, req.Id)
	if err != nil {
		if errors2.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Error(codes.Internal, "failed to get session")
	}

	return &types.QueryGetParticipantSessionResponse{
		Session: &session,
	}, nil
}

func (k Keeper) ListParticipantSessions(ctx context.Context, req *types.QueryListParticipantSessionsRequest) (*types.QueryListParticipantSessionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// Validate response_max_size
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64 // Default value
	}
	if req.ResponseMaxSize < 1 || req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1,024")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var sessions []types.ParticipantSession

	err := k.ParticipantSession.Walk(sdkCtx, nil, func(key string, session types.ParticipantSession) (bool, error) {
		// Apply modified_after filter if provided
		if req.ModifiedAfter != nil && !session.Modified.After(*req.ModifiedAfter) {
			return false, nil
		}

		sessions = append(sessions, session)
		return len(sessions) >= int(req.ResponseMaxSize), nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list sessions")
	}

	// Sort by modified time ascending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.Before(*sessions[j].Modified)
	})

	return &types.QueryListParticipantSessionsResponse{
		Sessions: sessions,
	}, nil
}

func (k Keeper) FindParticipantsWithDID(goCtx context.Context, req *types.QueryFindParticipantsWithDIDRequest) (*types.QueryFindParticipantsWithDIDResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PP-QRY-3-2] Checks
	if req.Did == "" {
		return nil, status.Error(codes.InvalidArgument, "DID is required")
	}
	if !validation.IsValidDID(req.Did) {
		return nil, status.Error(codes.InvalidArgument, "invalid DID format")
	}

	// Check type - convert uint32 to ParticipantRole
	if req.Role == 0 {
		return nil, status.Error(codes.InvalidArgument, "participant type is required")
	}

	// Validate participant type value is in range
	participantType := types.ParticipantRole(req.Role)
	if participantType < types.ParticipantRole_ISSUER ||
		participantType > types.ParticipantRole_HOLDER {
		return nil, status.Error(codes.InvalidArgument,
			fmt.Sprintf("invalid participant type value: %d, must be between 1 and 6", req.Role))
	}

	// Check schema ID
	if req.SchemaId == 0 {
		return nil, status.Error(codes.InvalidArgument, "schema ID is required")
	}

	// Check schema exists and get schema details
	cs, err := k.credentialSchemaKeeper.GetCredentialSchemaById(ctx, req.SchemaId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("credential schema not found: %v", err))
	}

	// [MOD-PP-QRY-3-3] Execution
	// country was removed from the Participant entity and from this query per spec v4 draft 13.
	var foundParticipants []types.Participant

	// Check if we need to handle the special OPEN mode case
	isOpenMode := false
	if (participantType == types.ParticipantRole_ISSUER &&
		cs.IssuerOnboardingMode == credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN) ||
		(participantType == types.ParticipantRole_VERIFIER &&
			cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN) {
		isOpenMode = true
	}

	// For now, we'll scan all participants
	err = k.Participant.Walk(ctx, nil, func(id uint64, participant types.Participant) (bool, error) {
		// Filter by schema ID
		if participant.SchemaId != req.SchemaId {
			return false, nil
		}

		// Filter by DID and type
		if participant.Did != req.Did || participant.Role != participantType {
			return false, nil
		}

		// If "when" is not specified, add all matching participants
		if req.When == nil {
			foundParticipants = append(foundParticipants, participant)
			return false, nil
		}

		// Filter by time validity
		if isParticipantValidAtTime(participant, *req.When) {
			foundParticipants = append(foundParticipants, participant)
		}

		return false, nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to query participants: %v", err))
	}

	// If we're in OPEN mode and didn't find any explicit participants,
	// check if there's an ECOSYSTEM participant that handles fees
	if isOpenMode && len(foundParticipants) == 0 {
		// Find ECOSYSTEM participant for this schema
		var ecosystemParticipant types.Participant
		ecosystemParticipantFound := false

		err = k.Participant.Walk(ctx, nil, func(id uint64, participant types.Participant) (bool, error) {
			if participant.SchemaId == req.SchemaId &&
				participant.Role == types.ParticipantRole_ECOSYSTEM {
				// Check time validity if "when" is specified
				if req.When == nil || isParticipantValidAtTime(participant, *req.When) {
					ecosystemParticipant = participant
					ecosystemParticipantFound = true
					return true, nil // Stop iteration once found
				}
			}
			return false, nil
		})

		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to query ECOSYSTEM participant: %v", err))
		}

		// In OPEN mode, if we found an ECOSYSTEM participant, we can consider the DID
		// authorized even without an explicit participant record
		if ecosystemParticipantFound {
			// Include a note in the response that this is an implicit participant in OPEN mode
			ecosystemParticipant.OpSummaryDigest = "OPEN_MODE_IMPLICIT_PERMISSION"
			foundParticipants = append(foundParticipants, ecosystemParticipant)
		}
	}

	return &types.QueryFindParticipantsWithDIDResponse{
		Participants: foundParticipants,
	}, nil
}

// Helper function to check if a participant is valid at a specific time
// This should align with IsValidParticipant logic for consistency
func isParticipantValidAtTime(participant types.Participant, when time.Time) bool {
	// Check repaid (REPAID state)
	if participant.Repaid != nil {
		return false
	}

	// Check slashed (SLASHED state) - use timestamp as per spec
	if participant.Slashed != nil {
		return false
	}

	// Check revoked (REVOKED state)
	// Spec: "else if `revoked` is lower than now(), => `participant_state` is `REVOKED`"
	// This means revoked < now(), so we check when.After(*participant.Revoked)
	if participant.Revoked != nil && when.After(*participant.Revoked) {
		return false
	}

	// Check expired (EXPIRED state)
	if participant.EffectiveUntil != nil && !when.Before(*participant.EffectiveUntil) {
		return false
	}

	// Check FUTURE state
	if participant.EffectiveFrom != nil && when.Before(*participant.EffectiveFrom) {
		return false
	}

	// Check INACTIVE state (effective_from is null)
	if participant.EffectiveFrom == nil {
		return false
	}

	// At this point, participant is ACTIVE
	return true
}

func (k Keeper) FindBeneficiaries(goCtx context.Context, req *types.QueryFindBeneficiariesRequest) (*types.QueryFindBeneficiariesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PP-QRY-4-2] Find Beneficiaries checks
	// if issuer_participant_id and verifier_participant_id are unset then MUST abort
	if req.IssuerParticipantId == 0 && req.VerifierParticipantId == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one of issuer_participant_id or verifier_participant_id must be provided")
	}

	var issuerParticipant, verifierParticipant *types.Participant

	// if issuer_participant_id is specified, load issuer_participant from issuer_participant_id, Participant MUST exist and MUST be a valid participant
	if req.IssuerParticipantId != 0 {
		participant, err := k.Participant.Get(ctx, req.IssuerParticipantId)
		if err != nil {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("issuer participant not found: %v", err))
		}

		// MUST be a valid participant
		if err := IsValidParticipant(participant, ctx.BlockTime()); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("issuer participant is not valid: %v", err))
		}

		issuerParticipant = &participant
	}

	// if verifier_participant_id is specified, load verifier_participant from verifier_participant_id, Participant MUST exist and MUST be a valid participant
	if req.VerifierParticipantId != 0 {
		participant, err := k.Participant.Get(ctx, req.VerifierParticipantId)
		if err != nil {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("verifier participant not found: %v", err))
		}

		// MUST be a valid participant
		if err := IsValidParticipant(participant, ctx.BlockTime()); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("verifier participant is not valid: %v", err))
		}

		verifierParticipant = &participant
	}

	// [MOD-PP-QRY-4-3] Find Beneficiaries execution
	// create Set found_participant_set
	foundParticipantMap := make(map[uint64]types.Participant)

	// if issuer_participant is not null
	if issuerParticipant != nil {
		// set current_participant = issuer_participant
		currentParticipant := issuerParticipant

		// while current_participant.validator_participant_id is not null
		for currentParticipant.ValidatorParticipantId != 0 {
			// set current_participant to loaded participant from current_participant.validator_participant_id
			participant, err := k.Participant.Get(ctx, currentParticipant.ValidatorParticipantId)
			if err != nil {
				return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get participant: %v", err))
			}
			currentParticipant = &participant

			// if current_participant.revoked IS NULL AND current_participant.slashed IS NULL, Add current_participant to found_participant_set
			// Note: SlashedDeposit > 0 indicates the participant has been slashed
			if currentParticipant.Revoked == nil && currentParticipant.Slashed == nil {
				foundParticipantMap[currentParticipant.Id] = *currentParticipant
			}
		}
	}

	// Additionally, if verifier_participant is not null
	if verifierParticipant != nil {
		// if issuer_participant is not null, add issuer_participant to found_participant_set
		if issuerParticipant != nil {
			if issuerParticipant.Revoked == nil && issuerParticipant.Slashed == nil {
				foundParticipantMap[issuerParticipant.Id] = *issuerParticipant
			}
		}

		// set current_participant = verifier_participant
		currentParticipant := verifierParticipant

		// while verifier_participant.validator_participant_id is not null
		for currentParticipant.ValidatorParticipantId != 0 {
			// set current_participant to loaded participant from current_participant.validator_participant_id
			participant, err := k.Participant.Get(ctx, currentParticipant.ValidatorParticipantId)
			if err != nil {
				return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get participant: %v", err))
			}
			currentParticipant = &participant

			// if current_participant.revoked IS NULL AND current_participant.slashed IS NULL, Add current_participant to found_participant_set
			if currentParticipant.Revoked == nil && currentParticipant.Slashed == nil {
				foundParticipantMap[currentParticipant.Id] = *currentParticipant
			}
		}
	}

	// Convert map to array
	participants := make([]types.Participant, 0, len(foundParticipantMap))
	for _, participant := range foundParticipantMap {
		participants = append(participants, participant)
	}

	return &types.QueryFindBeneficiariesResponse{
		Participants: participants,
	}, nil
}
