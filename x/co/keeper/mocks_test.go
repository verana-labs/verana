package keeper_test

import (
	"context"
	"errors"
	"time"

	"github.com/cosmos/cosmos-sdk/x/group"

	cotypes "github.com/verana-labs/verana/x/co/types"
	gftypes "github.com/verana-labs/verana/x/gf/types"
)

// mockDelegation lets a test pre-program AUTHZ-CHECK-1 results.
// Default zero value returns nil from CheckOperatorAuthorization (auth granted).
type mockDelegation struct {
	err   error
	calls int
}

func (m *mockDelegation) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	m.calls++
	return m.err
}

// mockGroup simulates x/group's CreateGroupWithPolicy. The default zero value
// returns ("cosmos1policy", nil); set policy/err to override.
type mockGroup struct {
	policy   string
	groupID  uint64
	err      error
	gotReq   *group.MsgCreateGroupWithPolicy
	callsCnt int
}

func (m *mockGroup) CreateGroupWithPolicy(_ context.Context, req *group.MsgCreateGroupWithPolicy) (*group.MsgCreateGroupWithPolicyResponse, error) {
	m.callsCnt++
	m.gotReq = req
	if m.err != nil {
		return nil, m.err
	}
	addr := m.policy
	if addr == "" {
		addr = "cosmos1policy"
	}
	gid := m.groupID
	if gid == 0 {
		gid = 1
	}
	return &group.MsgCreateGroupWithPolicyResponse{GroupId: gid, GroupPolicyAddress: addr}, nil
}

// mockGF captures calls to MOD-GF cross-module surface.
// `listResp` is returned by ListVersionsByCorporation; `createErr` short-circuits
// the seed call.
type mockGF struct {
	createErr  error
	createArgs struct {
		corpID                          uint64
		language, docURL, docDigestSRI string
	}
	createCalls int

	listResp  []gftypes.GovernanceFrameworkVersionWithDocs
	listErr   error
	listArgs  struct {
		corpID        uint64
		activeVersion uint32
		activeOnly    bool
		preferredLang string
	}
	listCalls int
}

func (m *mockGF) CreateInitialGFVersionForCorporation(_ context.Context, corpID uint64, language, docURL, docDigestSRI string) error {
	m.createCalls++
	m.createArgs.corpID = corpID
	m.createArgs.language = language
	m.createArgs.docURL = docURL
	m.createArgs.docDigestSRI = docDigestSRI
	return m.createErr
}

func (m *mockGF) ListVersionsByCorporation(_ context.Context, corpID uint64, activeVersion uint32, activeOnly bool, preferredLang string) ([]gftypes.GovernanceFrameworkVersionWithDocs, error) {
	m.listCalls++
	m.listArgs.corpID = corpID
	m.listArgs.activeVersion = activeVersion
	m.listArgs.activeOnly = activeOnly
	m.listArgs.preferredLang = preferredLang
	return m.listResp, m.listErr
}

// errAuthDenied is a convenience for tests that want to simulate DE denial.
var errAuthDenied = errors.New("authorization denied")

// Compile-time interface assertions.
var (
	_ cotypes.DelegationKeeper = (*mockDelegation)(nil)
	_ cotypes.GroupKeeper      = (*mockGroup)(nil)
	_ cotypes.GFKeeper         = (*mockGF)(nil)
)
