package keeper

import (
	"fmt"
	"regexp"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/tr/types"
)

func isValidLanguageTag(lang string) bool {
	// RFC1766 primary tag must be exactly 2 letters
	if len(lang) != 2 {
		return false
	}
	// Must be lowercase letters only
	match, _ := regexp.MatchString(`^[a-z]{2}$`, lang)
	return match
}

func (ms msgServer) createTrustRegistryEntries(ctx sdk.Context, msg *types.MsgCreateTrustRegistry, now time.Time) (types.TrustRegistry, types.GovernanceFrameworkVersion, types.GovernanceFrameworkDocument, error) {
	// Generate next ID for trust registry
	nextTrId, err := ms.Keeper.GetNextID(ctx, "tr")
	if err != nil {
		return types.TrustRegistry{}, types.GovernanceFrameworkVersion{}, types.GovernanceFrameworkDocument{}, fmt.Errorf("failed to generate trust registry ID: %w", err)
	}

	// Create trust registry
	tr := types.TrustRegistry{
		Id:            nextTrId,
		Did:           msg.Did,
		Controller:    msg.Creator,
		Created:       now,
		Modified:      now,
		Deposit:       0,
		Archived:      nil,
		Aka:           msg.Aka,
		ActiveVersion: 1,
		Language:      msg.Language,
	}

	// Generate next ID for governance framework version
	nextGfvId, err := ms.Keeper.GetNextID(ctx, "gfv")
	if err != nil {
		return types.TrustRegistry{}, types.GovernanceFrameworkVersion{}, types.GovernanceFrameworkDocument{}, fmt.Errorf("failed to generate governance framework version ID: %w", err)
	}

	// Create governance framework version
	gfv := types.GovernanceFrameworkVersion{
		Id:          nextGfvId,
		TrId:        tr.Id,
		Created:     now,
		Version:     1,
		ActiveSince: now,
	}

	// Generate next ID for governance framework document
	nextGfdId, err := ms.Keeper.GetNextID(ctx, "gfd")
	if err != nil {
		return types.TrustRegistry{}, types.GovernanceFrameworkVersion{}, types.GovernanceFrameworkDocument{}, fmt.Errorf("failed to generate governance framework document ID: %w", err)
	}

	// Create governance framework document
	gfd := types.GovernanceFrameworkDocument{
		Id:        nextGfdId,
		GfvId:     gfv.Id,
		Created:   now,
		Language:  msg.Language,
		Url:       msg.DocUrl,
		DigestSri: msg.DocDigestSri,
	}

	return tr, gfv, gfd, nil
}

func (ms msgServer) persistEntries(ctx sdk.Context, tr types.TrustRegistry, gfv types.GovernanceFrameworkVersion, gfd types.GovernanceFrameworkDocument) error {
	if err := ms.TrustRegistry.Set(ctx, tr.Id, tr); err != nil {
		return fmt.Errorf("failed to persist TrustRegistry: %w", err)
	}

	// Store DID -> ID index
	if err := ms.TrustRegistryDIDIndex.Set(ctx, tr.Did, tr.Id); err != nil {
		return fmt.Errorf("failed to persist DID index: %w", err)
	}

	if err := ms.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
		return fmt.Errorf("failed to persist GovernanceFrameworkVersion: %w", err)
	}

	if err := ms.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return fmt.Errorf("failed to persist GovernanceFrameworkDocument: %w", err)
	}

	return nil
}
