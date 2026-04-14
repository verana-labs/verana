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
						"corporation": {
							Name:         "corporation",
							Usage:        "Filter by corporation account address",
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
					Use:       "create-trust-registry [corporation] [did] [language]",
					Short:     "Create a new trust registry",
					Long:      "Create a new trust registry on behalf of a corporation (group account). The operator (transaction signer) must be authorized by the corporation.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
						{ProtoField: "did"},
						{ProtoField: "language"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"aka": {
							Name:         "aka",
							DefaultValue: "",
							Usage:        "optional additional URI of this trust registry",
						},
					},
				},
				{
					RpcMethod: "AddGovernanceFrameworkDocument",
					Use:       "add-governance-framework-document [corporation] [tr-id] [language] [url] [digest-sri] [version]",
					Short:     "Add a governance framework document",
					Long:      "Add a new governance framework document to a trust registry. The operator (transaction signer) must be authorized by the corporation.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
						{ProtoField: "tr_id"},
						{ProtoField: "language"},
						{ProtoField: "url"},
						{ProtoField: "digest_sri"},
						{ProtoField: "version"},
					},
				},
				{
					RpcMethod: "IncreaseActiveGovernanceFrameworkVersion",
					Use:       "increase-active-gf-version [corporation] [tr-id]",
					Short:     "Increase the active governance framework version",
					Long:      "Increase the active governance framework version for a trust registry. The operator (transaction signer) must be authorized by the corporation. Requires a document in the trust registry's default language for the new version.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
						{ProtoField: "tr_id"},
					},
				},
				{
					RpcMethod: "UpdateTrustRegistry",
					Use:       "update-trust-registry [corporation] [tr-id]",
					Short:     "Update a trust registry",
					Long:      "Update a trust registry's AKA URI and language. The operator (transaction signer) must be authorized by the corporation.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
						{ProtoField: "tr_id"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"aka": {
							Name:         "aka",
							DefaultValue: "",
							Usage:        "aka uri",
						},
						"language": {
							Name:         "language",
							DefaultValue: "",
							Usage:        "primary language tag (RFC1766)",
						},
					},
				},
				{
					RpcMethod: "ArchiveTrustRegistry",
					Use:       "archive-trust-registry [corporation] [tr-id] [archive]",
					Short:     "Archive or unarchive a trust registry",
					Long:      "Set the archive status of a trust registry. The operator (transaction signer) must be authorized by the corporation. Use true to archive, false to unarchive.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
						{ProtoField: "tr_id"},
						{ProtoField: "archive"},
					},
				},
			},
		},
	}
}
