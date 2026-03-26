package xr

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"github.com/verana-labs/verana/x/xr/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "GetExchangeRate",
					Use:       "get-exchange-rate [id]",
					Short:     "Query an exchange rate by id or asset pair",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id", Optional: true},
					},
				},
				{
					RpcMethod: "ListExchangeRates",
					Use:       "list-exchange-rates",
					Short:     "List exchange rates with optional filters",
				},
				{
					RpcMethod: "GetPrice",
					Use:       "get-price [base_asset_type] [base_asset] [quote_asset_type] [quote_asset] [amount]",
					Short:     "Compute converted price using an exchange rate",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "base_asset_type"},
						{ProtoField: "base_asset"},
						{ProtoField: "quote_asset_type"},
						{ProtoField: "quote_asset"},
						{ProtoField: "amount"},
					},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod: "CreateExchangeRate",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod: "UpdateExchangeRate",
					Use:       "update-exchange-rate [authority] [id] [rate]",
					Short:     "Update an existing exchange rate",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "authority"},
						{ProtoField: "id"},
						{ProtoField: "rate"},
					},
				},
				{
					RpcMethod: "ToggleExchangeRateState",
					Skip:      true, // skipped because authority gated
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
