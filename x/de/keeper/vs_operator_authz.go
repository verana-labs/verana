package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/de/types"
)

// GrantVSOperatorAuthorization implements [MOD-DE-MSG-5].
// It creates or updates a VSOperatorAuthorization for the given permission,
// adding the perm_id to the list of authorized permissions and managing the
// associated fee grant.
//
// This is an internal keeper method called by the perm module:
// SelfCreatePermission, AdjustPermission, SetPermissionVPToValidated.
func (k Keeper) GrantVSOperatorAuthorization(ctx context.Context, permID uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// [MOD-DE-MSG-5-2] Basic checks

	if k.permKeeper == nil {
		return types.ErrPermKeeperRequired
	}

	// Permission MUST exist
	authority, vsOperator, withFeegrant, effectiveUntil, err := k.permKeeper.GetPermissionForVSOA(ctx, permID)
	if err != nil {
		return fmt.Errorf("%w: %d", types.ErrPermissionNotFound, permID)
	}

	// perm.authority and perm.vs_operator MUST NOT be null
	if authority == "" || vsOperator == "" {
		return types.ErrPermissionFieldsNull
	}

	// Check mutual exclusivity: OperatorAuthorization must NOT exist for
	// (authority, vs_operator)
	oaKey := collections.Join(authority, vsOperator)
	hasOA, err := k.OperatorAuthorizations.Has(sdkCtx, oaKey)
	if err != nil {
		return fmt.Errorf("failed to check OperatorAuthorization: %w", err)
	}
	if hasOA {
		return types.ErrOperatorAuthzExistsMutex
	}

	// [MOD-DE-MSG-5-4] Execution

	// Load or create VSOperatorAuthorization
	vsKey := collections.Join(authority, vsOperator)
	vsoa, err := k.VSOperatorAuthorizations.Get(sdkCtx, vsKey)
	if err != nil {
		// Doesn't exist yet — create new
		vsoa = types.VSOperatorAuthorization{
			Authority:   authority,
			VsOperator:  vsOperator,
			Permissions: []uint64{},
		}
	}

	// Add perm_id to permissions list (avoid duplicates)
	found := false
	for _, pid := range vsoa.Permissions {
		if pid == permID {
			found = true
			break
		}
	}
	if !found {
		vsoa.Permissions = append(vsoa.Permissions, permID)
	}

	if err := k.VSOperatorAuthorizations.Set(sdkCtx, vsKey, vsoa); err != nil {
		return fmt.Errorf("failed to set VSOperatorAuthorization: %w", err)
	}

	// Handle feegrant
	if withFeegrant {
		maxExpire := effectiveUntil

		if maxExpire == nil {
			// No expiration — grant with no expiration and exit
			if err := k.GrantFeeAllowance(ctx, authority, vsOperator,
				[]string{"/verana.perm.v1.MsgCreateOrUpdatePermissionSession"},
				nil, nil, nil); err != nil {
				return fmt.Errorf("failed to grant fee allowance: %w", err)
			}
		} else {
			// Iterate all permissions to find the farthest effective_until
			for _, currentPermID := range vsoa.Permissions {
				_, _, curWithFeegrant, curEffectiveUntil, err := k.permKeeper.GetPermissionForVSOA(ctx, currentPermID)
				if err != nil {
					continue // permission may have been deleted
				}
				if curWithFeegrant {
					if curEffectiveUntil == nil {
						// Found a permission with no expiration — grant with no expiration and exit
						if err := k.GrantFeeAllowance(ctx, authority, vsOperator,
							[]string{"/verana.perm.v1.MsgCreateOrUpdatePermissionSession"},
							nil, nil, nil); err != nil {
							return fmt.Errorf("failed to grant fee allowance: %w", err)
						}
						goto emitEvent
					}
					if curEffectiveUntil.After(*maxExpire) {
						maxExpire = curEffectiveUntil
					}
				}
			}
			// Grant with the farthest expiration if it's in the future
			if maxExpire.After(now) {
				if err := k.GrantFeeAllowance(ctx, authority, vsOperator,
					[]string{"/verana.perm.v1.MsgCreateOrUpdatePermissionSession"},
					maxExpire, nil, nil); err != nil {
					return fmt.Errorf("failed to grant fee allowance: %w", err)
				}
			}
		}
	}

emitEvent:
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeGrantVSOperatorAuthorization,
			sdk.NewAttribute(types.AttributeKeyAuthority, authority),
			sdk.NewAttribute(types.AttributeKeyVsOperator, vsOperator),
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(permID, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	)

	return nil
}

