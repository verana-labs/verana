package gf_test

import (
	"context"
	"time"

	"github.com/verana-labs/verana/x/gf/types"
)

// Local stub keepers for module-level tests (the module pkg can't import
// x/gf/keeper without breaking the layer boundary; these mirror the runtime
// stubs).

type stubDelegationKeeper struct{}

func (stubDelegationKeeper) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}

type stubEcosystemKeeper struct{}

func (*stubEcosystemKeeper) GetEcosystemView(_ context.Context, _ uint64) (types.EcosystemView, bool) {
	return types.EcosystemView{}, false
}
func (*stubEcosystemKeeper) SetEcosystemActiveVersion(_ context.Context, _ uint64, _ uint32) error {
	return nil
}

type stubCorporationKeeper struct{}

func (*stubCorporationKeeper) ResolveByPolicyAddress(_ context.Context, _ string) (types.CorporationView, bool) {
	return types.CorporationView{}, false
}
func (*stubCorporationKeeper) GetByID(_ context.Context, _ uint64) (types.CorporationView, bool) {
	return types.CorporationView{}, false
}
func (*stubCorporationKeeper) SetActiveVersion(_ context.Context, _ uint64, _ uint32) error {
	return nil
}
