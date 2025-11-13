package types

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// JsonSchemaMetaSchema Official meta-schema for Draft 2020-12
const JsonSchemaMetaSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://example.com/meta-schema/credential-schema",
  "title": "Credential Schema Meta-Schema",
  "type": "object",
  "required": ["$id", "$schema", "type", "title", "description", "properties"],
  "properties": {
    "$id": {
      "type": "string",
      "format": "uri-reference",
      "pattern": "^vpr:verana:VPR_CHAIN_ID/cs/v1/js/VPR_CREDENTIAL_SCHEMA_ID$",
      "description": "$id must be a URI matching the rendering URL format"
    },
    "$schema": {
      "type": "string",
      "enum": ["https://json-schema.org/draft/2020-12/schema"],
      "description": "$schema must be the Draft 2020-12 URI"
    },
    "type": {
      "type": "string",
      "enum": ["object"],
      "description": "The root type must be 'object'"
    },
    "title": {
      "type": "string",
      "description": "The title of the credential schema"
    },
    "description": {
      "type": "string",
      "description": "The description of the credential schema"
    },
    "properties": {
      "type": "object",
      "additionalProperties": {
        "type": "object",
        "properties": {
          "type": {
            "type": "string",
            "enum": ["string", "number", "integer", "boolean", "object", "array"],
            "description": "The type of each property"
          },
          "description": {
            "type": "string"
          },
          "default": {
            "type": ["string", "number", "integer", "boolean", "object", "array", "null"]
          }
        },
        "required": ["type"]
      }
    },
    "required": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "List of required properties"
    },
    "additionalProperties": {
      "type": "boolean",
      "default": true
    },
    "$defs": {
      "type": "object",
      "additionalProperties": {
        "type": "object"
      },
      "description": "Optional definitions for reusable schema components"
    }
  },
  "additionalProperties": false,
  "examples": [
    {
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "$id": "vpr:verana:mainnet/cs/v1/js/1",
      "title": "ExampleCredential",
      "description": "ExampleCredential using JsonSchema",
      "type": "object",
      "properties": {
        "name": {
          "type": "string",
          "description": "Name of the entity"
        }
      },
      "required": ["name"],
      "additionalProperties": false
    }
  ]
}
`
const TypeMsgCreateCredentialSchema = "create_credential_schema"

var _ sdk.Msg = &MsgCreateCredentialSchema{}

// NewMsgCreateCredentialSchema creates a new MsgCreateCredentialSchema instance
func NewMsgCreateCredentialSchema(
	creator string,
	trId uint64,
	jsonSchema string,
	issuerGrantorValidationValidityPeriod uint32,
	verifierGrantorValidationValidityPeriod uint32,
	issuerValidationValidityPeriod uint32,
	verifierValidationValidityPeriod uint32,
	holderValidationValidityPeriod uint32,
	issuerPermManagementMode uint32,
	verifierPermManagementMode uint32,
) *MsgCreateCredentialSchema {
	msg := &MsgCreateCredentialSchema{
		Creator:                                 creator,
		TrId:                                    trId,
		JsonSchema:                              jsonSchema,
		IssuerGrantorValidationValidityPeriod:   &OptionalUInt32{Value: issuerGrantorValidationValidityPeriod},
		VerifierGrantorValidationValidityPeriod: &OptionalUInt32{Value: verifierGrantorValidationValidityPeriod},
		IssuerValidationValidityPeriod:          &OptionalUInt32{Value: issuerValidationValidityPeriod},
		VerifierValidationValidityPeriod:        &OptionalUInt32{Value: verifierValidationValidityPeriod},
		HolderValidationValidityPeriod:          &OptionalUInt32{Value: holderValidationValidityPeriod},
		IssuerPermManagementMode:                issuerPermManagementMode,
		VerifierPermManagementMode:              verifierPermManagementMode,
	}

	return msg
}

// Route implements sdk.Msg
func (msg *MsgCreateCredentialSchema) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg *MsgCreateCredentialSchema) Type() string {
	return TypeMsgCreateCredentialSchema
}

// GetSigners implements sdk.Msg
func (msg *MsgCreateCredentialSchema) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

// ValidateBasic implements sdk.Msg
func (msg *MsgCreateCredentialSchema) ValidateBasic() error {
	// Validate creator address
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	// Check mandatory parameters
	if msg.TrId == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "tr_id cannot be 0")
	}

	// Validate JSON Schema (without ID since it will be generated later)
	if err := validateJSONSchema(msg.JsonSchema); err != nil {
		return errors.Wrapf(ErrInvalidJSONSchema, err.Error())
	}

	// Validate validity periods (must be >= 0)
	if err := validateValidityPeriods(msg); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	// Validate perm management modes
	if err := validatePermManagementModes(msg); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	return nil
}

func validateJSONSchema(schemaJSON string) error {
	if schemaJSON == "" {
		return fmt.Errorf("json schema cannot be empty")
	}

	if len(schemaJSON) > int(DefaultCredentialSchemaSchemaMaxSize) {
		return fmt.Errorf("json schema exceeds maximum size of %d bytes", DefaultCredentialSchemaSchemaMaxSize)
	}

	// Parse JSON
	var schemaDoc map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaDoc); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Ignore $id field - it will be set to canonical value on creation
	// No validation of $id is needed

	// Check required fields (excluding $id since it's optional and will be set)
	requiredFields := []string{"$schema", "type", "title", "description"}
	for _, field := range requiredFields {
		if _, ok := schemaDoc[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate type is 'object'
	if schemaType, ok := schemaDoc["type"].(string); !ok || schemaType != "object" {
		return fmt.Errorf("root schema type must be 'object'")
	}

	// Validate title is non-empty string
	if title, ok := schemaDoc["title"].(string); !ok || title == "" {
		return fmt.Errorf("title must be a non-empty string")
	}

	// Validate description is non-empty string
	if description, ok := schemaDoc["description"].(string); !ok || description == "" {
		return fmt.Errorf("description must be a non-empty string")
	}

	// Validate properties exist
	if properties, ok := schemaDoc["properties"].(map[string]interface{}); !ok || len(properties) == 0 {
		return fmt.Errorf("schema must define non-empty properties")
	}

	return nil
}

// InjectCanonicalID parses the JSON schema, removes any existing $id, and injects the canonical $id
func InjectCanonicalID(schemaJSON string, chainID string, schemaID uint64) (string, error) {
	// Parse JSON
	var schemaDoc map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaDoc); err != nil {
		return "", fmt.Errorf("failed to parse JSON schema: %w", err)
	}

	// Remove any existing $id
	delete(schemaDoc, "$id")

	// Inject canonical $id
	canonicalID := fmt.Sprintf("vpr:verana:%s/cs/v1/js/%d", chainID, schemaID)
	schemaDoc["$id"] = canonicalID

	// Serialize back to JSON with indentation for readability
	updatedSchema, err := json.MarshalIndent(schemaDoc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize JSON schema: %w", err)
	}

	return string(updatedSchema), nil
}

// EnsureCanonicalID ensures the JSON schema has the canonical $id, updating it if needed
func EnsureCanonicalID(schemaJSON string, chainID string, schemaID uint64) (string, error) {
	// Parse JSON
	var schemaDoc map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaDoc); err != nil {
		return "", fmt.Errorf("failed to parse JSON schema: %w", err)
	}

	// Inject/update canonical $id
	canonicalID := fmt.Sprintf("vpr:verana:%s/cs/v1/js/%d", chainID, schemaID)
	schemaDoc["$id"] = canonicalID

	// Serialize back to JSON with indentation for readability
	updatedSchema, err := json.MarshalIndent(schemaDoc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize JSON schema: %w", err)
	}

	return string(updatedSchema), nil
}

func validateValidityPeriods(msg *MsgCreateCredentialSchema) error {
	// A value of 0 indicates no expiration (never expire)
	// All other values must be within the allowed range
	// For Create, fields are required (should not be nil), but we check anyway

	if msg.GetIssuerGrantorValidationValidityPeriod() != nil {
		val := msg.GetIssuerGrantorValidationValidityPeriod().GetValue()
		// uint32 cannot be negative, so skip negative check
		if val > 0 && val > DefaultCredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays {
			return fmt.Errorf("issuer grantor validation validity period exceeds maximum allowed days")
		}
	}

	if msg.GetVerifierGrantorValidationValidityPeriod() != nil {
		val := msg.GetVerifierGrantorValidationValidityPeriod().GetValue()
		// uint32 cannot be negative, so skip negative check
		if val > 0 && val > DefaultCredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays {
			return fmt.Errorf("verifier grantor validation validity period exceeds maximum allowed days")
		}
	}

	if msg.GetIssuerValidationValidityPeriod() != nil {
		val := msg.GetIssuerValidationValidityPeriod().GetValue()
		// uint32 cannot be negative, so skip negative check
		if val > 0 && val > DefaultCredentialSchemaIssuerValidationValidityPeriodMaxDays {
			return fmt.Errorf("issuer validation validity period exceeds maximum allowed days")
		}
	}

	if msg.GetVerifierValidationValidityPeriod() != nil {
		val := msg.GetVerifierValidationValidityPeriod().GetValue()
		// uint32 cannot be negative, so skip negative check
		if val > 0 && val > DefaultCredentialSchemaVerifierValidationValidityPeriodMaxDays {
			return fmt.Errorf("verifier validation validity period exceeds maximum allowed days")
		}
	}

	if msg.GetHolderValidationValidityPeriod() != nil {
		val := msg.GetHolderValidationValidityPeriod().GetValue()
		// uint32 cannot be negative, so skip negative check
		if val > 0 && val > DefaultCredentialSchemaHolderValidationValidityPeriodMaxDays {
			return fmt.Errorf("holder validation validity period exceeds maximum allowed days")
		}
	}

	return nil
}

func validatePermManagementModes(msg *MsgCreateCredentialSchema) error {
	// Check issuer perm management mode
	if msg.IssuerPermManagementMode == 0 {
		return fmt.Errorf("issuer perm management mode must be specified")
	}
	if msg.IssuerPermManagementMode > 3 {
		return fmt.Errorf("invalid issuer perm management mode: must be between 1 and 3")
	}

	// Check verifier perm management mode
	if msg.VerifierPermManagementMode == 0 {
		return fmt.Errorf("verifier perm management mode must be specified")
	}
	if msg.VerifierPermManagementMode > 3 {
		return fmt.Errorf("invalid verifier perm management mode: must be between 1 and 3")
	}

	return nil
}

func (msg *MsgUpdateCredentialSchema) ValidateBasic() error {
	// Validate creator address
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	// Check mandatory parameters
	if msg.Id == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "id cannot be 0")
	}

	if msg.Id == 0 {
		return fmt.Errorf("credential schema id is required")
	}

	return nil
}

func (msg *MsgArchiveCredentialSchema) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator address is required")
	}

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Id == 0 {
		return fmt.Errorf("credential schema id is required")
	}

	return nil
}
