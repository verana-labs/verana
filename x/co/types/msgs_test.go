package types_test

import (
	"testing"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	group "github.com/cosmos/cosmos-sdk/x/group"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/co/types"
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
	require.NoError(t, (&types.MsgUpdateParams{Authority: "cosmos1xxx", Params: types.DefaultParams()}).ValidateBasic())
	require.Error(t, (&types.MsgUpdateParams{Authority: ""}).ValidateBasic())
}

func TestMsgCreateCorporation_ValidateBasic(t *testing.T) {
	base := func() *types.MsgCreateCorporation {
		return &types.MsgCreateCorporation{
			Signer:         "cosmos1signer",
			Members:        []types.Member{{Address: "cosmos1m", Weight: "1"}},
			DecisionPolicy: validDecisionPolicy(t),
			Did:            "did:example:1",
			Language:       "en",
			DocUrl:         "https://example.com/cgf.pdf",
			DocDigestSri:   "sha256-aGVsbG8=",
		}
	}
	require.NoError(t, base().ValidateBasic())

	cases := []struct {
		name   string
		mutate func(*types.MsgCreateCorporation)
	}{
		{"empty signer", func(m *types.MsgCreateCorporation) { m.Signer = "" }},
		{"no members", func(m *types.MsgCreateCorporation) { m.Members = nil }},
		{"member no addr", func(m *types.MsgCreateCorporation) { m.Members = []types.Member{{Weight: "1"}} }},
		{"member no weight", func(m *types.MsgCreateCorporation) { m.Members = []types.Member{{Address: "cosmos1m"}} }},
		{"nil decision_policy", func(m *types.MsgCreateCorporation) { m.DecisionPolicy = nil }},
		{"empty did", func(m *types.MsgCreateCorporation) { m.Did = "" }},
		{"bad did", func(m *types.MsgCreateCorporation) { m.Did = "not-a-did" }},
		{"empty lang", func(m *types.MsgCreateCorporation) { m.Language = "" }},
		{"bad lang", func(m *types.MsgCreateCorporation) { m.Language = "x!!" }},
		{"empty url", func(m *types.MsgCreateCorporation) { m.DocUrl = "" }},
		{"bad url", func(m *types.MsgCreateCorporation) { m.DocUrl = "not a url" }},
		{"empty digest", func(m *types.MsgCreateCorporation) { m.DocDigestSri = "" }},
		{"bad digest", func(m *types.MsgCreateCorporation) { m.DocDigestSri = "md5-deadbeef" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := base()
			tc.mutate(m)
			require.Error(t, m.ValidateBasic())
		})
	}
}

func TestMsgUpdateCorporation_ValidateBasic(t *testing.T) {
	base := func() *types.MsgUpdateCorporation {
		return &types.MsgUpdateCorporation{Corporation: "cosmos1corp", Operator: "cosmos1op", Did: "did:example:2"}
	}
	require.NoError(t, base().ValidateBasic())

	for name, mutate := range map[string]func(*types.MsgUpdateCorporation){
		"empty corporation": func(m *types.MsgUpdateCorporation) { m.Corporation = "" },
		"empty operator":    func(m *types.MsgUpdateCorporation) { m.Operator = "" },
		"empty did":         func(m *types.MsgUpdateCorporation) { m.Did = "" },
		"bad did":           func(m *types.MsgUpdateCorporation) { m.Did = "not-a-did" },
	} {
		t.Run(name, func(t *testing.T) {
			m := base()
			mutate(m)
			require.Error(t, m.ValidateBasic())
		})
	}
}
