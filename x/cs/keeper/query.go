package keeper

import (
	"context"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana-blockchain/x/cs/types"
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
		if req.ModifiedAfter != nil && schema.Modified.Before(*req.ModifiedAfter) {
			return false, nil
		}

		schemas = append(schemas, schema)
		return len(schemas) >= int(req.ResponseMaxSize), nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by created timestamp ascending
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Created.Before(schemas[j].Created)
	})

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

	return &types.QueryRenderJsonSchemaResponse{
		Schema: schema.JsonSchema,
	}, nil
}
