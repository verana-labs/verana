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
	"github.com/verana-labs/verana/x/perm/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) ListPermissions(goCtx context.Context, req *types.QueryListPermissionsRequest) (*types.QueryListPermissionsResponse, error) {
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

	var permissions []types.Permission

	// [MOD-PERM-QRY-1-3] Execution
	// Collect all matching permissions
	err := k.Permission.Walk(ctx, nil, func(key uint64, perm types.Permission) (bool, error) {
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

	return &types.QueryListPermissionsResponse{
		Permissions: permissions,
	}, nil
}

func (k Keeper) GetPermission(goCtx context.Context, req *types.QueryGetPermissionRequest) (*types.QueryGetPermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PERM-QRY-2-2] Checks
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "perm ID cannot be 0")
	}

	// [MOD-PERM-QRY-2-3] Execution
	permission, err := k.Permission.Get(ctx, req.Id)
	if err != nil {
		if errors2.Is(collections.ErrNotFound, err) {
			return nil, status.Error(codes.NotFound, "perm not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get perm: %v", err))
	}

	return &types.QueryGetPermissionResponse{
		Permission: permission,
	}, nil
}

func (k Keeper) GetPermissionSession(ctx context.Context, req *types.QueryGetPermissionSessionRequest) (*types.QueryGetPermissionSessionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "session ID is required")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	session, err := k.PermissionSession.Get(sdkCtx, req.Id)
	if err != nil {
		if errors2.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Error(codes.Internal, "failed to get session")
	}

	return &types.QueryGetPermissionSessionResponse{
		Session: &session,
	}, nil
}

