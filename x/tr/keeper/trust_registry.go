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

// [MOD-TR-MSG-1-3] Spec draft 13: MsgCreateTrustRegistry seeds the trust
// registry, an active v1 governance framework version, AND an initial
// governance framework document (in the registry's default language) from
// doc_url / doc_digest_sri. The initial document is what makes v1 a valid
// ACTIVE version — spec [MOD-TR-MSG-3-2-1] requires every active version to
// have at least one document in the default language.
func (ms msgServer) createTrustRegistryEntries(ctx sdk.Context, msg *types.MsgCreateTrustRegistry, now time.Time) (types.TrustRegistry, types.GovernanceFrameworkVersion, types.GovernanceFrameworkDocument, error) {
	nextTrId, err := ms.Keeper.GetNextID(ctx, "tr")
	if err != nil {
		return types.TrustRegistry{}, types.GovernanceFrameworkVersion{}, types.GovernanceFrameworkDocument{}, fmt.Errorf("failed to generate trust registry ID: %w", err)
	}

	tr := types.TrustRegistry{
		Id:            nextTrId,
		Did:           msg.Did,
		Corporation:   msg.Corporation, // Corporation is the controlling group account of the trust registry
		Created:       now,
		Modified:      now,
		Archived:      nil,
		Aka:           msg.Aka,
		ActiveVersion: 1,
		Language:      msg.Language,
	}

	nextGfvId, err := ms.Keeper.GetNextID(ctx, "gfv")
	if err != nil {
		return types.TrustRegistry{}, types.GovernanceFrameworkVersion{}, types.GovernanceFrameworkDocument{}, fmt.Errorf("failed to generate governance framework version ID: %w", err)
	}

	gfv := types.GovernanceFrameworkVersion{
		Id:          nextGfvId,
		TrId:        tr.Id,
		Created:     now,
		Version:     1,
		ActiveSince: now,
	}

	nextGfdId, err := ms.Keeper.GetNextID(ctx, "gfd")
	if err != nil {
		return types.TrustRegistry{}, types.GovernanceFrameworkVersion{}, types.GovernanceFrameworkDocument{}, fmt.Errorf("failed to generate governance framework document ID: %w", err)
	}

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

	if err := ms.TrustRegistryDIDIndex.Set(ctx, tr.Did, tr.Id); err != nil {
		return fmt.Errorf("failed to persist DID index: %w", err)
	}

	if err := ms.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
		return fmt.Errorf("failed to persist GovernanceFrameworkVersion: %w", err)
	}

	if err := ms.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return fmt.Errorf("failed to persist initial GovernanceFrameworkDocument: %w", err)
	}

	return nil
}
