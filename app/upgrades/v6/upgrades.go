package v6

import (
	"context"
	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/verana-labs/verana-blockchain/app/upgrades/types"
	credentialschematypes "github.com/verana-labs/verana-blockchain/x/credentialschema/types"
	diddirectorytypes "github.com/verana-labs/verana-blockchain/x/diddirectory/types"
	permissiontypes "github.com/verana-labs/verana-blockchain/x/permission/types"
	trustdeposittypes "github.com/verana-labs/verana-blockchain/x/trustdeposit/types"
	trustregistrytypes "github.com/verana-labs/verana-blockchain/x/trustregistry/types"
	"strconv"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ types.BaseAppParamManager,
	keepers types.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(context context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(context)

		// Get extracted data
		trustRegistryData, trustDepositData, credentialSchemaData, permissionData, didDirectoryData := GetExtractedData()

		// Restore Trust Registry data
		if err := restoreTrustRegistryData(ctx, keepers, trustRegistryData); err != nil {
			return nil, fmt.Errorf("failed to restore trust registry data: %w", err)
		}

		// Restore Trust Deposit data
		if err := restoreTrustDepositData(ctx, keepers, trustDepositData); err != nil {
			return nil, fmt.Errorf("failed to restore trust deposit data: %w", err)
		}

		// Restore Credential Schema data
		if err := restoreCredentialSchemaData(ctx, keepers, credentialSchemaData); err != nil {
			return nil, fmt.Errorf("failed to restore credential schema data: %w", err)
		}

		// Restore Permission data
		if err := restorePermissionData(ctx, keepers, permissionData); err != nil {
			return nil, fmt.Errorf("failed to restore permission data: %w", err)
		}

		// Restore DID Directory data
		if err := restoreDIDDirectoryData(ctx, keepers, didDirectoryData); err != nil {
			return nil, fmt.Errorf("failed to restore did directory data: %w", err)
		}

		// Run standard migrations
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

func restoreTrustRegistryData(ctx sdk.Context, keepers types.AppKeepers, data TrustRegistryData) error {
	// Restore trust registries
	for _, tr := range data.TrustRegistries {
		id, _ := strconv.ParseUint(tr.ID, 10, 64)
		deposit, _ := math.NewIntFromString(tr.Deposit)
		trUpdated := trustregistrytypes.TrustRegistry{
			Id:            id,
			Did:           tr.DID,
			Controller:    tr.Controller,
			Created:       tr.Created,
			Modified:      tr.Modified,
			Archived:      nil,
			Deposit:       deposit.Int64(),
			Aka:           tr.AKA,
			ActiveVersion: tr.ActiveVersion,
			Language:      tr.Language,
		}
		err := keepers.GetTrustRegistryKeeper().TrustRegistry.Set(ctx, id, trUpdated)
		if err != nil {
			return err
		}

		fmt.Printf("Restoring trust registry ID: %s, DID: %s\n", tr.ID, tr.DID)
	}

	// Restore governance framework versions
	for _, gfv := range data.GovernanceFrameworkVersions {
		id, _ := strconv.ParseUint(gfv.ID, 10, 64)
		trID, _ := strconv.ParseUint(gfv.TRID, 10, 64)
		gfvUpdated := trustregistrytypes.GovernanceFrameworkVersion{
			Id:          id,
			TrId:        trID,
			Created:     gfv.Created,
			Version:     gfv.Version,
			ActiveSince: gfv.ActiveSince,
		}

		err := keepers.GetTrustRegistryKeeper().GFVersion.Set(ctx, id, gfvUpdated)
		if err != nil {
			return err
		}

		fmt.Printf("Restoring governance framework version ID: %s, TR ID: %s\n", gfv.ID, gfv.TRID)
	}

	// Restore governance framework documents
	for _, gfd := range data.GovernanceFrameworkDocuments {
		id, _ := strconv.ParseUint(gfd.ID, 10, 64)
		gfvID, _ := strconv.ParseUint(gfd.GFVID, 10, 64)
		gfdUpdated := trustregistrytypes.GovernanceFrameworkDocument{
			Id:        id,
			GfvId:     gfvID,
			Created:   gfd.Created,
			Language:  gfd.Language,
			Url:       gfd.URL,
			DigestSri: gfd.DigestSRI,
		}

		err := keepers.GetTrustRegistryKeeper().GFDocument.Set(ctx, id, gfdUpdated)
		if err != nil {
			return err
		}

		fmt.Printf("Restoring governance framework document ID: %s, GFV ID: %s\n", gfd.ID, gfd.GFVID)
	}

	// Restore counters
	for _, counter := range data.Counters {
		value, _ := strconv.ParseUint(counter.Value, 10, 64)

		err := keepers.GetTrustRegistryKeeper().Counter.Set(ctx, counter.EntityType, value)
		if err != nil {
			return err
		}
		fmt.Printf("Restoring counter %s: %s\n", counter.EntityType, counter.Value)
	}

	return nil
}

func restoreTrustDepositData(ctx sdk.Context, keepers types.AppKeepers, data TrustDepositData) error {
	// Restore trust deposits
	for _, td := range data.TrustDeposits {
		share, _ := math.NewIntFromString(td.Share)
		amount, _ := math.NewIntFromString(td.Amount)
		claimable, _ := math.NewIntFromString(td.Claimable)

		tdUpdated := trustdeposittypes.TrustDeposit{
			Account:        td.Account,
			Share:          share.Uint64(),
			Amount:         amount.Uint64(),
			Claimable:      claimable.Uint64(),
			SlashedDeposit: 0,
			RepaidDeposit:  0,
			LastSlashed:    nil,
			LastRepaid:     nil,
			SlashCount:     0,
			LastRepaidBy:   "",
		}
		// Call your keeper method to set trust deposit
		err := keepers.GetTrustDepositKeeper().TrustDeposit.Set(ctx, td.Account, tdUpdated)
		if err != nil {
			return err
		}

		fmt.Printf("Restoring trust deposit for account: %s, amount: %s\n", td.Account, td.Amount)
	}

	return nil
}

func restoreCredentialSchemaData(ctx sdk.Context, keepers types.AppKeepers, data CredentialSchemaData) error {
	// Restore credential schemas
	for _, cs := range data.CredentialSchemas {
		id, _ := strconv.ParseUint(cs.ID, 10, 64)
		trID, _ := strconv.ParseUint(cs.TRID, 10, 64)
		deposit, _ := math.NewIntFromString(cs.Deposit)
		IssuerPermManagementMode, _ := math.NewIntFromString(cs.IssuerPermManagementMode)
		VerifierPermManagementMode, _ := math.NewIntFromString(cs.VerifierPermManagementMode)
		csUpdated := credentialschematypes.CredentialSchema{
			Id:                                      id,
			TrId:                                    trID,
			Created:                                 cs.Created,
			Modified:                                cs.Modified,
			Archived:                                nil,
			Deposit:                                 deposit.Uint64(),
			JsonSchema:                              cs.JSONSchema,
			IssuerGrantorValidationValidityPeriod:   cs.IssuerGrantorValidationValidityPeriod,
			VerifierGrantorValidationValidityPeriod: cs.VerifierGrantorValidationValidityPeriod,
			IssuerValidationValidityPeriod:          cs.IssuerValidationValidityPeriod,
			VerifierValidationValidityPeriod:        cs.VerifierValidationValidityPeriod,
			HolderValidationValidityPeriod:          cs.HolderValidationValidityPeriod,
			IssuerPermManagementMode:                credentialschematypes.CredentialSchemaPermManagementMode(IssuerPermManagementMode.Int64()),
			VerifierPermManagementMode:              credentialschematypes.CredentialSchemaPermManagementMode(VerifierPermManagementMode.Int64()),
		}

		// Call your keeper method to set credential schema
		err := keepers.GetCredentialSchemaKeeper().CredentialSchema.Set(ctx, id, csUpdated)
		if err != nil {
			return err
		}

		fmt.Printf("Restoring credential schema ID: %s, TR ID: %s\n", cs.ID, cs.TRID)
	}

	// Set schema counter
	counter, _ := strconv.ParseUint(data.SchemaCounter, 10, 64)
	err := keepers.GetCredentialSchemaKeeper().Counter.Set(ctx, "cs", counter)
	if err != nil {
		return err
	}

	fmt.Printf("Restoring schema counter: %s\n", data.SchemaCounter)

	return nil
}

func restorePermissionData(ctx sdk.Context, keepers types.AppKeepers, data PermissionData) error {
	// Restore permissions
	for _, perm := range data.Permissions {
		id, _ := strconv.ParseUint(perm.ID, 10, 64)
		schemaID, _ := strconv.ParseUint(perm.SchemaID, 10, 64)
		permType, _ := strconv.ParseUint(perm.Type, 10, 64)
		validationFees, _ := strconv.ParseUint(perm.ValidationFees, 10, 64)
		issuanceFees, _ := strconv.ParseUint(perm.IssuanceFees, 10, 64)
		verificationFees, _ := strconv.ParseUint(perm.VerificationFees, 10, 64)
		deposit, _ := strconv.ParseUint(perm.Deposit, 10, 64)
		validatorPermID, _ := strconv.ParseUint(perm.ValidatorPermID, 10, 64)
		vpState, _ := strconv.ParseUint(perm.VPState, 10, 64)

		permUpdated := permissiontypes.Permission{
			Id:                 id,
			SchemaId:           schemaID,
			Type:               permissiontypes.PermissionType(permType),
			Did:                perm.DID,
			Grantee:            perm.Grantee,
			Created:            &perm.Created,
			CreatedBy:          perm.CreatedBy,
			Extended:           nil,
			ExtendedBy:         "",
			Slashed:            nil,
			SlashedBy:          "",
			Repaid:             nil,
			RepaidBy:           "",
			EffectiveFrom:      &perm.EffectiveFrom,
			EffectiveUntil:     &perm.EffectiveUntil,
			Modified:           &perm.Modified,
			ValidationFees:     validationFees,
			IssuanceFees:       issuanceFees,
			VerificationFees:   verificationFees,
			Deposit:            deposit,
			SlashedDeposit:     0,
			RepaidDeposit:      0,
			Revoked:            nil,
			RevokedBy:          "",
			Terminated:         nil,
			TerminatedBy:       "",
			Country:            perm.Country,
			ValidatorPermId:    validatorPermID,
			VpState:            permissiontypes.ValidationState(vpState),
			VpExp:              nil,
			VpLastStateChange:  nil,
			VpValidatorDeposit: 0,
			VpCurrentFees:      0,
			VpCurrentDeposit:   0,
			VpSummaryDigestSri: "",
			VpTermRequested:    nil,
		}
		err := keepers.GetPermissionKeeper().Permission.Set(ctx, id, permUpdated)
		if err != nil {
			return err
		}

		fmt.Printf("Restoring permission ID: %s, schema ID: %s, grantee: %s\n", perm.ID, perm.SchemaID, perm.Grantee)
	}

	// Set next permission ID
	nextID, _ := strconv.ParseUint(data.NextPermissionID, 10, 64)

	err := keepers.GetPermissionKeeper().PermissionCounter.Set(ctx, nextID)
	if err != nil {
		return err
	}

	fmt.Printf("Restoring next permission ID: %s\n", data.NextPermissionID)

	return nil
}

func restoreDIDDirectoryData(ctx sdk.Context, keepers types.AppKeepers, data DIDDirectoryData) error {
	// Restore DID directories
	for _, did := range data.DIDDirectories {
		deposit, _ := math.NewIntFromString(did.Deposit)

		didUpdated := diddirectorytypes.DIDDirectory{
			Did:        did.DID,
			Controller: did.Controller,
			Created:    did.Created,
			Modified:   did.Modified,
			Exp:        did.Exp,
			Deposit:    deposit.Int64(),
		}

		err := keepers.GetDidDirectoryKeeper().DIDDirectory.Set(ctx, did.DID, didUpdated)
		if err != nil {
			return err
		}

		fmt.Printf("Restoring DID directory: %s, controller: %s\n", did.DID, did.Controller)
	}

	return nil
}