func (k Keeper) ListPermissionSessions(ctx context.Context, req *types.QueryListPermissionSessionsRequest) (*types.QueryListPermissionSessionsResponse, error) {
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
	var sessions []types.PermissionSession

	err := k.PermissionSession.Walk(sdkCtx, nil, func(key string, session types.PermissionSession) (bool, error) {
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

	return &types.QueryListPermissionSessionsResponse{
		Sessions: sessions,
	}, nil
}

func (k Keeper) FindPermissionsWithDID(goCtx context.Context, req *types.QueryFindPermissionsWithDIDRequest) (*types.QueryFindPermissionsWithDIDResponse, error) {
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

	// Check type - convert uint32 to PermissionType
	if req.Type == 0 {
		return nil, status.Error(codes.InvalidArgument, "perm type is required")
	}

	// Validate perm type value is in range
	permType := types.PermissionType(req.Type)
	if permType < types.PermissionType_ISSUER ||
		permType > types.PermissionType_HOLDER {
		return nil, status.Error(codes.InvalidArgument,
			fmt.Sprintf("invalid perm type value: %d, must be between 1 and 6", req.Type))
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

	// Check country code if provided
	if req.Country != "" && !isValidCountryCode(req.Country) {
		return nil, status.Error(codes.InvalidArgument, "invalid country code format")
	}

	// [MOD-PERM-QRY-3-3] Execution
	var foundPerms []types.Permission

	// Check if we need to handle the special OPEN mode case
	isOpenMode := false
	if (permType == types.PermissionType_ISSUER &&
		cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN) ||
		(permType == types.PermissionType_VERIFIER &&
			cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN) {
		isOpenMode = true
	}

	// For now, we'll scan all permissions
	err = k.Permission.Walk(ctx, nil, func(id uint64, perm types.Permission) (bool, error) {
		// Filter by schema ID
		if perm.SchemaId != req.SchemaId {
			return false, nil
		}

		// Filter by DID and type
		if perm.Did != req.Did || perm.Type != permType {
			return false, nil
		}

		// Filter by country
		if req.Country != "" && perm.Country != "" && perm.Country != req.Country {
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
		var ecosystemPerm types.Permission
		ecosystemPermFound := false

		err = k.Permission.Walk(ctx, nil, func(id uint64, perm types.Permission) (bool, error) {
			if perm.SchemaId == req.SchemaId &&
				perm.Type == types.PermissionType_ECOSYSTEM {
				// Check country compatibility
				if req.Country == "" || perm.Country == "" || perm.Country == req.Country {
					// Check time validity if "when" is specified
					if req.When == nil || isPermissionValidAtTime(perm, *req.When) {
						ecosystemPerm = perm
						ecosystemPermFound = true
						return true, nil // Stop iteration once found
					}
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
			ecosystemPerm.VpSummaryDigestSri = "OPEN_MODE_IMPLICIT_PERMISSION"
			foundPerms = append(foundPerms, ecosystemPerm)
		}
	}

	return &types.QueryFindPermissionsWithDIDResponse{
		Permissions: foundPerms,
	}, nil
}

// Helper function to check if a perm is valid at a specific time
func isPermissionValidAtTime(perm types.Permission, when time.Time) bool {
	// Check effective_from
	if perm.EffectiveFrom != nil && when.Before(*perm.EffectiveFrom) {
		return false
	}

	// Check effective_until
	if perm.EffectiveUntil != nil && !when.Before(*perm.EffectiveUntil) {
		return false
	}

	// Check revoked
	if perm.Revoked != nil && !when.Before(*perm.Revoked) {
		return false
	}

	// Check slashed
	if perm.SlashedDeposit > 0 {
		return false
	}

	return true
}

func isValidDID(did string) bool {
	// Basic DID validation regex
	// This is a simplified version and may need to be expanded based on specific DID method requirements
	didRegex := regexp.MustCompile(`^did:[a-zA-Z0-9]+:[a-zA-Z0-9._-]+$`)
	return didRegex.MatchString(did)
}

func isValidCountryCode(code string) bool {
	// Basic check for ISO 3166-1 alpha-2 format
	match, _ := regexp.MatchString(`^[A-Z]{2}$`, code)
	return match
}

func (k Keeper) FindBeneficiaries(goCtx context.Context, req *types.QueryFindBeneficiariesRequest) (*types.QueryFindBeneficiariesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-PERM-QRY-4-2] Find Beneficiaries checks
	// if issuer_perm_id and verifier_perm_id are unset then MUST abort
	if req.IssuerPermId == 0 && req.VerifierPermId == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one of issuer_perm_id or verifier_perm_id must be provided")
	}

	var issuerPerm, verifierPerm *types.Permission

	// if issuer_perm_id is specified, load issuer_perm from issuer_perm_id, Permission MUST exist and MUST be a valid permission
	if req.IssuerPermId != 0 {
		perm, err := k.Permission.Get(ctx, req.IssuerPermId)
		if err != nil {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("issuer permission not found: %v", err))
		}

		// MUST be a valid permission
		if err := IsValidPermission(perm, perm.Country, ctx.BlockTime()); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("issuer permission is not valid: %v", err))
		}

		issuerPerm = &perm
	}

	// if verifier_perm_id is specified, load verifier_perm from verifier_perm_id, Permission MUST exist and MUST be a valid permission
	if req.VerifierPermId != 0 {
		perm, err := k.Permission.Get(ctx, req.VerifierPermId)
		if err != nil {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("verifier permission not found: %v", err))
		}

		// MUST be a valid permission
		if err := IsValidPermission(perm, perm.Country, ctx.BlockTime()); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("verifier permission is not valid: %v", err))
		}

		verifierPerm = &perm
	}

	// [MOD-PERM-QRY-4-3] Find Beneficiaries execution
	// create Set found_perm_set
	foundPermMap := make(map[uint64]types.Permission)

	// if issuer_perm is not null
	if issuerPerm != nil {
		// set current_perm = issuer_perm
		currentPerm := issuerPerm

		// while current_perm.validator_perm_id is not null
		for currentPerm.ValidatorPermId != 0 {
			// set current_perm to loaded permission from current_perm.validator_perm_id
			perm, err := k.Permission.Get(ctx, currentPerm.ValidatorPermId)
			if err != nil {
				return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get permission: %v", err))
			}
			currentPerm = &perm

			// if current_perm.revoked IS NULL AND current_perm.slashed IS NULL, Add current_perm to found_perm_set
			// Note: SlashedDeposit > 0 indicates the permission has been slashed
			if currentPerm.Revoked == nil && currentPerm.SlashedDeposit == 0 {
				foundPermMap[currentPerm.Id] = *currentPerm
			}
		}
	}

	// Additionally, if verifier_perm is not null
	if verifierPerm != nil {
		// if issuer_perm is not null, add issuer_perm to found_perm_set
		if issuerPerm != nil {
			if issuerPerm.Revoked == nil && issuerPerm.SlashedDeposit == 0 {
				foundPermMap[issuerPerm.Id] = *issuerPerm
			}
		}

		// set current_perm = verifier_perm
		currentPerm := verifierPerm

		// while verifier_perm.validator_perm_id is not null
		for currentPerm.ValidatorPermId != 0 {
			// set current_perm to loaded permission from current_perm.validator_perm_id
			perm, err := k.Permission.Get(ctx, currentPerm.ValidatorPermId)
			if err != nil {
				return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get permission: %v", err))
			}
			currentPerm = &perm

			// if current_perm.revoked IS NULL AND current_perm.slashed IS NULL, Add current_perm to found_perm_set
			if currentPerm.Revoked == nil && currentPerm.SlashedDeposit == 0 {
				foundPermMap[currentPerm.Id] = *currentPerm
			}
		}
	}

	// Convert map to array
	permissions := make([]types.Permission, 0, len(foundPermMap))
	for _, perm := range foundPermMap {
		permissions = append(permissions, perm)
	}

	return &types.QueryFindBeneficiariesResponse{
		Permissions: permissions,
	}, nil
}
