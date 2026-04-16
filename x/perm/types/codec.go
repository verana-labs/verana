package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	// this line is used by starport scaffolding # 1
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgStartPermissionVP{}, "verana/x/perm/MsgStartPermissionVP")
	legacy.RegisterAminoMsg(cdc, &MsgRenewPermissionVP{}, "verana/x/perm/MsgRenewPermissionVP")
	legacy.RegisterAminoMsg(cdc, &MsgSetPermissionVPToValidated{}, "verana/x/perm/MsgSetPermVPValidated")
	legacy.RegisterAminoMsg(cdc, &MsgCancelPermissionVPLastRequest{}, "verana/x/perm/MsgCancelPermVPLastReq")
	legacy.RegisterAminoMsg(cdc, &MsgCreateRootPermission{}, "verana/x/perm/MsgCreateRootPermission")
	legacy.RegisterAminoMsg(cdc, &MsgAdjustPermission{}, "verana/x/perm/MsgAdjustPermission")
	legacy.RegisterAminoMsg(cdc, &MsgRevokePermission{}, "verana/x/perm/MsgRevokePermission")
	legacy.RegisterAminoMsg(cdc, &MsgCreateOrUpdatePermissionSession{}, "verana/x/perm/MsgCreateOrUpdatePermSess")
	legacy.RegisterAminoMsg(cdc, &MsgSlashPermissionTrustDeposit{}, "verana/x/perm/MsgSlashPermTD")
	legacy.RegisterAminoMsg(cdc, &MsgRepayPermissionSlashedTrustDeposit{}, "verana/x/perm/MsgRepayPermSlashedTD")
	legacy.RegisterAminoMsg(cdc, &MsgSelfCreatePermission{}, "verana/x/perm/MsgSelfCreatePermission")
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgStartPermissionVP{},
		&MsgRenewPermissionVP{},
		&MsgSetPermissionVPToValidated{},
		&MsgCancelPermissionVPLastRequest{},
		&MsgCreateRootPermission{},
		&MsgAdjustPermission{},
		&MsgRevokePermission{},
		&MsgCreateOrUpdatePermissionSession{},
		&MsgSlashPermissionTrustDeposit{},
		&MsgRepayPermissionSlashedTrustDeposit{},
		&MsgSelfCreatePermission{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
