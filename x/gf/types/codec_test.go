package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/gf/types"
)

func TestRegisterInterfaces(t *testing.T) {
	registry := cdctypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	// Every Msg type the module exposes must be resolvable through the
	// registered sdk.Msg interface.
	for _, name := range []string{
		"/verana.gf.v1.MsgUpdateParams",
		"/verana.gf.v1.MsgAddGovernanceFrameworkDocument",
		"/verana.gf.v1.MsgIncreaseActiveGovernanceFrameworkVersion",
	} {
		msg, err := registry.Resolve(name)
		require.NoError(t, err, "expected %s registered", name)
		_, ok := msg.(sdk.Msg)
		require.True(t, ok, "%s must implement sdk.Msg", name)
	}
}

func TestRegisterLegacyAminoCodec(t *testing.T) {
	cdc := codec.NewLegacyAmino()
	require.NotPanics(t, func() {
		types.RegisterLegacyAminoCodec(cdc)
	})
}
