package lib

import (
	"context"
	"fmt"
	"time"

	permtypes "github.com/verana-labs/verana/x/pp/types"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cotypes "github.com/verana-labs/verana/x/co/types"
	cschema "github.com/verana-labs/verana/x/cs/types"
	ectypes "github.com/verana-labs/verana/x/ec/types"
	gftypes "github.com/verana-labs/verana/x/gf/types"
)

// QueryEcosystem gets an ecosystem by ID
func QueryEcosystem(client cosmosclient.Client, ctx context.Context, trID uint64) (*ectypes.QueryGetEcosystemResponse, error) {
	queryClient := ectypes.NewQueryClient(client.Context())
	return queryClient.GetEcosystem(ctx, &ectypes.QueryGetEcosystemRequest{
		Id: trID,
	})
}

// ListTrustRegistries lists all ecosystems (kept for backward-compat; use ListEcosystems for new code).
func ListTrustRegistries(client cosmosclient.Client, ctx context.Context, responseMaxSize uint32) (*ectypes.QueryListEcosystemsResponse, error) {
	return ListEcosystems(client, ctx, 0, responseMaxSize)
}

// ListEcosystems lists ecosystems, optionally filtered by corporation.
func ListEcosystems(client cosmosclient.Client, ctx context.Context, corporationID uint64, responseMaxSize uint32) (*ectypes.QueryListEcosystemsResponse, error) {
	queryClient := ectypes.NewQueryClient(client.Context())
	return queryClient.ListEcosystems(ctx, &ectypes.QueryListEcosystemsRequest{
		CorporationId:   corporationID,
		ResponseMaxSize: responseMaxSize,
	})
}

// QueryCorporation fetches a corporation by its id.
func QueryCorporation(client cosmosclient.Client, ctx context.Context, corpID uint64) (*cotypes.QueryGetCorporationResponse, error) {
	qc := cotypes.NewQueryClient(client.Context())
	return qc.GetCorporation(ctx, &cotypes.QueryGetCorporationRequest{CorporationId: corpID})
}

// ListCorporations lists corporations up to responseMaxSize results.
func ListCorporations(client cosmosclient.Client, ctx context.Context, responseMaxSize uint32) (*cotypes.QueryListCorporationsResponse, error) {
	qc := cotypes.NewQueryClient(client.Context())
	return qc.ListCorporations(ctx, &cotypes.QueryListCorporationsRequest{ResponseMaxSize: responseMaxSize})
}

// QueryGFV fetches a GovernanceFrameworkVersion by its id.
func QueryGFV(client cosmosclient.Client, ctx context.Context, gfvID uint64) (*gftypes.QueryGetGovernanceFrameworkVersionResponse, error) {
	qc := gftypes.NewQueryClient(client.Context())
	return qc.GetGovernanceFrameworkVersion(ctx, &gftypes.QueryGetGovernanceFrameworkVersionRequest{Id: gfvID})
}

// ListGFVs lists GovernanceFrameworkVersions for a corporation or ecosystem.
// Exactly one of corporationID / ecosystemID must be non-zero.
func ListGFVs(client cosmosclient.Client, ctx context.Context, corporationID, ecosystemID uint64, activeOnly bool, responseMaxSize uint32) (*gftypes.QueryListGovernanceFrameworkVersionsResponse, error) {
	qc := gftypes.NewQueryClient(client.Context())
	return qc.ListGovernanceFrameworkVersions(ctx, &gftypes.QueryListGovernanceFrameworkVersionsRequest{
		CorporationId:   corporationID,
		EcosystemId:     ecosystemID,
		ActiveOnly:      activeOnly,
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
func QueryPermission(client cosmosclient.Client, ctx context.Context, permID uint64) (*permtypes.QueryGetParticipantResponse, error) {
	permQueryClient := permtypes.NewQueryClient(client.Context())
	return permQueryClient.GetParticipant(ctx, &permtypes.QueryGetParticipantRequest{Id: permID})
}

// ListPermissions lists all permissions
func ListParticipants(client cosmosclient.Client, ctx context.Context) ([]permtypes.Participant, error) {
	permQueryClient := permtypes.NewQueryClient(client.Context())
	resp, err := permQueryClient.ListParticipants(ctx, &permtypes.QueryListParticipantsRequest{
		ResponseMaxSize: 1024,
	})
	if err != nil {
		return nil, err
	}
	return resp.Participants, nil
}

// VerifyEcosystem verifies an ecosystem exists with expected properties
func VerifyEcosystem(client cosmosclient.Client, ctx context.Context, trID uint64, expectedDID string) bool {
	resp, err := QueryEcosystem(client, ctx, trID)
	if err != nil {
		fmt.Printf("❌ Ecosystem verification failed: %v\n", err)
		return false
	}

	// Verify DID matches what we expect
	if resp.Ecosystem.Did != expectedDID {
		fmt.Printf("❌ Ecosystem verification failed: Expected DID %s, got %s\n",
			expectedDID, resp.Ecosystem.Did)
		return false
	}

	fmt.Printf("✅ Verified Ecosystem ID %d exists with expected DID %s\n",
		trID, resp.Ecosystem.Did)
	return true
}

// VerifyCredentialSchema verifies a credential schema exists with expected properties
func VerifyCredentialSchema(client cosmosclient.Client, ctx context.Context, csID uint64, expectedTrID uint64) bool {
	resp, err := QueryCredentialSchema(client, ctx, csID)
	if err != nil {
		fmt.Printf("❌ Credential Schema verification failed: %v\n", err)
		return false
	}

	// Verify Ecosystem ID matches what we expect
	if resp.Schema.EcosystemId != expectedTrID {
		fmt.Printf("❌ Credential Schema verification failed: Expected Ecosystem ID %d, got %d\n",
			expectedTrID, resp.Schema.EcosystemId)
		return false
	}

	fmt.Printf("✅ Verified Credential Schema ID %d exists with expected Ecosystem ID %d\n",
		csID, resp.Schema.EcosystemId)
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
	if resp.Participant.SchemaId != expectedSchemaID {
		fmt.Printf("❌ Permission verification failed: Expected Schema ID %d, got %d\n",
			expectedSchemaID, resp.Participant.SchemaId)
		return false
	}

	permType := permtypes.ParticipantRole_name[int32(resp.Participant.Role)]
	if permType != expectedType {
		fmt.Printf("❌ Permission verification failed: Expected type %s, got %s\n",
			expectedType, permType)
		return false
	}

	fmt.Printf("✅ Verified Permission ID %d exists with expected Schema ID %d and type %s\n",
		permID, resp.Participant.SchemaId, permType)
	return true
}
