package participant

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/verana-labs/verana/api/verana/pp/v1"
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
					RpcMethod: "ListParticipants",
					Use:       "list-participants",
					Short:     "List all participants",
					Long:      "List all participants with optional filtering by modified time and pagination",
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
					RpcMethod: "GetParticipant",
					Use:       "get-participant [id]",
					Short:     "Get participant by ID",
					Long:      "Get detailed information about a participant by its ID",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
				},
				{
					RpcMethod: "GetParticipantSession",
					Use:       "get-participant-session [id]",
					Short:     "Get participant session by ID",
					Long:      "Get details about a specific participant session by its ID",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
				},
				{
					RpcMethod: "ListParticipantSessions",
					Use:       "list-participant-sessions",
					Short:     "List participant sessions",
					Long:      "List all participant sessions with optional filtering and pagination",
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
					RpcMethod: "FindParticipantsWithDID",
					Use:       "find-participants-with-did [did] [role] [schema-id]",
					Short:     "Find participants with DID",
					Long:      "Find participants matching the specified DID, type, and schema ID with optional filtering by timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "did"},
						{ProtoField: "role"},
						{ProtoField: "schema_id"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
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
					Short:     "Find beneficiary participants in the participant tree",
					Long:      "Find beneficiary participants by traversing the participant tree for issuer and/or verifier participants. At least one of issuer-participant-id or verifier-participant-id must be provided.",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"issuer_participant_id": {
							Name:         "issuer-participant-id",
							DefaultValue: "0",
							Usage:        "ID of the issuer participant",
						},
						"verifier_participant_id": {
							Name:         "verifier-participant-id",
							DefaultValue: "0",
							Usage:        "ID of the verifier participant",
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
					RpcMethod: "StartParticipantOP",
					Use:       "start-participant-op [role] [validator-participant-id] [did]",
					Short:     "Start a new participant validation process",
					Long: `Start a new participant validation process with the specified parameters:
- type: Participant type (issuer, verifier, issuer-grantor, verifier-grantor, ecosystem, holder)
- validator-participant-id: ID of the validator participant
- did: DID for this participant (mandatory, must conform to DID syntax)`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "role",
						},
						{
							ProtoField: "validator_participant_id",
						},
						{
							ProtoField: "did",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							Name:  "corporation",
							Usage: "Group account (corporation) on whose behalf this message is executed",
						},
						"validation_fees": {
							Name:         "validation-fees",
							Usage:        "Optional requested validation fees (can be modified by validator)",
							DefaultValue: "0",
						},
						"issuance_fees": {
							Name:         "issuance-fees",
							Usage:        "Optional requested issuance fees (can be modified by validator)",
							DefaultValue: "0",
						},
						"verification_fees": {
							Name:         "verification-fees",
							Usage:        "Optional requested verification fees (can be modified by validator)",
							DefaultValue: "0",
						},
						"vs_operator": {
							Name:         "vs-operator",
							Usage:        "Optional Verifiable Service operator account address",
							DefaultValue: "",
						},
						"vs_operator_authz_msg_types": {
							Name:  "vs-operator-authz-msg-types",
							Usage: "Delegable msg types granted to vs_operator (presence triggers a VSOA record)",
						},
						"vs_operator_authz_with_feegrant": {
							Name:         "vs-operator-authz-with-feegrant",
							Usage:        "Enable fee grant for vs_operator",
							DefaultValue: "false",
						},
					},
				},
				{
					RpcMethod: "RenewParticipantOP",
					Use:       "renew-participant-op [id]",
					Short:     "Renew a participant validation process",
					Long: `Renew a participant validation process for an existing participant:
- id: ID of the participant to renew`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							Name:  "corporation",
							Usage: "Group account (corporation) on whose behalf this message is executed",
						},
					},
				},
				{
					RpcMethod: "SetParticipantOPToValidated",
					Use:       "set-participant-op-validated [id]",
					Short:     "Set participant validation process to validated state",
					Long: `Set a participant validation process to validated state.

Requires authority/operator authorization. The authority must be the validator participant's authority.

Parameters:
- id: ID of the participant to validate
- authority: Group account (corporation) on whose behalf this message is executed
- effective-until: Optional timestamp until when this participant is effective (RFC3339 format)
- validation-fees: Validation fees (mandatory, 0 for no fees)
- issuance-fees: Issuance fees (mandatory, 0 for no fees)
- verification-fees: Verification fees (mandatory, 0 for no fees)
- op-summary-digest-sri: Optional digest SRI of validation information
- issuance-fee-discount: Issuance fee discount (0-10000, where 10000 = 100% discount, default 0)
- verification-fee-discount: Verification fee discount (0-10000, where 10000 = 100% discount, default 0)`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							Name:  "corporation",
							Usage: "Group account (corporation) on whose behalf this message is executed",
						},
						"effective_until": {
							Name:         "effective-until",
							Usage:        "Timestamp until when this participant is effective (RFC3339)",
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
						"op_summary_digest": {
							Name:         "op-summary-digest",
							Usage:        "Digest SRI of validation information",
							DefaultValue: "",
						},
						"issuance_fee_discount": {
							Name:         "issuance-fee-discount",
							Usage:        "Issuance fee discount (0-10000, where 10000 = 100% discount)",
							DefaultValue: "0",
						},
						"verification_fee_discount": {
							Name:         "verification-fee-discount",
							Usage:        "Verification fee discount (0-10000, where 10000 = 100% discount)",
							DefaultValue: "0",
						},
					},
				},

				{
					RpcMethod: "CancelParticipantOPLastRequest",
					Use:       "cancel-participant-op-request [id]",
					Short:     "Cancel a pending participant VP request",
					Long:      "Cancel a pending participant VP request. Can only be executed by the participant authority and only when the participant is in PENDING state.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							Name:  "corporation",
							Usage: "Group account (corporation) on whose behalf this message is executed",
						},
					},
				},
				{
					RpcMethod: "CreateRootParticipant",
					Use:       "create-root-participant [schema-id] [did] [validation-fees] [issuance-fees] [verification-fees]",
					Short:     "Create a new root participant for a credential schema",
					Long:      "Create a new root participant for a credential schema. Can only be executed by the trust registry controller.",
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
						"corporation": {
							Name:  "corporation",
							Usage: "Group account (corporation) on whose behalf this message is executed",
						},
						"effective_from": {
							Name:         "effective-from",
							DefaultValue: "",
							Usage:        "Timestamp (RFC3339) from when the participant is effective (mandatory, must be in the future)",
						},
						"effective_until": {
							Name:         "effective-until",
							DefaultValue: "",
							Usage:        "Optional timestamp (RFC3339) until when the participant is effective",
						},
					},
				},
				{
					RpcMethod: "SetParticipantEffectiveUntil",
					Use:       "set-participant-effective-until [id] [effective-until]",
					Short:     "Adjust a participant's effective duration",
					Long:      "Adjust a participant's effective duration. Can be executed by the authority (for ECOSYSTEM or self-created participants) or by the validator (for VP managed participants).",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
						{
							ProtoField: "effective_until",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							DefaultValue: "",
							Usage:        "The group policy address (corporation) on whose behalf this message is executed",
						},
					},
				},
				{
					RpcMethod: "RevokeParticipant",
					Use:       "revoke-participant [id]",
					Short:     "Revoke a participant",
					Long:      "Revoke a participant. Can be executed by the participant authority, a validator ancestor, or the trust registry controller.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							DefaultValue: "",
							Usage:        "The group policy address (corporation) on whose behalf this message is executed",
						},
					},
				},
				{
					RpcMethod: "CreateOrUpdateParticipantSession",
					Use:       "create-or-update-participant-session [id] [agent-participant-id] [wallet-agent-participant-id]",
					Short:     "Create or update a participant session",
					Long:      "Create or update a participant session for credential exchange operations. At least one of issuer-participant-id or verifier-participant-id must be provided.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
						{
							ProtoField: "agent_participant_id",
						},
						{
							ProtoField: "wallet_agent_participant_id",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							DefaultValue: "",
							Usage:        "The group policy address (corporation) on whose behalf this message is executed",
						},
						"issuer_participant_id": {
							Name:         "issuer-participant-id",
							Usage:        "ID of the issuer participant",
							DefaultValue: "0",
						},
						"verifier_participant_id": {
							Name:         "verifier-participant-id",
							Usage:        "ID of the verifier participant",
							DefaultValue: "0",
						},
						"digest": {
							Name:         "digest",
							Usage:        "Optional digest derived from an issued or verified credential",
							DefaultValue: "",
						},
					},
				},
				{
					RpcMethod: "SlashParticipantTrustDeposit",
					Use:       "slash-participant-td [id] [amount] [reason]",
					Short:     "Slash a participant's trust deposit",
					Long:      "Slash a participant's trust deposit. Requires a non-empty reason per [MOD-PP-MSG-12-1].",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
						{
							ProtoField: "amount",
						},
						{
							ProtoField: "reason",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							DefaultValue: "",
							Usage:        "The group policy address (corporation) on whose behalf this message is executed",
						},
					},
				},
				{
					RpcMethod: "RepayParticipantSlashedTrustDeposit",
					Use:       "repay-participant-slashed-td [id] [amount] --corporation [corporation]",
					Short:     "Repay a slashed participant's trust deposit",
					Long: `Repay a portion or all of the slashed trust deposit of a participant. Can only be called by the authority that owns the participant.
The specified amount will be credited to the participant authority's trust deposit.
Note: This does not make the slashed participant reusable - a new participant must be requested.

Parameters:
- id: ID of the participant with slashed deposit to repay
- amount: Amount to repay (must be positive and not exceed outstanding slashed amount)
- authority: The group policy address (corporation) that owns the participant`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "id",
						},
						{
							ProtoField: "amount",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							DefaultValue: "",
							Usage:        "The group policy address (corporation) on whose behalf this message is executed",
						},
					},
				},
				{
					RpcMethod: "SelfCreateParticipant",
					Use:       "self-create-participant [role] [validator-participant-id] [did] --corporation [corporation]",
					Short:     "Self-create a new participant for open schemas",
					Long:      "Self-create a new ISSUER or VERIFIER participant for schemas with OPEN management mode.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "role",
						},
						{
							ProtoField: "validator_participant_id",
						},
						{
							ProtoField: "did",
						},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"corporation": {
							DefaultValue: "",
							Usage:        "The group policy address (corporation) on whose behalf this message is executed",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
