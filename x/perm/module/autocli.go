package permission

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/verana-labs/verana-blockchain/api/verana/perm/v1"
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
					RpcMethod: "ListPermissions",
					Use:       "list-permissions",
					Short:     "List all permissions",
					Long:      "List all permissions with optional filtering by modified time and pagination",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"modified_after": {
							Name:         "modified-after",
							Usage:        "Filter by modified time (RFC3339 format)",
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
					RpcMethod: "GetPermission",
					Use:       "get-perm [id]",
					Short:     "Get perm by ID",
					Long:      "Get detailed information about a perm by its ID",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
				},
				{
					RpcMethod: "GetPermissionSession",
					Use:       "get-perm-session [id]",
					Short:     "Get perm session by ID",
					Long:      "Get details about a specific perm session by its ID",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
				},
				{
					RpcMethod: "ListPermissionSessions",
					Use:       "list-perm-sessions",
					Short:     "List perm sessions",
					Long:      "List all perm sessions with optional filtering and pagination",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"modified_after": {
							Name:         "modified-after",
							Usage:        "Filter by modified time (RFC3339 format)",
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
					RpcMethod: "FindPermissionsWithDID",
					Use:       "find-permissions-with-did [did] [type] [schema-id]",
					Short:     "Find permissions with DID",
					Long:      "Find permissions matching the specified DID, type, and schema ID with optional filtering by country and timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "did"},
						{ProtoField: "type"},
						{ProtoField: "schema_id"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"country": {
							Name:         "country",
							DefaultValue: "",
							Usage:        "Filter by country code (ISO 3166-1 alpha-2)",
						},
						"when": {
							Name:         "when",
							DefaultValue: "",
							Usage:        "Filter by validity at specified timestamp (RFC3339 format)",
						},
					},
				},
				{
					RpcMethod: "FindBeneficiaries",
					Use:       "find-beneficiaries",
					Short:     "Find beneficiary permissions in the perm tree",
					Long:      "Find beneficiary permissions by traversing the perm tree for issuer and/or verifier permissions",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"issuer_perm_id": {
							Name:         "issuer-perm-id",
							DefaultValue: "0",
							Usage:        "ID of the issuer perm",
						},
						"verifier_perm_id": {
							Name:         "verifier-perm-id",
							DefaultValue: "0",
							Usage:        "ID of the verifier perm",
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
					RpcMethod: "StartPermissionVP",
					Use:       "start-perm-vp [type] [validator-perm-id] [country]",
					Short:     "Start a new perm validation process",
					Long: `Start a new perm validation process with the specified parameters:
- type: Permission type (0=Unspecified, 1=Issuer, 2=Verifier, 3=IssuerGrantor, 4=VerifierGrantor, 5=TrustRegistry, 6=Holder)
- validator-perm-id: ID of the validator perm
- country: ISO 3166-1 alpha-2 country code`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "type",
						},
						{
							ProtoField: "validator_perm_id",
						},
						{
							ProtoField: "country",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"did": {
							Name:         "did",
							Usage:        "Optional DID for this perm",
							DefaultValue: "",
						},
					},
				},
				{
					RpcMethod: "RenewPermissionVP",
					Use:       "renew-perm-vp [id]",
					Short:     "Renew a perm validation process",
					Long: `Renew a perm validation process for an existing perm:
- id: ID of the perm to renew`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
				},
				{
					RpcMethod: "SetPermissionVPToValidated",
					Use:       "set-perm-vp-validated [id]",
					Short:     "Set perm validation process to validated state",
					Long: `Set a perm validation process to validated state with optional parameters:
- id: ID of the perm to validate
- effective-until: Optional timestamp until when this perm is effective (RFC3339 format)
- validation-fees: Optional validation fees
- issuance-fees: Optional issuance fees
- verification-fees: Optional verification fees
- country: Optional country code (ISO 3166-1 alpha-2)
- vp-summary-digest-sri: Optional digest SRI of validation information`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"effective_until": {
							Name:         "effective-until",
							Usage:        "Timestamp until when this perm is effective (RFC3339)",
							DefaultValue: "",
						},
						"validation_fees": {
							Name:         "validation-fees",
							Usage:        "Validation fees",
							DefaultValue: "0",
						},
						"issuance_fees": {
							Name:         "issuance-fees",
							Usage:        "Issuance fees",
							DefaultValue: "0",
						},
						"verification_fees": {
							Name:         "verification-fees",
							Usage:        "Verification fees",
							DefaultValue: "0",
						},
						"country": {
							Name:         "country",
							Usage:        "Country code (ISO 3166-1 alpha-2)",
							DefaultValue: "",
						},
						"vp_summary_digest_sri": {
							Name:         "vp-summary-digest-sri",
							Usage:        "Digest SRI of validation information",
							DefaultValue: "",
						},
					},
				},
				{
					RpcMethod: "RequestPermissionVPTermination",
					Use:       "request-vp-termination [id]",
					Short:     "Request termination of a perm validation process",
					Long: `Request termination of a perm validation process:
- id: ID of the perm validation process to terminate
Note: For expired VPs, either the grantee or validator can request termination.
For active VPs, only the grantee can request termination unless it's a HOLDER type.`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
				},
				// Add to the RpcCommandOptions array in the Tx ServiceCommandDescriptor:
				{
					RpcMethod: "ConfirmPermissionVPTermination",
					Use:       "confirm-vp-termination [id]",
					Short:     "Confirm the termination of a perm VP",
					Long:      "Confirm the termination of a perm VP. Can be called by the validator, or by the grantee after the timeout period has elapsed.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
				},
				{
					RpcMethod: "CancelPermissionVPLastRequest",
					Use:       "cancel-perm-vp-request [id]",
					Short:     "Cancel a pending perm VP request",
					Long:      "Cancel a pending perm VP request. Can only be executed by the perm grantee and only when the perm is in PENDING state.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
				},
				{
					RpcMethod: "CreateRootPermission",
					Use:       "create-root-perm [schema-id] [did] [validation-fees] [issuance-fees] [verification-fees]",
					Short:     "Create a new root perm for a credential schema",
					Long:      "Create a new root perm for a credential schema. Can only be executed by the trust registry controller.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "schema_id",
						},
						{
							ProtoField: "did",
						},
						{
							ProtoField: "validation_fees",
						},
						{
							ProtoField: "issuance_fees",
						},
						{
							ProtoField: "verification_fees",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"country": {
							Name:         "country",
							DefaultValue: "",
							Usage:        "Optional country code (ISO 3166-1 alpha-2)",
						},
						"effective_from": {
							Name:         "effective-from",
							DefaultValue: "",
							Usage:        "Optional timestamp (RFC3339) from when the perm is effective",
						},
						"effective_until": {
							Name:         "effective-until",
							DefaultValue: "",
							Usage:        "Optional timestamp (RFC3339) until when the perm is effective",
						},
					},
				},
				{
					RpcMethod: "ExtendPermission",
					Use:       "extend-perm [id] [effective-until]",
					Short:     "Extend a perm's effective duration",
					Long:      "Extend a perm's effective duration. Can only be executed by the validator of the perm.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
						{
							ProtoField: "effective_until",
						},
					},
				},
				{
					RpcMethod: "RevokePermission",
					Use:       "revoke-perm [id]",
					Short:     "Revoke a perm",
					Long:      "Revoke a perm. Can only be executed by the validator of the perm.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
				},
				{
					RpcMethod: "CreateOrUpdatePermissionSession",
					Use:       "create-or-update-perm-session [id] [agent-perm-id]",
					Short:     "Create or update a perm session",
					Long: `Create or update a perm session with the specified parameters:
- id: UUID of the session
- agent-perm-id: ID of the agent perm (HOLDER)
Optional parameters:
- issuer-perm-id: ID of the issuer perm
- verifier-perm-id: ID of the verifier perm
- wallet-agent-perm-id: ID of the wallet agent perm if different from agent

At least one of issuer-perm-id or verifier-perm-id must be provided.`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
						{
							ProtoField: "agent_perm_id",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"issuer_perm_id": {
							Name:         "issuer-perm-id",
							Usage:        "ID of the issuer perm",
							DefaultValue: "0",
						},
						"verifier_perm_id": {
							Name:         "verifier-perm-id",
							Usage:        "ID of the verifier perm",
							DefaultValue: "0",
						},
						"wallet_agent_perm_id": {
							Name:         "wallet-agent-perm-id",
							Usage:        "ID of the wallet agent perm if different from agent",
							DefaultValue: "0",
						},
					},
				},
				{
					RpcMethod: "SlashPermissionTrustDeposit",
					Use:       "slash-perm-td [id] [amount]",
					Short:     "Slash a perm's trust deposit",
					Long: `Slash a perm's trust deposit. Can only be executed by:
- The validator that created the perm
- The grantee of the ECOSYSTEM perm (trust registry controller) for the corresponding credential schema
- The network governance authority (via proposal)

Parameters:
- id: ID of the perm to slash
- amount: Amount to slash from the trust deposit`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
						{
							ProtoField: "amount",
						},
					},
				},
				{
					RpcMethod: "RepayPermissionSlashedTrustDeposit",
					Use:       "repay-perm-slashed-td [id]",
					Short:     "Repay a slashed perm's trust deposit",
					Long: `Repay the slashed trust deposit of a perm. Can be executed by anyone willing to pay.
This will repay the full remaining slashed amount and credit it to the perm grantee's trust deposit.
Note: This does not make the slashed perm reusable - a new perm must be requested.

Parameters:
- id: ID of the perm with slashed deposit to repay`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
				},
				{
					RpcMethod: "CreatePermission",
					Use:       "create-perm [schema-id] [type] [did]",
					Short:     "Create a new perm for open schemas",
					Long: `Create a new ISSUER or VERIFIER perm for schemas with OPEN management mode.
This allows self-creation of permissions without validation process.

Parameters:
- schema-id: ID of the credential schema
- type: Permission type (1=ISSUER, 2=VERIFIER)
- did: DID of the grantee service

Optional flags:
- country: ISO 3166-1 alpha-2 country code
- effective-from: Timestamp when perm becomes effective (RFC3339)
- effective-until: Timestamp when perm expires (RFC3339)
- verification-fees: Fees for credential verification (default: 0)`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "schema_id",
						},
						{
							ProtoField: "type",
						},
						{
							ProtoField: "did",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"country": {
							Name:         "country",
							DefaultValue: "",
							Usage:        "Optional country code (ISO 3166-1 alpha-2)",
						},
						"effective_from": {
							Name:         "effective-from",
							DefaultValue: "",
							Usage:        "Optional timestamp (RFC3339) from when the perm is effective",
						},
						"effective_until": {
							Name:         "effective-until",
							DefaultValue: "",
							Usage:        "Optional timestamp (RFC3339) until when the perm is effective",
						},
						"verification_fees": {
							Name:         "verification-fees",
							DefaultValue: "0",
							Usage:        "Verification fees in trust units",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
