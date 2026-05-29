package keeper_test

import (
	"context"
	"errors"
	"time"

	gftypes "github.com/verana-labs/verana/x/gf/types"
	"github.com/verana-labs/verana/x/ec/types"
)

// mockDelegation: configurable AUTHZ-CHECK result. Default returns nil (auth granted).
type mockDelegation struct {
	err   error
	calls int
}

func (m *mockDelegation) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	m.calls++
	return m.err
}

// mockCorporation: AUTHZ-CHECK-5 resolver. Pre-program signer → CorporationView.
// If addr is not registered, returns (zero, false) — that surfaces as
// ErrCorporationNotRegistered in the keeper.
type mockCorporation struct {
	byAddr map[string]types.CorporationView
}

func newMockCorporation() *mockCorporation {
	return &mockCorporation{byAddr: map[string]types.CorporationView{}}
}

func (m *mockCorporation) register(addr string, id uint64) {
	m.byAddr[addr] = types.CorporationView{Id: id, PolicyAddress: addr, Language: "en", ActiveVersion: 1}
}

func (m *mockCorporation) ResolveByPolicyAddress(_ context.Context, addr string) (types.CorporationView, bool) {
	v, ok := m.byAddr[addr]
	return v, ok
}

func (m *mockCorporation) GetByID(_ context.Context, id uint64) (types.CorporationView, bool) {
	for _, v := range m.byAddr {
		if v.Id == id {
			return v, true
		}
	}
	return types.CorporationView{}, false
}

// mockGF: cross-module gf surface. Captures CreateInitialGFVersionForEcosystem
// and ListVersionsByEcosystem call args + lets tests preprogram list responses
// and errors.
type mockGF struct {
	createErr   error
	createCalls int
	createArgs  struct {
		ecID                            uint64
		language, docURL, docDigestSRI string
	}

	listResp  []gftypes.GovernanceFrameworkVersionWithDocs
	listErr   error
	listCalls int
	listArgs  struct {
		ecID          uint64
		activeVersion uint32
		activeOnly    bool
		preferredLang string
	}
}

func (m *mockGF) CreateInitialGFVersionForEcosystem(_ context.Context, ecID uint64, language, docURL, docDigestSRI string) error {
	m.createCalls++
	m.createArgs.ecID = ecID
	m.createArgs.language = language
	m.createArgs.docURL = docURL
	m.createArgs.docDigestSRI = docDigestSRI
	return m.createErr
}

func (m *mockGF) ListVersionsByEcosystem(_ context.Context, ecID uint64, activeVersion uint32, activeOnly bool, preferredLang string) ([]gftypes.GovernanceFrameworkVersionWithDocs, error) {
	m.listCalls++
	m.listArgs.ecID = ecID
	m.listArgs.activeVersion = activeVersion
	m.listArgs.activeOnly = activeOnly
	m.listArgs.preferredLang = preferredLang
	return m.listResp, m.listErr
}

// errAuthDenied is a convenience for tests simulating DE denial.
var errAuthDenied = errors.New("authorization denied")

// Compile-time interface assertions.
var (
	_ types.DelegationKeeper  = (*mockDelegation)(nil)
	_ types.CorporationKeeper = (*mockCorporation)(nil)
	_ types.GFKeeper          = (*mockGF)(nil)
)
