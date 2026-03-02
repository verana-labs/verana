package keeper

import (
	"context"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/cs/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) ListCredentialSchemas(goCtx context.Context, req *types.QueryListCredentialSchemasRequest) (*types.QueryListCredentialSchemasResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate response_max_size
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize > 1024 {
		return nil, fmt.Errorf("response_max_size must be between 1 and 1024")
	}

	var schemas []types.CredentialSchema
	err := k.CredentialSchema.Walk(ctx, nil, func(key uint64, schema types.CredentialSchema) (bool, error) {
		// Filter by trust registry if specified
		if req.TrId != 0 && schema.TrId != req.TrId {
			return false, nil
		}

		// Filter by modification time if specified
		if req.ModifiedAfter != nil && !schema.Modified.After(*req.ModifiedAfter) {
			return false, nil
		}

		// Filter archived entries if only_active is set
		if req.OnlyActive && schema.Archived != nil {
			return false, nil
		}

		// Filter by issuer_perm_management_mode if specified
		if req.IssuerPermManagementMode != types.CredentialSchemaPermManagementMode_MODE_UNSPECIFIED &&
			schema.IssuerPermManagementMode != req.IssuerPermManagementMode {
			return false, nil
		}

		// Filter by verifier_perm_management_mode if specified
		if req.VerifierPermManagementMode != types.CredentialSchemaPermManagementMode_MODE_UNSPECIFIED &&
			schema.VerifierPermManagementMode != req.VerifierPermManagementMode {
			return false, nil
		}

		// Ensure canonical $id is present in the JSON schema
		schemaWithCanonicalID, err := types.EnsureCanonicalID(schema.JsonSchema, ctx.ChainID(), schema.Id)
		if err != nil {
			// Log error but don't fail the entire query
			k.Logger().Error("failed to ensure canonical ID", "schema_id", schema.Id, "error", err)
		} else {
			schema.JsonSchema = schemaWithCanonicalID
		}

		schemas = append(schemas, schema)
		return false, nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by modified timestamp descending (spec: results MUST be ordered by modified DESC)
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Modified.After(schemas[j].Modified)
	})

	// Apply response_max_size limit after sorting
	if len(schemas) > int(req.ResponseMaxSize) {
		schemas = schemas[:req.ResponseMaxSize]
	}

	return &types.QueryListCredentialSchemasResponse{
		Schemas: schemas,
	}, nil
}

func (k Keeper) GetCredentialSchema(goCtx context.Context, req *types.QueryGetCredentialSchemaRequest) (*types.QueryGetCredentialSchemaResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	schema, err := k.CredentialSchema.Get(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "credential schema not found")
	}

	// Ensure canonical $id is present in the JSON schema
	schemaWithCanonicalID, err := types.EnsureCanonicalID(schema.JsonSchema, ctx.ChainID(), schema.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to ensure canonical ID: %v", err))
	}
	schema.JsonSchema = schemaWithCanonicalID

	return &types.QueryGetCredentialSchemaResponse{
		Schema: schema,
	}, nil
}

func (k Keeper) RenderJsonSchema(goCtx context.Context, req *types.QueryRenderJsonSchemaRequest) (*types.QueryRenderJsonSchemaResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	schema, err := k.CredentialSchema.Get(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "credential schema not found")
	}

	// Ensure canonical $id is present in the JSON schema
	schemaWithCanonicalID, err := types.EnsureCanonicalID(schema.JsonSchema, ctx.ChainID(), schema.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to ensure canonical ID: %v", err))
	}

	// Apply full JCS canonicalization (RFC 8785): sorted keys, no insignificant whitespace
	canonicalized, err := types.CanonicalizeJCS(schemaWithCanonicalID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to JCS-canonicalize schema: %v", err))
	}

	return &types.QueryRenderJsonSchemaResponse{
		Schema: canonicalized,
	}, nil
}
