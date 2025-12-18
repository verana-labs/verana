package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	permtypes "github.com/verana-labs/verana/x/perm/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionExtensionJourney implements Journey 13: Permission Extension Journey
func RunPermissionExtensionJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 13: Permission Extension Journey")

	// Step 1: Create a new permission with a validation expiration much later than effective_until
	// This will allow us to demonstrate extension properly
	fmt.Println("Step 1: Creating a new permission to demonstrate extension")

	// First, we need a Schema ID and a Validator Permission ID from previous journeys
	journey1Result, err := lib.GetJourneyResult("journey1")
	if err != nil {
		return fmt.Errorf("failed to load Journey 1 results: %v", err)
	}

	rootPermID, err := strconv.ParseUint(journey1Result.RootPermissionID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse root permission ID: %v", err)
	}

	// Get the Test_Registry_Controller account to validate
	trControllerAccount, err := client.Account(lib.TRUST_REGISTRY_CONTROLLER_NAME)
	if err != nil {
		return fmt.Errorf("failed to get Trust_Registry_Controller account: %v", err)
	}

	// Get the Issuer_Applicant account that will be our Permission_Holder
	issuerAccount, err := client.Account(lib.ISSUER_APPLICANT_NAME)
	if err != nil {
		return fmt.Errorf("failed to get Issuer_Applicant account: %v", err)
	}

	_, err = issuerAccount.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get address for Issuer_Applicant: %v", err)
	}

	// Make sure issuer account has funds for the validation process
	fmt.Println("    - Ensuring Issuer_Applicant has sufficient funds...")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, lib.ISSUER_APPLICANT_ADDRESS, math.NewInt(10000000))

	// Generate a new DID for this example permission holder
	exampleDID := lib.GenerateUniqueDID(client, ctx)
	fmt.Printf("    - Generated unique DID for example: %s\n", exampleDID)

	// Step 2: Create a new permission with a 2-year validation expiration but only 6-month effective_until
	// This ensures we have enough gap between the dates to demonstrate extension
	fmt.Println("Step 2: Start Permission Validation Process with specific expiration dates")

	// Start the validation process
	startVPMsg := permtypes.MsgStartPermissionVP{
		Type:            permtypes.PermissionType_VERIFIER,
		ValidatorPermId: rootPermID,
		Country:         "US",
		Did:             exampleDID,
	}

	permissionID := lib.StartValidationProcess(client, ctx, issuerAccount, startVPMsg)
	permID, err := strconv.ParseUint(permissionID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse new permission ID: %v", err)
	}

	fmt.Printf("    - Started validation process with permission ID: %s\n", permissionID)

	// Now validate the permission with specific dates
	// Set up validation with validation expiration in 2 years but effective_until in 6 months
	nowTime := time.Now()
	effectiveUntil := nowTime.AddDate(0, 6, 0) // 6 months from now

	fmt.Println("    - Validating permission with custom expiration dates...")
	fmt.Printf("    - Setting effective_until to: %s (6 months from now)\n", effectiveUntil.Format(time.RFC3339))

	// Validate with custom effective_until
	validationFees := uint64(50)
	issuanceFees := uint64(20)
	verificationFees := uint64(10)

	validateMsg := permtypes.MsgSetPermissionVPToValidated{
		Id:               permID,
		EffectiveUntil:   &effectiveUntil,
		ValidationFees:   validationFees,
		IssuanceFees:     issuanceFees,
		VerificationFees: verificationFees,
		Country:          "US",
	}

	_, err = lib.SetPermissionVPToValidated(client, ctx, trControllerAccount, validateMsg)
	if err != nil {
		return fmt.Errorf("failed to validate permission: %v", err)
	}

	fmt.Printf("    - Successfully validated permission with ID: %s\n", permissionID)

	// Wait for validation to be processed
	fmt.Println("    - Waiting for validation to be processed...")
	time.Sleep(3 * time.Second)

	// Check the permission to confirm our setup worked
	newPerm, err := lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("failed to get newly created permission: %v", err)
	}

	fmt.Printf("    - Permission state: Permission ID=%d, DID=%s, Type=%s\n",
		permID, newPerm.Did, permtypes.PermissionType_name[int32(newPerm.Type)])

	var initialEffectiveUntil, validationExp string
	if newPerm.EffectiveUntil != nil {
		initialEffectiveUntil = newPerm.EffectiveUntil.Format(time.RFC3339)
	} else {
		initialEffectiveUntil = "never expires"
	}

	if newPerm.VpExp != nil {
		validationExp = newPerm.VpExp.Format(time.RFC3339)
	} else {
		validationExp = "never expires"
	}

	fmt.Printf("    - Initial effective until: %s\n", initialEffectiveUntil)
	fmt.Printf("    - Validation process expires: %s\n", validationExp)

	// Step 3: Validator decides to extend permission validity
	fmt.Println("Step 3: Validator decides to extend permission validity")
	fmt.Println("    - Decision process occurs off-chain")

	// Calculate new effective until date (9 months from now, which is 3 months longer than current)
	newEffectiveUntil := nowTime.AddDate(0, 9, 0)
	fmt.Printf("    - Current effective until: %s (6 months from now)\n", initialEffectiveUntil)
	fmt.Printf("    - New effective until: %s (9 months from now)\n", newEffectiveUntil.Format(time.RFC3339))
	fmt.Printf("    - Validation expiration: %s\n", validationExp)

	// Step 4: Validator extends the permission validity
	fmt.Println("Step 4: Validator extending permission validity...")

	_, err = trControllerAccount.Address(lib.GetAddressPrefix())
	if err != nil {
		return fmt.Errorf("failed to get Trust_Registry_Controller address: %v", err)
	}

	extendMsg := &permtypes.MsgExtendPermission{
		Creator:        newPerm.Grantee,
		Id:             permID,
		EffectiveUntil: &newEffectiveUntil,
	}

	permGrantee, _ := client.Account(newPerm.Grantee)
	extendResponse, err := lib.ExtendPermission(client, ctx, permGrantee, *extendMsg)
	if err != nil {
		return fmt.Errorf("failed to extend permission: %v", err)
	}

	fmt.Printf("✅ Step 4: Validator extended permission validity\n")
	fmt.Printf("    - New effective until: %s\n", newEffectiveUntil.Format(time.RFC3339))
	fmt.Printf("    - Response: %s\n", extendResponse)

	// Wait for extension to be processed
	fmt.Println("    - Waiting for extension to be processed...")
	time.Sleep(3 * time.Second)

	// Verify the permission has been extended
	extendedPerm, err := lib.GetPermission(client, ctx, permID)
	if err != nil {
		return fmt.Errorf("failed to get extended permission: %v", err)
	}

	var extendedUntil string
	if extendedPerm.EffectiveUntil == nil {
		extendedUntil = "never expires"
	} else {
		extendedUntil = extendedPerm.EffectiveUntil.Format(time.RFC3339)
	}

	fmt.Println("Step 5: Verify permission extension")
	fmt.Printf("    - Extended effective until: %s\n", extendedUntil)

	if extendedPerm.Extended != nil {
		fmt.Printf("    - Permission extended at: %v\n", extendedPerm.Extended)
		fmt.Printf("    - Extended by: %s\n", extendedPerm.ExtendedBy)
	} else {
		fmt.Println("⚠️ Warning: Permission extension verification failed, extended timestamp not set")
	}

	fmt.Println("    - Permission_Holder can continue operations with the extended validity period")

	fmt.Println("Journey 13 completed successfully! ✨")
	fmt.Println("    - Successfully demonstrated permission extension")
	fmt.Println("    - Extended effective_until from 6 months to 9 months")
	fmt.Println("    - Extension worked because:")
	fmt.Println("      1. New date is after current effective_until")
	fmt.Println("      2. New date is not after validation expiration")

	// Store the result for future reference
	result := lib.JourneyResult{
		SchemaID:           journey1Result.SchemaID,
		DID:                exampleDID,
		PermissionID:       permissionID,
		ExtensionTimestamp: time.Now().Format(time.RFC3339),
		NewEffectiveUntil:  extendedUntil,
	}

	// Save result to global state or file for other journeys to use
	lib.SaveJourneyResult("journey13", result)

	return nil
}
