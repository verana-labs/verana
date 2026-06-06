package keeper

import (
	"context"
	"fmt"

	"github.com/verana-labs/verana/x/de/types"
)

// corpKeeperRef is a stable container for the CorporationKeeper interface,
// indirected behind a pointer so all by-value copies of Keeper (held by the msg
// server, query server, etc.) see the same instance. SetCorporationKeeper
// writes to this container after construction — required because MOD-CO depends
// on MOD-DE's DelegationKeeper while MOD-DE needs MOD-CO for AUTHZ-CHECK-5
// (cycle break per #308).
type corpKeeperRef struct {
	K types.CorporationKeeper
}

// StubCorporationKeeper is the pre-wiring default. If a delegable MOD-DE handler
// somehow runs before SetCorporationKeeper has been invoked, AUTHZ-CHECK-5
// aborts cleanly instead of nil-panicking. Replaced via Keeper.SetCorporationKeeper.
type StubCorporationKeeper struct{}

func (StubCorporationKeeper) ResolveCorporationByPolicyAddress(_ context.Context, policyAddress string) (types.CorporationView, error) {
	return types.CorporationView{}, fmt.Errorf("corporation keeper not wired: cannot resolve signing account %s", policyAddress)
}
