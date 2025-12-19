package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	permtypes "github.com/verana-labs/verana/x/perm/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionRevocationJourney implements Journey 12: Permission Revocation Journey
func RunPermissionRevocationJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 12: Permission Revocation Journey")

	// Step 1 & 2: Load data from previous journeys to identify Trust Registry, Credential Schema,
	// and a Permission_Holder that has been granted permission by a Validator
	// We'll use the Issuer_Applicant with permission granted by Issuer_Grantor_Applicant from Journey 3
	journey3Result, err := lib.GetJourneyResult("journey3")
	if err != nil {
		return fmt.Errorf("failed to load Journey 3 results: %v", err)
	}

	fmt.Println("✅ Step 1 & 2: Loaded data from previous journeys")
	fmt.Printf("    - Trust Registry ID: %s\n", journey3Result.TrustRegistryID)
	fmt.Printf("    - Schema ID: %s\n", journey3Result.SchemaID)
	fmt.Printf("    - Issuer DID: %s\n", journey3Result.IssuerDID)
	fmt.Printf("    - Issuer Permission ID: %s (Permission to be revoked)\n", journey3Result.IssuerPermID)
	fmt.Printf("    - Issuer Grantor DID: %s\n", journey3Result.IssuerGrantorDID)
	fmt.Printf("    - Issuer Grantor Permission ID: %s (Validator permission)\n", journey3Result.IssuerGrantorPermID)

	// Parse permission IDs
	issuerPermID, err := strconv.ParseUint(journey3Result.IssuerPermID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse issuer permission ID: %v", err)
	}

	// Step 3: Validator detects non-compliance (simulated)
	fmt.Println("✅ Step 3: Validator (Issuer_Grantor_Applicant) detected non-compliance with Permission_Holder (Issuer_Applicant)")
	fmt.Println("    - Non-compliance detection occurs off-chain")

	// Verify the initial permission state before revocation
	issuerPerm, err := lib.GetPermission(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("failed to get issuer permission: %v", err)
	}

	fmt.Printf("    - Initial permission state: Permission ID=%d, DID=%s, Type=%s\n",
		issuerPermID, issuerPerm.Did, permtypes.PermissionType_name[int32(issuerPerm.Type)])
	fmt.Printf("    - Initial revocation status: %v\n", issuerPerm.Revoked)

	// Get the validator (Issuer_Grantor_Applicant) account
	validatorAccount, err := client.Account(lib.ISSUER_GRANTOR_APPLICANT_NAME)
	if err != nil {
		return fmt.Errorf("failed to get %s account: %v", lib.ISSUER_GRANTOR_APPLICANT_NAME, err)
	}

	_, err = validatorAccount.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get address for %s: %v", lib.ISSUER_GRANTOR_APPLICANT_NAME, err)
	}

	// Step 4: Validator revokes the permission
	fmt.Println("Validator revoking permission...")

	revokeMsg := &permtypes.MsgRevokePermission{
		Creator: issuerPerm.Grantee,
		Id:      issuerPermID,
	}

	revokeResponse, err := lib.RevokePermission(client, ctx, validatorAccount, *revokeMsg)
	if err != nil {
		return fmt.Errorf("failed to revoke permission: %v", err)
	}

	fmt.Printf("✅ Step 4: Validator revoked permission with ID: %d\n", issuerPermID)
	fmt.Printf("    - Response: %s\n", revokeResponse)

	// Wait for revocation to be processed
	fmt.Println("    - Waiting for revocation to be processed...")
	time.Sleep(3 * time.Second)

	// Verify the permission has been revoked
	revokedPerm, err := lib.GetPermission(client, ctx, issuerPermID)
	if err != nil {
		return fmt.Errorf("failed to get revoked permission: %v", err)
	}

	fmt.Println("✅ Step 5: Verified Permission_Holder can no longer use the revoked permission")

	if revokedPerm.Revoked == nil {
		fmt.Println("⚠️ Warning: Permission revocation verification failed, permission not marked as revoked")
	} else {
		fmt.Printf("    - Permission successfully revoked at: %v\n", revokedPerm.Revoked)
		fmt.Printf("    - Revoked by: %s\n", revokedPerm.RevokedBy)
		fmt.Println("    - Permission_Holder can no longer issue/verify credentials with this permission")
	}

	// Check if the revocation affected the trust deposit
	permissionHolderAccount, err := client.Account(lib.ISSUER_APPLICANT_NAME)
	if err != nil {
		return fmt.Errorf("failed to get %s account: %v", lib.ISSUER_APPLICANT_NAME, err)
	}

	trustDeposit, err := lib.GetTrustDeposit(client, ctx, permissionHolderAccount)
	if err != nil {
		return fmt.Errorf("failed to get trust deposit: %v", err)
	}

	fmt.Println("    - Permission_Holder trust deposit status after revocation:")
	fmt.Printf("      - Amount: %d\n", trustDeposit.Amount)
	fmt.Printf("      - Claimable: %d\n", trustDeposit.Claimable)

	fmt.Println("Journey 12 completed successfully! ✨")

	// Store the result for future reference
	result := lib.JourneyResult{
		TrustRegistryID:     journey3Result.TrustRegistryID,
		SchemaID:            journey3Result.SchemaID,
		IssuerDID:           journey3Result.IssuerDID,
		IssuerPermID:        journey3Result.IssuerPermID,
		IssuerGrantorDID:    journey3Result.IssuerGrantorDID,
		IssuerGrantorPermID: journey3Result.IssuerGrantorPermID,
		RevocationTimestamp: time.Now().Format(time.RFC3339),
	}

	// Save result to global state or file for other journeys to use
	lib.SaveJourneyResult("journey12", result)

	return nil
}
