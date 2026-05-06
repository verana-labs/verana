package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/verana-labs/verana/testutil/keeper"
	cstypes "github.com/verana-labs/verana/x/cs/types"
	detypes "github.com/verana-labs/verana/x/de/types"
	ditypes "github.com/verana-labs/verana/x/di/types"
	permtypes "github.com/verana-labs/verana/x/perm/types"
	tdtypes "github.com/verana-labs/verana/x/td/types"
	trtypes "github.com/verana-labs/verana/x/tr/types"
	xrtypes "github.com/verana-labs/verana/x/xr/types"
)

// Compile-time assertions: StatefulBankMock satisfies every module's
// BankKeeper interface. If any module's interface grows, these will fail.
var (
	_ permtypes.BankKeeper = (*testkeeper.StatefulBankMock)(nil)
	_ tdtypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
	_ cstypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
	_ trtypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
	_ detypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
	_ xrtypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
	_ ditypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
)

// Test addresses, reused across tests.
var (
	denom = "uvna"
	alice = sdk.AccAddress([]byte("alice_______________"))
	bob   = sdk.AccAddress([]byte("bob_________________"))
)

func newMock(t *testing.T) *testkeeper.StatefulBankMock {
	t.Helper()
	return testkeeper.NewStatefulBankMock(testkeeper.DefaultModuleAddrs())
}

// captureT implements require.TestingT and records whether an assertion
// would have failed. Used to test RequireBalanceDelta itself.
type captureT struct {
	failed bool
	last   string
}

func (c *captureT) Errorf(format string, args ...interface{}) {
	c.failed = true
}
func (c *captureT) FailNow() { c.failed = true }

// Suppress unused-variable warnings for package-level vars used by future tasks.
var (
	_ = denom
	_ = alice
	_ = bob
	_ require.TestingT = (*captureT)(nil)
)
