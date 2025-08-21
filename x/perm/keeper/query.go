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
	if permType < types.PermissionType_PERMISSION_TYPE_ISSUER ||
		permType > types.PermissionType_PERMISSION_TYPE_HOLDER {
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
	if (permType == types.PermissionType_PERMISSION_TYPE_ISSUER &&
		cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN) ||
		(permType == types.PermissionType_PERMISSION_TYPE_VERIFIER &&
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
				perm.Type == types.PermissionType_PERMISSION_TYPE_ECOSYSTEM {
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

	// Check terminated
	if perm.Terminated != nil && !when.Before(*perm.Terminated) {
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

	// [MOD-PERM-QRY-4-2] Checks
	// At least one of issuer_perm_id or verifier_perm_id must be provided
	if req.IssuerPermId == 0 && req.VerifierPermId == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one of issuer_perm_id or verifier_perm_id must be provided")
	}

	var issuerPerm, verifierPerm *types.Permission
	var schemaID uint64

	// Load issuer perm if specified
	if req.IssuerPermId != 0 {
		perm, err := k.Permission.Get(ctx, req.IssuerPermId)
		if err != nil {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("issuer perm not found: %v", err))
		}
		issuerPerm = &perm
		schemaID = perm.SchemaId
	}

	// Load verifier perm if specified
	if req.VerifierPermId != 0 {
		perm, err := k.Permission.Get(ctx, req.VerifierPermId)
		if err != nil {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("verifier perm not found: %v", err))
		}
		verifierPerm = &perm
		if schemaID == 0 {
			schemaID = perm.SchemaId
		}
	}

	// Get schema to check perm management mode
	cs, err := k.credentialSchemaKeeper.GetCredentialSchemaById(ctx, schemaID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("credential schema not found: %v", err))
	}

	// Check if schema is configured with OPEN perm management mode
	isIssuerOpenMode := false
	isVerifierOpenMode := false

	if cs.IssuerPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
		isIssuerOpenMode = true
	}

	if cs.VerifierPermManagementMode == credentialschematypes.CredentialSchemaPermManagementMode_OPEN {
		isVerifierOpenMode = true
	}

	// Handle OPEN mode case
	// If in OPEN mode, we need to find the ECOSYSTEM perm for this schema
	if (req.IssuerPermId != 0 && isIssuerOpenMode) || (req.VerifierPermId != 0 && isVerifierOpenMode) {
		// Find ECOSYSTEM perm for this schema
		var ecosystemPerm types.Permission
		ecosystemPermFound := false

		err = k.Permission.Walk(ctx, nil, func(id uint64, perm types.Permission) (bool, error) {
			if perm.SchemaId == schemaID &&
				perm.Type == types.PermissionType_PERMISSION_TYPE_ECOSYSTEM &&
				perm.Revoked == nil && perm.Terminated == nil {
				ecosystemPerm = perm
				ecosystemPermFound = true
				return true, nil // Stop iteration once found
			}
			return false, nil
		})

		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to query ECOSYSTEM perm: %v", err))
		}

		if ecosystemPermFound {
			// For OPEN mode, the only beneficiary is the ECOSYSTEM perm
			permissions := []types.Permission{ecosystemPerm}
			return &types.QueryFindBeneficiariesResponse{
				Permissions: permissions,
			}, nil
		}
	}

	// If not in OPEN mode or ECOSYSTEM perm not found, proceed with normal hierarchy traversal
	// [MOD-PERM-QRY-4-3] Execution
	// Use a map to implement the set functionality
	foundPermMap := make(map[uint64]types.Permission)

	// Process issuer perm hierarchy
	if issuerPerm != nil {
		// Start with the validator of issuer_perm
		if issuerPerm.ValidatorPermId != 0 {
			currentPermID := issuerPerm.ValidatorPermId

			// Traverse up the validator chain
			for currentPermID != 0 {
				currentPerm, err := k.Permission.Get(ctx, currentPermID)
				if err != nil {
					return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get perm: %v", err))
				}

				// Add to set if not revoked or terminated
				if currentPerm.Revoked == nil && currentPerm.Terminated == nil && currentPerm.SlashedDeposit == 0 {
					foundPermMap[currentPermID] = currentPerm
				}

				// Move up to the next validator
				currentPermID = currentPerm.ValidatorPermId
			}
		}
	}

	// Process verifier perm hierarchy
	if verifierPerm != nil {
		// First add issuer_perm to the set if it exists
		if issuerPerm != nil && issuerPerm.Revoked == nil && issuerPerm.Terminated == nil && issuerPerm.SlashedDeposit == 0 {
			foundPermMap[req.IssuerPermId] = *issuerPerm
		}

		// Start with the validator of verifier_perm
		if verifierPerm.ValidatorPermId != 0 {
			currentPermID := verifierPerm.ValidatorPermId

			// Traverse up the validator chain
			for currentPermID != 0 {
				currentPerm, err := k.Permission.Get(ctx, currentPermID)
				if err != nil {
					return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get perm: %v", err))
				}

				// Add to set if not revoked or terminated
				if currentPerm.Revoked == nil && currentPerm.Terminated == nil && currentPerm.SlashedDeposit == 0 {
					foundPermMap[currentPermID] = currentPerm
				}

				// Move up to the next validator
				currentPermID = currentPerm.ValidatorPermId
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
