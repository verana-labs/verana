package keeper

import (
	"context"
	errors2 "errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

	// [MOD-PERM-QRY-1-2] Checks
	// Validate response_max_size
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64 // Default value
	}
	if req.ResponseMaxSize < 1 || req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be between 1 and 1,024")
	}

	var permissions []types.Participant

	// [MOD-PERM-QRY-1-3] Execution
	// Collect all matching permissions
	err := k.Participant.Walk(ctx, nil, func(key uint64, perm types.Participant) (bool, error) {
		// Apply modified_after filter if provided
		if req.ModifiedAfter != nil && !perm.Modified.After(*req.ModifiedAfter) {
			return false, nil
		}

		permissions = append(permissions, perm)
		return len(permissions) >= int(req.ResponseMaxSize), nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Sort by modified time ascending
	sort.Slice(permissions, func(i, j int) bool {
		return permissions[i].Modified.Before(*permissions[j].Modified)
	})

	return &types.QueryListParticipantsResponse{
		Participants: permissions,
	}, nil
}

func (k Keeper) GetParticipant(goCtx context.Context, req *types.QueryGetParticipantRequest) (*types.QueryGetParticipantResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PERM-QRY-2-2] Checks
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "perm ID cannot be 0")
	}

	// [MOD-PERM-QRY-2-3] Execution
	permission, err := k.Participant.Get(ctx, req.Id)
	if err != nil {
		if errors2.Is(collections.ErrNotFound, err) {
			return nil, status.Error(codes.NotFound, "perm not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get perm: %v", err))
	}

	return &types.QueryGetParticipantResponse{
		Participant: permission,
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

	// [MOD-PERM-QRY-3-2] Checks
	if req.Did == "" {
		return nil, status.Error(codes.InvalidArgument, "DID is required")
	}
	if !isValidDID(req.Did) {
		return nil, status.Error(codes.InvalidArgument, "invalid DID format")
	}

	// Check type - convert uint32 to ParticipantRole
	if req.Role == 0 {
		return nil, status.Error(codes.InvalidArgument, "perm type is required")
	}

	// Validate perm type value is in range
	permType := types.ParticipantRole(req.Role)
	if permType < types.ParticipantRole_ISSUER ||
		permType > types.ParticipantRole_HOLDER {
		return nil, status.Error(codes.InvalidArgument,
			fmt.Sprintf("invalid perm type value: %d, must be between 1 and 6", req.Role))
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

	// [MOD-PERM-QRY-3-3] Execution
	// country was removed from the Participant entity and from this query per spec v4 draft 13.
	var foundPerms []types.Participant

	// Check if we need to handle the special OPEN mode case
	isOpenMode := false
	if (permType == types.ParticipantRole_ISSUER &&
		cs.IssuerOnboardingMode == credentialschematypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN) ||
		(permType == types.ParticipantRole_VERIFIER &&
			cs.VerifierOnboardingMode == credentialschematypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN) {
		isOpenMode = true
	}

	// For now, we'll scan all permissions
	err = k.Participant.Walk(ctx, nil, func(id uint64, perm types.Participant) (bool, error) {
		// Filter by schema ID
		if perm.SchemaId != req.SchemaId {
			return false, nil
		}

		// Filter by DID and type
		if perm.Did != req.Did || perm.Role != permType {
			return false, nil
		}

		// If "when" is not specified, add all matching permissions
		if req.When == nil {
			foundPerms = append(foundPerms, perm)
			return false, nil
		}

		// Filter by time validity
		if isPermissionValidAtTime(perm, *req.When) {
			foundPerms = append(foundPerms, perm)
		}

		return false, nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to query permissions: %v", err))
	}

	// If we're in OPEN mode and didn't find any explicit permissions,
	// check if there's an ECOSYSTEM perm that handles fees
	if isOpenMode && len(foundPerms) == 0 {
		// Find ECOSYSTEM perm for this schema
		var ecosystemPerm types.Participant
		ecosystemPermFound := false

		err = k.Participant.Walk(ctx, nil, func(id uint64, perm types.Participant) (bool, error) {
			if perm.SchemaId == req.SchemaId &&
				perm.Role == types.ParticipantRole_ECOSYSTEM {
				// Check time validity if "when" is specified
				if req.When == nil || isPermissionValidAtTime(perm, *req.When) {
					ecosystemPerm = perm
					ecosystemPermFound = true
					return true, nil // Stop iteration once found
				}
			}
			return false, nil
		})

		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to query ECOSYSTEM perm: %v", err))
		}

		// In OPEN mode, if we found an ECOSYSTEM perm, we can consider the DID
		// authorized even without an explicit perm record
		if ecosystemPermFound {
			// Include a note in the response that this is an implicit perm in OPEN mode
			ecosystemPerm.OpSummaryDigest = "OPEN_MODE_IMPLICIT_PERMISSION"
			foundPerms = append(foundPerms, ecosystemPerm)
		}
	}

	return &types.QueryFindParticipantsWithDIDResponse{
		Participants: foundPerms,
	}, nil
}

// Helper function to check if a perm is valid at a specific time
// This should align with IsValidPermission logic for consistency
func isPermissionValidAtTime(perm types.Participant, when time.Time) bool {
	// Check repaid (REPAID state)
	if perm.Repaid != nil {
		return false
	}

	// Check slashed (SLASHED state) - use timestamp as per spec
	if perm.Slashed != nil {
		return false
	}

	// Check revoked (REVOKED state)
	// Spec: "else if `revoked` is lower than now(), => `perm_state` is `REVOKED`"
	// This means revoked < now(), so we check when.After(*perm.Revoked)
	if perm.Revoked != nil && when.After(*perm.Revoked) {
		return false
	}

	// Check expired (EXPIRED state)
	if perm.EffectiveUntil != nil && !when.Before(*perm.EffectiveUntil) {
		return false
	}

	// Check FUTURE state
	if perm.EffectiveFrom != nil && when.Before(*perm.EffectiveFrom) {
		return false
	}

	// Check INACTIVE state (effective_from is null)
	if perm.EffectiveFrom == nil {
		return false
	}

	// At this point, permission is ACTIVE
	return true
}

func isValidDID(did string) bool {
	// DID validation regex following W3C DID specification
	// Format: did:<method-name>:<method-specific-id>
	// Method-specific-id can contain alphanumeric, dots, underscores, hyphens, colons, and slashes
	didRegex := regexp.MustCompile(`^did:[a-zA-Z0-9]+:[a-zA-Z0-9._:/%\-]+$`)
	return didRegex.MatchString(did)
}

func (k Keeper) FindBeneficiaries(goCtx context.Context, req *types.QueryFindBeneficiariesRequest) (*types.QueryFindBeneficiariesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PERM-QRY-4-2] Find Beneficiaries checks
	// if issuer_perm_id and verifier_perm_id are unset then MUST abort
	if req.IssuerParticipantId == 0 && req.VerifierParticipantId == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one of issuer_perm_id or verifier_perm_id must be provided")
	}

	var issuerPerm, verifierPerm *types.Participant

	// if issuer_perm_id is specified, load issuer_perm from issuer_perm_id, Participant MUST exist and MUST be a valid permission
	if req.IssuerParticipantId != 0 {
		perm, err := k.Participant.Get(ctx, req.IssuerParticipantId)
		if err != nil {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("issuer permission not found: %v", err))
		}

		// MUST be a valid permission
		if err := IsValidPermission(perm, ctx.BlockTime()); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("issuer permission is not valid: %v", err))
		}

		issuerPerm = &perm
	}

	// if verifier_perm_id is specified, load verifier_perm from verifier_perm_id, Participant MUST exist and MUST be a valid permission
	if req.VerifierParticipantId != 0 {
		perm, err := k.Participant.Get(ctx, req.VerifierParticipantId)
		if err != nil {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("verifier permission not found: %v", err))
		}

		// MUST be a valid permission
		if err := IsValidPermission(perm, ctx.BlockTime()); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("verifier permission is not valid: %v", err))
		}

		verifierPerm = &perm
	}

	// [MOD-PERM-QRY-4-3] Find Beneficiaries execution
	// create Set found_perm_set
	foundPermMap := make(map[uint64]types.Participant)

	// if issuer_perm is not null
	if issuerPerm != nil {
		// set current_perm = issuer_perm
		currentPerm := issuerPerm

		// while current_perm.validator_participant_id is not null
		for currentPerm.ValidatorParticipantId != 0 {
			// set current_perm to loaded permission from current_perm.validator_participant_id
			perm, err := k.Participant.Get(ctx, currentPerm.ValidatorParticipantId)
			if err != nil {
				return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get permission: %v", err))
			}
			currentPerm = &perm

			// if current_perm.revoked IS NULL AND current_perm.slashed IS NULL, Add current_perm to found_perm_set
			// Note: SlashedDeposit > 0 indicates the permission has been slashed
			if currentPerm.Revoked == nil && currentPerm.Slashed == nil {
				foundPermMap[currentPerm.Id] = *currentPerm
			}
		}
	}

	// Additionally, if verifier_perm is not null
	if verifierPerm != nil {
		// if issuer_perm is not null, add issuer_perm to found_perm_set
		if issuerPerm != nil {
			if issuerPerm.Revoked == nil && issuerPerm.Slashed == nil {
				foundPermMap[issuerPerm.Id] = *issuerPerm
			}
		}

		// set current_perm = verifier_perm
		currentPerm := verifierPerm

		// while verifier_perm.validator_participant_id is not null
		for currentPerm.ValidatorParticipantId != 0 {
			// set current_perm to loaded permission from current_perm.validator_participant_id
			perm, err := k.Participant.Get(ctx, currentPerm.ValidatorParticipantId)
			if err != nil {
				return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get permission: %v", err))
			}
			currentPerm = &perm

			// if current_perm.revoked IS NULL AND current_perm.slashed IS NULL, Add current_perm to found_perm_set
			if currentPerm.Revoked == nil && currentPerm.Slashed == nil {
				foundPermMap[currentPerm.Id] = *currentPerm
			}
		}
	}

	// Convert map to array
	permissions := make([]types.Participant, 0, len(foundPermMap))
	for _, perm := range foundPermMap {
		permissions = append(permissions, perm)
	}

	return &types.QueryFindBeneficiariesResponse{
		Participants: permissions,
	}, nil
}
