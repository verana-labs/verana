package lib

import (
	"context"
	"fmt"
	"time"

	permtypes "github.com/verana-labs/verana/x/perm/types"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana/x/cs/types"
	trtypes "github.com/verana-labs/verana/x/tr/types"
)

// QueryTrustRegistry gets a trust registry by ID
func QueryTrustRegistry(client cosmosclient.Client, ctx context.Context, trID uint64) (*trtypes.QueryGetTrustRegistryResponse, error) {
	queryClient := trtypes.NewQueryClient(client.Context())
	return queryClient.GetTrustRegistry(ctx, &trtypes.QueryGetTrustRegistryRequest{
		TrId: trID,
	})
}

// ListTrustRegistries lists all trust registries
func ListTrustRegistries(client cosmosclient.Client, ctx context.Context, responseMaxSize uint32) (*trtypes.QueryListTrustRegistriesResponse, error) {
	queryClient := trtypes.NewQueryClient(client.Context())
	return queryClient.ListTrustRegistries(ctx, &trtypes.QueryListTrustRegistriesRequest{
		ResponseMaxSize: responseMaxSize,
	})
}

// ListCredentialSchemas lists credential schemas
func ListCredentialSchemas(client cosmosclient.Client, ctx context.Context, modifiedAfter time.Time, responseMaxSize uint32) (*cschema.QueryListCredentialSchemasResponse, error) {
	csClient := cschema.NewQueryClient(client.Context())
	return csClient.ListCredentialSchemas(ctx, &cschema.QueryListCredentialSchemasRequest{
		ModifiedAfter:   &modifiedAfter,
		ResponseMaxSize: responseMaxSize,
	})
}

// QueryCredentialSchema queries for a credential schema by ID
func QueryCredentialSchema(client cosmosclient.Client, ctx context.Context, csID uint64) (*cschema.QueryGetCredentialSchemaResponse, error) {
	csQueryClient := cschema.NewQueryClient(client.Context())
	return csQueryClient.GetCredentialSchema(ctx, &cschema.QueryGetCredentialSchemaRequest{
		Id: csID,
	})
}

// QueryPermission queries for a permission by ID
func QueryPermission(client cosmosclient.Client, ctx context.Context, permID uint64) (*permtypes.QueryGetPermissionResponse, error) {
	permQueryClient := permtypes.NewQueryClient(client.Context())
	return permQueryClient.GetPermission(ctx, &permtypes.QueryGetPermissionRequest{Id: permID})
}

// ListPermissions lists all permissions
func ListPermissions(client cosmosclient.Client, ctx context.Context) ([]permtypes.Permission, error) {
	permQueryClient := permtypes.NewQueryClient(client.Context())
	resp, err := permQueryClient.ListPermissions(ctx, &permtypes.QueryListPermissionsRequest{
		ResponseMaxSize: 1024,
	})
	if err != nil {
		return nil, err
	}
	return resp.Permissions, nil
}

// VerifyTrustRegistry verifies a trust registry exists with expected properties
func VerifyTrustRegistry(client cosmosclient.Client, ctx context.Context, trID uint64, expectedDID string) bool {
	resp, err := QueryTrustRegistry(client, ctx, trID)
	if err != nil {
		fmt.Printf("❌ Trust Registry verification failed: %v\n", err)
		return false
	}

	// Verify DID matches what we expect
	if resp.TrustRegistry.Did != expectedDID {
		fmt.Printf("❌ Trust Registry verification failed: Expected DID %s, got %s\n",
			expectedDID, resp.TrustRegistry.Did)
		return false
	}

	fmt.Printf("✅ Verified Trust Registry ID %d exists with expected DID %s\n",
		trID, resp.TrustRegistry.Did)
	return true
}

// VerifyCredentialSchema verifies a credential schema exists with expected properties
func VerifyCredentialSchema(client cosmosclient.Client, ctx context.Context, csID uint64, expectedTrID uint64) bool {
	resp, err := QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		fmt.Printf("❌ Credential Schema verification failed: %v\n", err)
		return false
	}

	// Verify Trust Registry ID matches what we expect
	if resp.Schema.TrId != expectedTrID {
		fmt.Printf("❌ Credential Schema verification failed: Expected Trust Registry ID %d, got %d\n",
			expectedTrID, resp.Schema.TrId)
		return false
	}

	fmt.Printf("✅ Verified Credential Schema ID %d exists with expected Trust Registry ID %d\n",
		csID, resp.Schema.TrId)
	return true
}

// VerifyPermission verifies a permission exists with expected properties
func VerifyPermission(client cosmosclient.Client, ctx context.Context, permID uint64, expectedSchemaID uint64, expectedType string) bool {
	resp, err := QueryPermission(client, ctx, permID)
	if err != nil {
		fmt.Printf("❌ Permission verification failed: %v\n", err)
		return false
	}

	// Verify Schema ID and permission type match what we expect
	if resp.Permission.SchemaId != expectedSchemaID {
		fmt.Printf("❌ Permission verification failed: Expected Schema ID %d, got %d\n",
			expectedSchemaID, resp.Permission.SchemaId)
		return false
	}

	permType := permtypes.PermissionType_name[int32(resp.Permission.Type)]
	if permType != expectedType {
		fmt.Printf("❌ Permission verification failed: Expected type %s, got %s\n",
			expectedType, permType)
		return false
	}

	fmt.Printf("✅ Verified Permission ID %d exists with expected Schema ID %d and type %s\n",
		permID, resp.Permission.SchemaId, permType)
	return true
}

