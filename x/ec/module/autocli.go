package ecosystem // renamed from trustregistry per issue #305

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/verana-labs/verana/api/verana/ec/v1"
)

// AutoCLIOptions implements autocli.HasAutoCLIConfig.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "GetEcosystem",
					Use:       "get-ecosystem [id]",
					Short:     "Get an ecosystem by id, with nested governance framework data",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"active_gf_only":     {Name: "active-gf-only", DefaultValue: "false", Usage: "If true, include only the active GF version"},
						"preferred_language": {Name: "preferred-language", DefaultValue: "", Usage: "Preferred document language"},
					},
				},
				{
					RpcMethod: "ListEcosystems",
					Use:       "list-ecosystems",
					Short:     "List ecosystems with optional filters",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation_id":     {Name: "corporation-id", DefaultValue: "0", Usage: "Filter by controlling corporation id"},
						"modified_after":     {Name: "modified-after", DefaultValue: "", Usage: "Filter by modified time (RFC3339)"},
						"active_gf_only":     {Name: "active-gf-only", DefaultValue: "false", Usage: "Include only the active GF version"},
						"preferred_language": {Name: "preferred-language", DefaultValue: "", Usage: "Preferred document language"},
						"response_max_size":  {Name: "response-max-size", DefaultValue: "64", Usage: "Max results (1-1024)"},
					},
				},
				{RpcMethod: "Params", Use: "params", Short: "Get the current module parameters"},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "CreateEcosystem",
					Use:       "create-ecosystem [corporation] [did] [language] [doc-url] [doc-digest-sri]",
					Short:     "Create a new ecosystem on behalf of a corporation",
					Long:      "Create a new Ecosystem. The transaction is signed by `operator` on behalf of `corporation` (the policy_address of the controlling Corporation). The doc_url and doc_digest_sri seed v1 of the Ecosystem's Governance Framework.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
						{ProtoField: "did"},
						{ProtoField: "language"},
						{ProtoField: "doc_url"},
						{ProtoField: "doc_digest_sri"},
					},
				},
				{
					RpcMethod: "UpdateEcosystem",
					Use:       "update-ecosystem [corporation] [id] [did]",
					Short:     "Rotate an ecosystem's DID",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
						{ProtoField: "id"},
						{ProtoField: "did"},
					},
				},
				{
					RpcMethod: "ArchiveEcosystem",
					Use:       "archive-ecosystem [corporation] [id] [archive]",
					Short:     "Archive (true) or unarchive (false) an ecosystem",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
						{ProtoField: "id"},
						{ProtoField: "archive"},
					},
				},
			},
		},
	}
}
