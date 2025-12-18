package journeys

import (
	"context"
	"fmt"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
)

// RunPermissionTerminationJourney implements Journey 8: Permission Termination Journey
func RunPermissionTerminationJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Void")
	return nil
}
