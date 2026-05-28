package co_test

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/x/group"

	cotypes "github.com/verana-labs/verana/x/co/types"
	gftypes "github.com/verana-labs/verana/x/gf/types"
)

type stubDelegation struct{}

func (stubDelegation) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}

type stubGroup struct{}

func (stubGroup) CreateGroupWithPolicy(_ context.Context, _ *group.MsgCreateGroupWithPolicy) (*group.MsgCreateGroupWithPolicyResponse, error) {
	return &group.MsgCreateGroupWithPolicyResponse{GroupId: 1, GroupPolicyAddress: "cosmos1policy"}, nil
}

type stubGF struct{}

func (stubGF) CreateInitialGFVersionForCorporation(_ context.Context, _ uint64, _, _, _ string) error {
	return nil
}
func (stubGF) ListVersionsByCorporation(_ context.Context, _ uint64, _ uint32, _ bool, _ string) ([]gftypes.GovernanceFrameworkVersionWithDocs, error) {
	return nil, nil
}

// stubGFDelegation + stubEcosystem satisfy the MOD-GF NewKeeper signature for
// tests that need to build a real GF concrete keeper (post-cycle-break).
type stubGFDelegation struct{}

func (stubGFDelegation) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}

type stubEcosystem struct{}

func (stubEcosystem) GetEcosystemView(_ context.Context, _ uint64) (gftypes.EcosystemView, bool) {
	return gftypes.EcosystemView{}, false
}
func (stubEcosystem) SetEcosystemActiveVersion(_ context.Context, _ uint64, _ uint32) error {
	return nil
}

// Compile-time interface assertions.
var (
	_ cotypes.DelegationKeeper = stubDelegation{}
	_ cotypes.GroupKeeper      = stubGroup{}
	_ cotypes.GFKeeper         = stubGF{}
	_ gftypes.DelegationKeeper = stubGFDelegation{}
	_ gftypes.EcosystemKeeper  = stubEcosystem{}
)
