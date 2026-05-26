package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/co/types"
)

func TestRegisterInterfaces(t *testing.T) {
	registry := cdctypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	for _, name := range []string{
		"/verana.co.v1.MsgUpdateParams",
		"/verana.co.v1.MsgCreateCorporation",
		"/verana.co.v1.MsgUpdateCorporation",
	} {
		msg, err := registry.Resolve(name)
		require.NoError(t, err, "expected %s registered", name)
		_, ok := msg.(sdk.Msg)
		require.True(t, ok, "%s must implement sdk.Msg", name)
	}
}

func TestRegisterLegacyAminoCodec_NoPanic(t *testing.T) {
	amino := codec.NewLegacyAmino()
	require.NotPanics(t, func() { types.RegisterLegacyAminoCodec(amino) })
}
