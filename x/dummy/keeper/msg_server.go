package keeper

import (
	"context"
	"fmt"
	"github.com/verana-labs/verana-blockchain/x/dummy/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (ms msgServer) StartPermissionVP(goCtx context.Context, msg *types.MsgStartPermissionVP) (*types.MsgStartPermissionVPResponse, error) {
	fmt.Println("its executing....")
	return &types.MsgStartPermissionVPResponse{
		PermissionId: 0,
	}, nil
}
