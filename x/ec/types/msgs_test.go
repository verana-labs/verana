package types_test

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/ec/types"
)

const (
	tSigner    = "cosmos1hfyt5r4f3rnu5gqrgfwr4446zgn00gdj0nn7dx"
	tOperator  = "cosmos1fvz0kp4jfseea3zyduu78dd5yqcwrarwtxthjn"
	tCorp      = "cosmos14wcc52lpsxwuxxhqjxrhvuumhm0xr6z247un93"
	tAuthority = "cosmos1z39xu0w27yfq58dmqyk7efuyqt43kvfc0jdte2"
)

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	require.NoError(t, (&types.MsgUpdateParams{Authority: tAuthority, Params: types.DefaultParams()}).ValidateBasic())
	require.ErrorIs(t, (&types.MsgUpdateParams{Authority: ""}).ValidateBasic(), sdkerrors.ErrInvalidAddress)
	require.ErrorIs(t, (&types.MsgUpdateParams{Authority: "cosmos1signer", Params: types.DefaultParams()}).ValidateBasic(), sdkerrors.ErrInvalidAddress)
}

func TestMsgCreateEcosystem_ValidateBasic(t *testing.T) {
	base := func() *types.MsgCreateEcosystem {
		return &types.MsgCreateEcosystem{
			Corporation:  tCorp,
			Operator:     tOperator,
			Did:          "did:example:1",
			Language:     "en",
			DocUrl:       "https://example.com/ec.pdf",
			DocDigestSri: "sha256-aGVsbG8=",
		}
	}
	require.NoError(t, base().ValidateBasic())

	cases := []struct {
		name    string
		mutate  func(*types.MsgCreateEcosystem)
		errKind error
	}{
		{"empty corp", func(m *types.MsgCreateEcosystem) { m.Corporation = "" }, sdkerrors.ErrInvalidAddress},
		{"bad bech32 corp", func(m *types.MsgCreateEcosystem) { m.Corporation = "cosmos1corp" }, sdkerrors.ErrInvalidAddress},
		{"empty operator", func(m *types.MsgCreateEcosystem) { m.Operator = "" }, sdkerrors.ErrInvalidAddress},
		{"bad bech32 operator", func(m *types.MsgCreateEcosystem) { m.Operator = "cosmos1op" }, sdkerrors.ErrInvalidAddress},
		{"empty did", func(m *types.MsgCreateEcosystem) { m.Did = "" }, types.ErrInvalidDID},
		{"bad did", func(m *types.MsgCreateEcosystem) { m.Did = "not-a-did" }, types.ErrInvalidDID},
		{"empty lang", func(m *types.MsgCreateEcosystem) { m.Language = "" }, types.ErrInvalidLanguage},
		{"bad lang", func(m *types.MsgCreateEcosystem) { m.Language = "x!!" }, types.ErrInvalidLanguage},
		{"empty url", func(m *types.MsgCreateEcosystem) { m.DocUrl = "" }, types.ErrInvalidURL},
		{"bad url", func(m *types.MsgCreateEcosystem) { m.DocUrl = "not a url" }, types.ErrInvalidURL},
		{"empty digest", func(m *types.MsgCreateEcosystem) { m.DocDigestSri = "" }, types.ErrInvalidDigestSRI},
		{"bad digest", func(m *types.MsgCreateEcosystem) { m.DocDigestSri = "md5-deadbeef" }, types.ErrInvalidDigestSRI},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := base()
			tc.mutate(m)
			err := m.ValidateBasic()
			require.Error(t, err)
			require.ErrorIs(t, err, tc.errKind)
		})
	}
}

func TestMsgUpdateEcosystem_ValidateBasic(t *testing.T) {
	base := func() *types.MsgUpdateEcosystem {
		return &types.MsgUpdateEcosystem{Corporation: tCorp, Operator: tOperator, Id: 1, Did: "did:example:rotated"}
	}
	require.NoError(t, base().ValidateBasic())

	cases := []struct {
		name    string
		mutate  func(*types.MsgUpdateEcosystem)
		errKind error
	}{
		{"empty corp", func(m *types.MsgUpdateEcosystem) { m.Corporation = "" }, sdkerrors.ErrInvalidAddress},
		{"bad bech32 corp", func(m *types.MsgUpdateEcosystem) { m.Corporation = "cosmos1corp" }, sdkerrors.ErrInvalidAddress},
		{"empty operator", func(m *types.MsgUpdateEcosystem) { m.Operator = "" }, sdkerrors.ErrInvalidAddress},
		{"id zero", func(m *types.MsgUpdateEcosystem) { m.Id = 0 }, types.ErrInvalidSubject},
		{"empty did", func(m *types.MsgUpdateEcosystem) { m.Did = "" }, types.ErrInvalidDID},
		{"bad did", func(m *types.MsgUpdateEcosystem) { m.Did = "not-a-did" }, types.ErrInvalidDID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := base()
			tc.mutate(m)
			err := m.ValidateBasic()
			require.Error(t, err)
			require.ErrorIs(t, err, tc.errKind)
		})
	}
}

func TestMsgArchiveEcosystem_ValidateBasic(t *testing.T) {
	base := func() *types.MsgArchiveEcosystem {
		return &types.MsgArchiveEcosystem{Corporation: tCorp, Operator: tOperator, Id: 1, Archive: true}
	}
	require.NoError(t, base().ValidateBasic())

	cases := []struct {
		name    string
		mutate  func(*types.MsgArchiveEcosystem)
		errKind error
	}{
		{"empty corp", func(m *types.MsgArchiveEcosystem) { m.Corporation = "" }, sdkerrors.ErrInvalidAddress},
		{"bad bech32 corp", func(m *types.MsgArchiveEcosystem) { m.Corporation = "cosmos1corp" }, sdkerrors.ErrInvalidAddress},
		{"empty operator", func(m *types.MsgArchiveEcosystem) { m.Operator = "" }, sdkerrors.ErrInvalidAddress},
		{"id zero", func(m *types.MsgArchiveEcosystem) { m.Id = 0 }, types.ErrInvalidSubject},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := base()
			tc.mutate(m)
			err := m.ValidateBasic()
			require.Error(t, err)
			require.ErrorIs(t, err, tc.errKind)
		})
	}

	// Note: proto3 bool cannot distinguish absent from explicit-false on the
	// wire. Spec [MOD-ES-MSG-3-1] marks `archive` as mandatory; this is
	// unenforceable at proto level. The keeper's idempotency-abort branch
	// catches submissions of `archive=false` against un-archived ecosystems.
	_ = tSigner // reserved for future negative tests
}
