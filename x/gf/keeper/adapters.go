package keeper

import (
	"context"
	"errors"

	gftypes "github.com/verana-labs/verana/x/gf/types"
)

// StubCorporationKeeper is the pre-wiring default for the corp-keeper
// reference inside x/gf. It returns "not found" for every lookup so that any
// handler running before SetCorporationKeeper has been invoked aborts cleanly
// with gftypes.ErrSubjectNotFound. After app wiring, MOD-CO's
// CoAsGFCorporationKeeper replaces it via Keeper.SetCorporationKeeper.
type StubCorporationKeeper struct{}

func NewStubCorporationKeeper() gftypes.CorporationKeeper {
	return StubCorporationKeeper{}
}

func (StubCorporationKeeper) ResolveByPolicyAddress(_ context.Context, _ string) (gftypes.CorporationView, bool) {
	return gftypes.CorporationView{}, false
}

func (StubCorporationKeeper) GetByID(_ context.Context, _ uint64) (gftypes.CorporationView, bool) {
	return gftypes.CorporationView{}, false
}

func (StubCorporationKeeper) SetActiveVersion(_ context.Context, _ uint64, _ uint32) error {
	return errors.New("corporation keeper not wired")
}

// EC ↔ GF adapter lives in x/ec/keeper/cross_module.go (EcAsGFEcosystemKeeper)
// and is wired into x/gf post-construction via Keeper.SetEcosystemKeeper, the
// same cycle-break pattern used for CorporationKeeper. Required because both
// x/ec ↔ x/gf depend on each other at the keeper layer.

// StubEcosystemKeeper is the pre-wiring default for the ecosystem-keeper
// reference inside x/gf. Returns (zero, false) for every lookup so handlers
// running before SetEcosystemKeeper has been called fail the
// ecosystem-controller check cleanly.
type StubEcosystemKeeper struct{}

func NewStubEcosystemKeeper() gftypes.EcosystemKeeper {
	return StubEcosystemKeeper{}
}

func (StubEcosystemKeeper) GetEcosystemView(_ context.Context, _ uint64) (gftypes.EcosystemView, bool) {
	return gftypes.EcosystemView{}, false
}

func (StubEcosystemKeeper) SetEcosystemActiveVersion(_ context.Context, _ uint64, _ uint32) error {
	return errors.New("ecosystem keeper not wired")
}
