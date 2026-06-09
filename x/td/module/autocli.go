package trustdeposit

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/verana-labs/verana/api/verana/td/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "GetTrustDeposit",
					Use:       "get-trust-deposit [corporation-id]",
					Short:     "Query trust deposit for a corporation",
					Long:      "Get the trust deposit information for a given corporation id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "corporation_id",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod: "ReclaimTrustDepositYield",
					Use:       "reclaim-yield [corporation]",
					Short:     "Reclaim earned interest from trust deposits",
					Long:      "Reclaim any available interest earned from trust deposits. The interest is calculated based on share value and current deposit amount.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
					},
				},
				{
					RpcMethod: "RepaySlashedTrustDeposit",
					Use:       "repay-slashed-td [corporation] [deposit]",
					Short:     "Repay slashed trust deposit",
					Long:      "Repay the outstanding slashed trust deposit. The deposit must exactly match the outstanding slashed amount.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
						{ProtoField: "deposit"},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}

