package types_test

import (
	"testing"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	group "github.com/cosmos/cosmos-sdk/x/group"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/co/types"
)

// Valid bech32 fixtures (deterministically generated from labels).
const (
	tSigner    = "cosmos1hfyt5r4f3rnu5gqrgfwr4446zgn00gdj0nn7dx"
	tOperator  = "cosmos1fvz0kp4jfseea3zyduu78dd5yqcwrarwtxthjn"
	tCorp      = "cosmos14wcc52lpsxwuxxhqjxrhvuumhm0xr6z247un93"
	tMember    = "cosmos1jpjwc8r9y5xqpj7q3w9c2qhda0ednjcmxvtna4"
	tAuthority = "cosmos1z39xu0w27yfq58dmqyk7efuyqt43kvfc0jdte2"
)

func validDecisionPolicy(t *testing.T) *cdctypes.Any {
	t.Helper()
	a, err := cdctypes.NewAnyWithValue(&group.ThresholdDecisionPolicy{
		Threshold: "1",
		Windows: &group.DecisionPolicyWindows{
			VotingPeriod: 0,
		},
	})
	require.NoError(t, err)
	return a
}

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		require.NoError(t, (&types.MsgUpdateParams{Authority: tAuthority, Params: types.DefaultParams()}).ValidateBasic())
	})
	t.Run("empty authority", func(t *testing.T) {
		require.ErrorIs(t, (&types.MsgUpdateParams{Authority: ""}).ValidateBasic(), sdkerrors.ErrInvalidAddress)
	})
	t.Run("malformed bech32 authority", func(t *testing.T) {
		require.ErrorIs(t, (&types.MsgUpdateParams{Authority: "cosmos1signer", Params: types.DefaultParams()}).ValidateBasic(), sdkerrors.ErrInvalidAddress)
	})
}

func TestMsgCreateCorporation_ValidateBasic(t *testing.T) {
	base := func() *types.MsgCreateCorporation {
		return &types.MsgCreateCorporation{
			Signer:         tSigner,
			Members:        []types.Member{{Address: tMember, Weight: "1"}},
			DecisionPolicy: validDecisionPolicy(t),
			Did:            "did:example:1",
			Language:       "en",
			DocUrl:         "https://example.com/cgf.pdf",
			DocDigestSri:   "sha256-aGVsbG8=",
		}
	}
	require.NoError(t, base().ValidateBasic())

	cases := []struct {
		name    string
		mutate  func(*types.MsgCreateCorporation)
		errKind error // optional: error to assert via ErrorIs
	}{
		{"empty signer", func(m *types.MsgCreateCorporation) { m.Signer = "" }, sdkerrors.ErrInvalidAddress},
		{"malformed bech32 signer", func(m *types.MsgCreateCorporation) { m.Signer = "cosmos1signer" }, sdkerrors.ErrInvalidAddress},
		{"no members", func(m *types.MsgCreateCorporation) { m.Members = nil }, types.ErrInvalidMembers},
		{"member no addr", func(m *types.MsgCreateCorporation) { m.Members = []types.Member{{Weight: "1"}} }, types.ErrInvalidMembers},
		{"member malformed bech32", func(m *types.MsgCreateCorporation) { m.Members = []types.Member{{Address: "cosmos1m", Weight: "1"}} }, sdkerrors.ErrInvalidAddress},
		{"member no weight", func(m *types.MsgCreateCorporation) { m.Members = []types.Member{{Address: tMember}} }, types.ErrInvalidMembers},
		{"nil decision_policy", func(m *types.MsgCreateCorporation) { m.DecisionPolicy = nil }, types.ErrInvalidDecisionPolicy},
		{"empty did", func(m *types.MsgCreateCorporation) { m.Did = "" }, types.ErrInvalidDID},
		{"bad did", func(m *types.MsgCreateCorporation) { m.Did = "not-a-did" }, types.ErrInvalidDID},
		{"empty lang", func(m *types.MsgCreateCorporation) { m.Language = "" }, types.ErrInvalidLanguage},
		{"bad lang", func(m *types.MsgCreateCorporation) { m.Language = "x!!" }, types.ErrInvalidLanguage},
		{"empty url", func(m *types.MsgCreateCorporation) { m.DocUrl = "" }, types.ErrInvalidURL},
		{"bad url", func(m *types.MsgCreateCorporation) { m.DocUrl = "not a url" }, types.ErrInvalidURL},
		{"empty digest", func(m *types.MsgCreateCorporation) { m.DocDigestSri = "" }, types.ErrInvalidDigestSRI},
		{"bad digest", func(m *types.MsgCreateCorporation) { m.DocDigestSri = "md5-deadbeef" }, types.ErrInvalidDigestSRI},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := base()
			tc.mutate(m)
			err := m.ValidateBasic()
			require.Error(t, err)
			if tc.errKind != nil {
				require.ErrorIs(t, err, tc.errKind)
			}
		})
	}
}

func TestMsgUpdateCorporation_ValidateBasic(t *testing.T) {
	base := func() *types.MsgUpdateCorporation {
		return &types.MsgUpdateCorporation{Corporation: tCorp, Operator: tOperator, Did: "did:example:2"}
	}
	require.NoError(t, base().ValidateBasic())

	cases := []struct {
		name    string
		mutate  func(*types.MsgUpdateCorporation)
		errKind error
	}{
		{"empty corporation", func(m *types.MsgUpdateCorporation) { m.Corporation = "" }, sdkerrors.ErrInvalidAddress},
		{"malformed bech32 corporation", func(m *types.MsgUpdateCorporation) { m.Corporation = "cosmos1corp" }, sdkerrors.ErrInvalidAddress},
		{"empty operator", func(m *types.MsgUpdateCorporation) { m.Operator = "" }, sdkerrors.ErrInvalidAddress},
		{"malformed bech32 operator", func(m *types.MsgUpdateCorporation) { m.Operator = "cosmos1op" }, sdkerrors.ErrInvalidAddress},
		{"empty did", func(m *types.MsgUpdateCorporation) { m.Did = "" }, types.ErrInvalidDID},
		{"bad did", func(m *types.MsgUpdateCorporation) { m.Did = "not-a-did" }, types.ErrInvalidDID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := base()
			tc.mutate(m)
			err := m.ValidateBasic()
			require.Error(t, err)
			if tc.errKind != nil {
				require.ErrorIs(t, err, tc.errKind)
			}
		})
	}
}
