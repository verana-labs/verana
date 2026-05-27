package keeper

import (
	"context"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MockTrustDepositKeeper is a no-op mock satisfying x/cs / x/perm /
// trustDeposit-consumer interfaces in test wiring. Extracted from the
// pre-rename testutil/keeper/trustregistry.go.
type MockTrustDepositKeeper struct{}

func (m *MockTrustDepositKeeper) AdjustTrustDeposit(_ sdk.Context, _ string, _ int64, _ string) error {
	return nil
}

func (m *MockTrustDepositKeeper) AdjustTrustDepositOnBehalf(_ sdk.Context, _ string, _ sdk.AccAddress, _ int64) error {
	return nil
}

func (m *MockTrustDepositKeeper) GetTrustDepositRate(_ sdk.Context) math.LegacyDec {
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

func (m *MockTrustDepositKeeper) GetUserAgentRewardRate(_ sdk.Context) math.LegacyDec {
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

func (m *MockTrustDepositKeeper) GetWalletUserAgentRewardRate(_ sdk.Context) math.LegacyDec {
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

func (m *MockTrustDepositKeeper) BurnEcosystemSlashedTrustDeposit(_ sdk.Context, _ string, _ uint64) error {
	return nil
}

// MockDelegationKeeper is a mock implementation of the DelegationKeeper
// interface used by cs / perm / td / ec tests. By default it allows all
// operator authorizations (ErrToReturn is nil). Set ErrToReturn to simulate
// authorization failures. Extracted from the pre-rename
// testutil/keeper/trustregistry.go.
type MockDelegationKeeper struct {
	ErrToReturn error

	RemovePermFromVSOARemainingPerms []uint64
	GetVSOAPermissionsResult         []uint64

	AddPermToVSOACalls      []AddPermToVSOACall
	RemovePermFromVSOACalls []RemovePermFromVSOACall
	GrantFeeAllowanceCalls  []GrantFeeAllowanceCall
	RevokeFeeAllowanceCalls []RevokeFeeAllowanceCall
}

type AddPermToVSOACall struct {
	Authority, VsOperator string
	PermID                uint64
}

type RemovePermFromVSOACall struct {
	Authority, VsOperator string
	PermID                uint64
}

type GrantFeeAllowanceCall struct {
	Authority, Grantee string
	MsgTypes           []string
	Expiration         *time.Time
	SpendLimit         sdk.Coins
	Period             *time.Duration
}

type RevokeFeeAllowanceCall struct {
	Authority, Grantee string
}

func (m *MockDelegationKeeper) Reset() {
	m.AddPermToVSOACalls = nil
	m.RemovePermFromVSOACalls = nil
	m.GrantFeeAllowanceCalls = nil
	m.RevokeFeeAllowanceCalls = nil
}

func (m *MockDelegationKeeper) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) CheckVSOperatorAuthorization(_ context.Context, _, _ string) error {
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) AddPermToVSOA(_ context.Context, authority, vsOperator string, permID uint64) error {
	m.AddPermToVSOACalls = append(m.AddPermToVSOACalls, AddPermToVSOACall{Authority: authority, VsOperator: vsOperator, PermID: permID})
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) RemovePermFromVSOA(_ context.Context, authority, vsOperator string, permID uint64) ([]uint64, error) {
	m.RemovePermFromVSOACalls = append(m.RemovePermFromVSOACalls, RemovePermFromVSOACall{Authority: authority, VsOperator: vsOperator, PermID: permID})
	return m.RemovePermFromVSOARemainingPerms, m.ErrToReturn
}

func (m *MockDelegationKeeper) GetVSOAPermissions(_ context.Context, _, _ string) ([]uint64, error) {
	return m.GetVSOAPermissionsResult, m.ErrToReturn
}

func (m *MockDelegationKeeper) HasOperatorAuthorization(_ context.Context, _, _ string) (bool, error) {
	return false, m.ErrToReturn
}

func (m *MockDelegationKeeper) GrantFeeAllowance(_ context.Context, authority string, grantee string, msgTypes []string, expiration *time.Time, spendLimit sdk.Coins, period *time.Duration) error {
	m.GrantFeeAllowanceCalls = append(m.GrantFeeAllowanceCalls, GrantFeeAllowanceCall{
		Authority:  authority,
		Grantee:    grantee,
		MsgTypes:   msgTypes,
		Expiration: expiration,
		SpendLimit: spendLimit,
		Period:     period,
	})
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) RevokeFeeAllowance(_ context.Context, authority, grantee string) error {
	m.RevokeFeeAllowanceCalls = append(m.RevokeFeeAllowanceCalls, RevokeFeeAllowanceCall{Authority: authority, Grantee: grantee})
	return m.ErrToReturn
}
