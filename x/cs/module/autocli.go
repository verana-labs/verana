package credentialschema

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/verana-labs/verana/api/verana/cs/v1"
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
					RpcMethod: "ListCredentialSchemas",
					Use:       "list-schemas",
					Short:     "List credential schemas with optional filters",
					Long: `List credential schemas with optional filters.
Example:
$ veranad query cs list-schemas
$ veranad query cs list-schemas --tr_id 1 --modified_after 2024-01-01T00:00:00Z --response_max_size 100`,
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"tr_id": {
							Name:         "tr_id",
							Usage:        "Filter by trust registry ID",
							DefaultValue: "0",
						},
						"modified_after": {
							Name:         "modified_after",
							Usage:        "Show schemas modified after this datetime (RFC3339 format)",
							DefaultValue: "",
						},
						"response_max_size": {
							Name:         "response_max_size",
							Usage:        "Maximum number of results (1-1024, default 64)",
							DefaultValue: "64",
						},
					},
				},
				{
					RpcMethod: "GetCredentialSchema",
					Use:       "get-schema [id]",
					Short:     "Get a credential schema by ID",
					Long: `Get a credential schema by its ID.

Example:
$ veranad query cs get 1`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
					},
				},
				{
					RpcMethod: "RenderJsonSchema",
					Use:       "render-json-schema [id]",
					Short:     "Get the JSON schema definition",
					Long: `Render the JSON schema definition for a credential schema.
Response will be in application/schema+json format.

Example:
$ veranad query cs schema 1`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
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
					RpcMethod: "CreateCredentialSchema",
					Use:       "create-credential-schema [tr-id] [json-schema] [issuer-mode] [verifier-mode]",
					Short:     "Create a new credential schema",
					Long: `Create a new credential schema with the specified parameters. The JSON schema supports placeholder replacement:
- VPR_CREDENTIAL_SCHEMA_ID: Replaced with the generated schema ID  
- VPR_CHAIN_ID: Replaced with the current chain ID

Required Parameters:
- tr-id: Trust registry ID (must be controlled by creator)
- json-schema: Path to JSON schema file or inline JSON string
- issuer-mode: Permission management mode (1=OPEN, 2=GRANTOR_VALIDATION, 3=TRUST_REGISTRY_VALIDATION)
- verifier-mode: Permission management mode (same options as issuer-mode)

Optional Flags (default to 0 days):
- --issuer-grantor-validation-validity-period: Validation period for issuer grantors (days, default: 0)
- --verifier-grantor-validation-validity-period: Validation period for verifier grantors (days, default: 0)
- --issuer-validation-validity-period: Validation period for issuers (days, default: 0)
- --verifier-validation-validity-period: Validation period for verifiers (days, default: 0)
- --holder-validation-validity-period: Validation period for holders (days, default: 0)

Example:
$ veranad tx cs create-credential-schema 1 schema.json 2 2 --issuer-grantor-validation-validity-period 365 --verifier-grantor-validation-validity-period 365
$ veranad tx cs create-credential-schema 1 schema.json 2 2`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "tr_id",
						},
						{
							ProtoField: "json_schema",
						},
						{
							ProtoField: "issuer_perm_management_mode",
						},
						{
							ProtoField: "verifier_perm_management_mode",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"issuer_grantor_validation_validity_period": {
							Name:         "issuer-grantor-validation-validity-period",
							Usage:        "Validation period for issuer grantors in days (default: 0)",
							DefaultValue: "0",
						},
						"verifier_grantor_validation_validity_period": {
							Name:         "verifier-grantor-validation-validity-period",
							Usage:        "Validation period for verifier grantors in days (default: 0)",
							DefaultValue: "0",
						},
						"issuer_validation_validity_period": {
							Name:         "issuer-validation-validity-period",
							Usage:        "Validation period for issuers in days (default: 0)",
							DefaultValue: "0",
						},
						"verifier_validation_validity_period": {
							Name:         "verifier-validation-validity-period",
							Usage:        "Validation period for verifiers in days (default: 0)",
							DefaultValue: "0",
						},
						"holder_validation_validity_period": {
							Name:         "holder-validation-validity-period",
							Usage:        "Validation period for holders in days (default: 0)",
							DefaultValue: "0",
						},
					},
				},
				{
					RpcMethod: "UpdateCredentialSchema",
					Use:       "update [id]",
					Short:     "Update a credential schema's validity periods",
					Long: `Update the validity periods of an existing credential schema.

Optional Flags (default to 0 days):
- --issuer-grantor-validation-validity-period: Validation period for issuer grantors (days, default: 0)
- --verifier-grantor-validation-validity-period: Validation period for verifier grantors (days, default: 0)
- --issuer-validation-validity-period: Validation period for issuers (days, default: 0)
- --verifier-validation-validity-period: Validation period for verifiers (days, default: 0)
- --holder-validation-validity-period: Validation period for holders (days, default: 0)

Example:
$ veranad tx cs update 1 --issuer-validation-validity-period 365 --verifier-validation-validity-period 180`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"issuer_grantor_validation_validity_period": {
							Name:         "issuer-grantor-validation-validity-period",
							Usage:        "Validation period for issuer grantors in days (default: 0)",
							DefaultValue: "0",
						},
						"verifier_grantor_validation_validity_period": {
							Name:         "verifier-grantor-validation-validity-period",
							Usage:        "Validation period for verifier grantors in days (default: 0)",
							DefaultValue: "0",
						},
						"issuer_validation_validity_period": {
							Name:         "issuer-validation-validity-period",
							Usage:        "Validation period for issuers in days (default: 0)",
							DefaultValue: "0",
						},
						"verifier_validation_validity_period": {
							Name:         "verifier-validation-validity-period",
							Usage:        "Validation period for verifiers in days (default: 0)",
							DefaultValue: "0",
						},
						"holder_validation_validity_period": {
							Name:         "holder-validation-validity-period",
							Usage:        "Validation period for holders in days (default: 0)",
							DefaultValue: "0",
						},
					},
				},
				{
					RpcMethod: "ArchiveCredentialSchema",
					Use:       "archive [id] [archive]",
					Short:     "Archive or unarchive a credential schema",
					Long:      "Set the archive status of a credential schema. Use true to archive, false to unarchive",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
						{
							ProtoField: "archive",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
