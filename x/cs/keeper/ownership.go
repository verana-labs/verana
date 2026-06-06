package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cotypes "github.com/verana-labs/verana/x/co/types"
	"github.com/verana-labs/verana/x/cs/types"
)

// checkSchemaOwnership enforces that the signing corporation is the
// controller of the Ecosystem that owns the given CredentialSchema.
// Replaces the old `tr.Corporation == msg.Corporation` ownership check
// post-MOD-EC rename: resolves the signing policy_address → co.Id and
// compares against ec.CorporationId.
func (k Keeper) checkSchemaOwnership(ctx sdk.Context, cs types.CredentialSchema, signingCorp string) error {
	ec, err := k.ecosystemKeeper.GetEcosystem(ctx, cs.EcosystemId)
	if err != nil {
		return fmt.Errorf("ecosystem %d not found: %w", cs.EcosystemId, err)
	}
	co, ok := k.coKeeper.ResolveByPolicyAddress(ctx, signingCorp)
	if !ok {
		// AUTHZ-CHECK-5: signing account is not the policy_address of any Corporation.
		return errors.Wrapf(cotypes.ErrCorporationNotRegistered,
			"signing account %s has not been registered as the policy_address of a Corporation (see MOD-CO-MSG-1)", signingCorp)
	}
	if ec.CorporationId != co.Id {
		return fmt.Errorf("corporation %d does not control the ecosystem (%d) that owns this credential schema", co.Id, ec.CorporationId)
	}
	return nil
}

// checkCreateSchemaOwnership enforces ownership at CreateCredentialSchema time
// where the CredentialSchema entry does not yet exist; resolves directly from
// msg.EcosystemId and signing corporation.
func (k Keeper) checkCreateSchemaOwnership(ctx sdk.Context, ecosystemID uint64, signingCorp string) error {
	ec, err := k.ecosystemKeeper.GetEcosystem(ctx, ecosystemID)
	if err != nil {
		return fmt.Errorf("ecosystem %d not found: %w", ecosystemID, err)
	}
	co, ok := k.coKeeper.ResolveByPolicyAddress(ctx, signingCorp)
	if !ok {
		// AUTHZ-CHECK-5: signing account is not the policy_address of any Corporation.
		return errors.Wrapf(cotypes.ErrCorporationNotRegistered,
			"signing account %s has not been registered as the policy_address of a Corporation (see MOD-CO-MSG-1)", signingCorp)
	}
	if ec.CorporationId != co.Id {
		return fmt.Errorf("corporation %d does not control ecosystem %d", co.Id, ec.CorporationId)
	}
	return nil
}
