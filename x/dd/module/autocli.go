package diddirectory

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/verana-labs/verana/api/verana/dd/v1"
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
					RpcMethod: "ListDIDs",
					Use:       "list-dids",
					Short:     "List DIDs with optional filtering",
					Long:      "List DIDs in the directory with optional filtering by controller, changed time, expiration status, and pagination",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"account": {
							Name:         "account",
							Usage:        "Filter by controller account address",
							DefaultValue: "",
						},
						"changed": {
							Name:         "changed",
							Usage:        "Filter by changed time (RFC3339 format)",
							DefaultValue: "",
						},
						"expired": {
							Name:         "expired",
							Usage:        "Show expired services",
							DefaultValue: "false",
						},
						"over_grace": {
							Name:         "over-grace",
							Usage:        "Show services over grace period",
							DefaultValue: "false",
						},
						"response_max_size": {
							Name:         "max-results",
							Usage:        "Maximum number of results (1-1024, default 64)",
							DefaultValue: "64",
						},
					},
				},
				{
					RpcMethod: "GetDID",
					Use:       "get-did [did]",
					Short:     "Get details of a DID entry",
					Long:      "Get the full details of a DID entry from the directory",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "did"},
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
					RpcMethod: "AddDID",
					Use:       "add-did [did] [years]",
					Short:     "Add a new DID to the directory",
					Long:      "Add a new DID to the directory with optional years parameter (1-31, default 1)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "did"},
						{
							ProtoField: "years",
							Optional:   true,
						},
					},
				},
				{
					RpcMethod: "RenewDID",
					Use:       "renew-did [did] [years]",
					Short:     "Renew an existing DID registration",
					Long:      "Renew an existing DID registration for additional years (1-31, default 1). Must be called by the DID controller.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "did"},
						{
							ProtoField: "years",
							Optional:   true,
						},
					},
				},
				{
					RpcMethod: "RemoveDID",
					Use:       "remove-did [did]",
					Short:     "Remove a DID from the directory",
					Long:      "Remove a DID from the directory. Only the controller can remove before grace period, anyone can remove after grace period has passed.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "did"},
					},
				},
				{
					RpcMethod: "TouchDID",
					Use:       "touch-did [did]",
					Short:     "Update the last modified time of a DID",
					Long:      "Update the last modified time of a DID to indicate it should be reindexed by DID resolvers and crawlers",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "did"},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
