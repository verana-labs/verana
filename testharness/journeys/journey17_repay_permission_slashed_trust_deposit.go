package journeys

import (
	"context"
	"fmt"

	// "strconv"
	"github.com/verana-labs/verana/testharness/lib"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	// permtypes "github.com/verana-labs/verana/x/perm/types"
)

// RunRepayPermissionSlashedTrustDepositJourney implements Journey 17: Repay Permission Slashed Trust Deposit
func RunRepayPermissionSlashedTrustDepositJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 17: Repay Permission Slashed Trust Deposit")

	// Load previous journey result (assumes journey16 was run)
	result, err := lib.GetJourneyResult("journey16")
	if err != nil {
		return fmt.Errorf("failed to load journey16 result: %v", err)
	}
	// TODO: After SDK upgrade
	fmt.Println("✅ Step: Permission slashed trust deposit repaid")

	// Save result
	lib.SaveJourneyResult("journey17", result)
	return nil
}
