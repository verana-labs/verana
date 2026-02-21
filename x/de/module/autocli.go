package de

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"github.com/verana-labs/verana/x/de/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Query_serviceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					// Skip autocli for this RPC — custom command provided in cli_query.go
					// to work around gogo/pulsar proto codec mismatch in autocli JSON rendering.
					RpcMethod: "ListOperatorAuthorizations",
					Skip:      true,
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
					RpcMethod: "GrantOperatorAuthorization",
					Use:       "grant-operator-authz [authority] [grantee]",
					Short:     "Grant operator authorization to a grantee on behalf of an authority",
					Long:      "[MOD-DE-MSG-3] Grant operator authorization. The grantee receives authorization to execute specified message types on behalf of the authority. Optionally includes a fee grant.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "authority"},
						{ProtoField: "grantee"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"operator": {
							Name:  "operator",
							Usage: "optional operator account authorized to run this Msg",
						},
						"msg_types": {
							Name:  "msg-types",
							Usage: "comma-separated list of VPR delegable message type URLs",
						},
						"expiration": {
							Name:  "expiration",
							Usage: "authorization expiration timestamp (RFC3339)",
						},
						"authz_spend_limit": {
							Name:  "authz-spend-limit",
							Usage: "maximum spendable amount (e.g. 100stake)",
						},
						"authz_spend_limit_period": {
							Name:  "authz-spend-limit-period",
							Usage: "reset period for authz spend limit",
						},
						"with_feegrant": {
							Name:         "with-feegrant",
							Usage:        "whether to also grant fee allowance",
							DefaultValue: "false",
						},
						"feegrant_spend_limit": {
							Name:  "feegrant-spend-limit",
							Usage: "maximum fee amount (e.g. 100stake). Ignored if --with-feegrant is false",
						},
						"feegrant_spend_limit_period": {
							Name:  "feegrant-spend-limit-period",
							Usage: "reset period for fee spend limit. Ignored if --with-feegrant is false",
						},
					},
				},
				{
					RpcMethod: "RevokeOperatorAuthorization",
					Use:       "revoke-operator-authz [authority] [grantee]",
					Short:     "Revoke operator authorization for a grantee",
					Long:      "[MOD-DE-MSG-4] Revoke operator authorization. Removes the authorization entry and any associated fee grant for the given authority/grantee pair.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "authority"},
						{ProtoField: "grantee"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"operator": {
							Name:  "operator",
							Usage: "optional operator account authorized to run this Msg",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
