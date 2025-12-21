package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/verana-labs/verana/testharness/lib"

	"cosmossdk.io/math"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	cschema "github.com/verana-labs/verana/x/cs/types"
	permtypes "github.com/verana-labs/verana/x/perm/types"
)

// RunCreatePermissionJourney implements Journey 18: Create Permission
func RunCreatePermissionJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 18: Create Permission")

	// Setup: Fund account, create trust registry and schema
	trustRegistryControllerAccount := lib.GetAccount(client, lib.TRUST_REGISTRY_CONTROLLER_NAME)
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.TRUST_REGISTRY_CONTROLLER_ADDRESS, math.NewInt(10000000)) // 10 VNA
	did := lib.GenerateUniqueDID(client, ctx)
	trustRegistryID := lib.CreateNewTrustRegistry(client, ctx, trustRegistryControllerAccount, did)
	schemaData := lib.GenerateSimpleSchema(trustRegistryID)
	schemaID := lib.CreateSimpleCredentialSchema(
		client, ctx, trustRegistryControllerAccount, trustRegistryID, schemaData,
		cschema.CredentialSchemaPermManagementMode_OPEN,
		cschema.CredentialSchemaPermManagementMode_OPEN,
	)

	effectiveFrom := time.Now().Add(time.Second * 10)
	effectiveUntil := effectiveFrom.Add(time.Hour * 24 * 360)

	// Step: Create the root (ecosystem) permission for the schema (required by spec)
	_ = lib.CreateRootPermissionWithDates(
		client, ctx, trustRegistryControllerAccount, schemaID, did, effectiveFrom, effectiveUntil, 0, 0, 0,
	)

	// Step: Create a permission directly
	permMsg := permtypes.MsgCreatePermission{
		SchemaId:         func() uint64 { id, _ := strconv.ParseUint(schemaID, 10, 64); return id }(),
		Type:             permtypes.PermissionType_ISSUER,
		Did:              did,
		Country:          "US",
		EffectiveFrom:    &effectiveFrom,
		EffectiveUntil:   &effectiveUntil,
		VerificationFees: 1000, // Example fee
	}
	fmt.Println("Creating permission directly...")
	lib.CreatePermission(client, ctx, trustRegistryControllerAccount, permMsg)
	fmt.Println("✅ Step: Permission created directly")

	// Save result
	result := lib.JourneyResult{
		TrustRegistryID: trustRegistryID,
		SchemaID:        schemaID,
		DID:             did,
	}
	lib.SaveJourneyResult("journey18", result)
	return nil
}
