package trustregistry

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	modulev1 "github.com/verana-labs/verana/api/verana/tr/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "GetTrustRegistry",
					Use:       "get-trust-registry [tr_id]",
					Short:     "Get trust registry information by ID",
					Long:      "Get the trust registry information for a given trust registry ID, with options to filter by active governance framework and preferred language",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "tr_id"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"active_gf_only": {
							Name:         "active-gf-only",
							DefaultValue: "false",
							Usage:        "If true, include only current governance framework data",
						},
						"preferred_language": {
							Name:         "preferred-language",
							DefaultValue: "",
							Usage:        "Preferred language for the returned documents",
						},
					},
				},
				{
					RpcMethod: "ListTrustRegistries",
					Use:       "list-trust-registries",
					Short:     "List Trust Registries",
					Long:      "List Trust Registries with optional filtering and pagination. Results are ordered by modified time ascending.",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"controller": {
							Name:         "controller",
							Usage:        "Filter by controller account address",
							DefaultValue: "",
						},
						"modified_after": {
							Name:         "modified-after",
							Usage:        "Filter by modified time (RFC3339 format)",
							DefaultValue: "",
						},
						"active_gf_only": {
							Name:         "active-gf-only",
							Usage:        "Include only current governance framework data",
							DefaultValue: "false",
						},
						"preferred_language": {
							Name:         "preferred-language",
							Usage:        "Preferred language for returned documents",
							DefaultValue: "",
						},
						"response_max_size": {
							Name:         "response-max-size",
							Usage:        "Maximum number of results to return (1-1024)",
							DefaultValue: "64",
						},
					},
				},
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Get the current module parameters",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "CreateTrustRegistry",
					Use:       "create-trust-registry [did] [language] [doc-url] [doc-digest-sri]",
					Short:     "Create a new trust registry",
					Long:      "Create a new trust registry with the specified DID, language, and initial governance framework document",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "did"},
						{ProtoField: "language"},
						{ProtoField: "doc_url"},
						{ProtoField: "doc_digest_sri"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"aka": {
							Name:         "aka",
							DefaultValue: "",
							Usage:        "aka uri",
						},
					},
				},
				{
					RpcMethod: "AddGovernanceFrameworkDocument",
					Use:       "add-governance-framework-document [id] [doc-language] [url] [doc-digest-sri] [version]",
					Short:     "Add a governance framework document",
					Long:      "Add a new governance framework document to a trust registry",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
						{ProtoField: "doc_language"},
						{ProtoField: "doc_url"},
						{ProtoField: "doc_digest_sri"},
						{ProtoField: "version"},
					},
				},
				{
					RpcMethod: "IncreaseActiveGovernanceFrameworkVersion",
					Use:       "increase-active-gf-version [id]",
					Short:     "Increase the active governance framework version",
					Long:      "Increase the active governance framework version for a trust registry. This can only be done by the controller of the trust registry and requires a document in the trust registry's default language for the new version.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
					},
				},
				{
					RpcMethod: "UpdateTrustRegistry",
					Use:       "update-trust-registry [id] [did]",
					Short:     "Update a trust registry",
					Long:      "Update a trust registry's DID and AKA URI. Only the controller can update a trust registry.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
						{ProtoField: "did"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"aka": {
							Name:         "aka",
							DefaultValue: "",
							Usage:        "aka uri",
						},
					},
				},
				{
					RpcMethod: "ArchiveTrustRegistry",
					Use:       "archive-trust-registry [id] [archive]",
					Short:     "Archive or unarchive a trust registry",
					Long:      "Set the archive status of a trust registry. Use true to archive, false to unarchive. Only the controller can archive/unarchive.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
						{ProtoField: "archive"},
					},
				},
			},
		},
	}
}