// CheckVSOperatorAuthorization checks if a VS operator is authorized to act on behalf of the authority.
// [AUTHZ-CHECK-3] A VSOperatorAuthorization entry must exist where authority and vs_operator match.
func (k Keeper) CheckVSOperatorAuthorization(
	ctx context.Context,
	authority string,
	vsOperator string,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	vsKey := collections.Join(authority, vsOperator)
	has, err := k.VSOperatorAuthorizations.Has(sdkCtx, vsKey)
	if err != nil {
		return fmt.Errorf("failed to check VS operator authorization: %w", err)
	}
	if !has {
		return fmt.Errorf("VS operator %s is not authorized for authority %s", vsOperator, authority)
	}

	return nil
}

// RevokeVSOperatorAuthorization implements [MOD-DE-MSG-6].
// It removes a permission from the VSOperatorAuthorization and recalculates
// the fee grant expiration.
//
// This is an internal keeper method called by the perm module:
// RevokePermission, SlashPermissionTrustDeposit.
func (k Keeper) RevokeVSOperatorAuthorization(ctx context.Context, permID uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// [MOD-DE-MSG-6-2] Basic checks

	if k.permKeeper == nil {
		return types.ErrPermKeeperRequired
	}

	// Permission MUST exist
	authority, vsOperator, permWithFeegrant, _, err := k.permKeeper.GetPermissionForVSOA(ctx, permID)
	if err != nil {
		return fmt.Errorf("%w: %d", types.ErrPermissionNotFound, permID)
	}

	// perm.authority and perm.vs_operator MUST NOT be null
	if authority == "" || vsOperator == "" {
		return types.ErrPermissionFieldsNull
	}

	// [MOD-DE-MSG-6-4] Execution

	vsKey := collections.Join(authority, vsOperator)
	vsoa, err := k.VSOperatorAuthorizations.Get(sdkCtx, vsKey)
	if err != nil {
		// vs_operator_authz is null — nothing to do
		return nil
	}

	// Remove perm_id from permissions list
	newPerms := make([]uint64, 0, len(vsoa.Permissions))
	for _, pid := range vsoa.Permissions {
		if pid != permID {
			newPerms = append(newPerms, pid)
		}
	}
	vsoa.Permissions = newPerms

	// If the removed permission had a feegrant, recalculate
	if permWithFeegrant {
		if len(vsoa.Permissions) == 0 {
			// No more permissions — revoke fee allowance
			if err := k.RevokeFeeAllowance(ctx, authority, vsOperator); err != nil {
				return fmt.Errorf("failed to revoke fee allowance: %w", err)
			}
		} else {
			// Recalculate max_expire from remaining permissions
			var maxExpire *time.Time
			for _, currentPermID := range vsoa.Permissions {
				_, _, curWithFeegrant, curEffectiveUntil, err := k.permKeeper.GetPermissionForVSOA(ctx, currentPermID)
				if err != nil {
					continue
				}
				if curWithFeegrant {
					if curEffectiveUntil == nil {
						// Found a permission with no expiration — keep feegrant with no expiration
						maxExpire = nil
						break
					}
					if maxExpire == nil {
						maxExpire = curEffectiveUntil
					} else if curEffectiveUntil.After(*maxExpire) {
						maxExpire = curEffectiveUntil
					}
				}
			}

			if maxExpire != nil && maxExpire.After(now) {
				if err := k.GrantFeeAllowance(ctx, authority, vsOperator,
					[]string{"/verana.perm.v1.MsgCreateOrUpdatePermissionSession"},
					maxExpire, nil, nil); err != nil {
					return fmt.Errorf("failed to grant fee allowance: %w", err)
				}
			}
		}
	}

	// Update or remove the VSOA entry
	if len(vsoa.Permissions) == 0 {
		if err := k.VSOperatorAuthorizations.Remove(sdkCtx, vsKey); err != nil {
			return fmt.Errorf("failed to remove VSOperatorAuthorization: %w", err)
		}
	} else {
		if err := k.VSOperatorAuthorizations.Set(sdkCtx, vsKey, vsoa); err != nil {
			return fmt.Errorf("failed to update VSOperatorAuthorization: %w", err)
		}
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRevokeVSOperatorAuthorization,
			sdk.NewAttribute(types.AttributeKeyAuthority, authority),
			sdk.NewAttribute(types.AttributeKeyVsOperator, vsOperator),
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(permID, 10)),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	)

	return nil
}
