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

// ValidDigestAlgorithms defines the valid digest algorithms per W3C SRI spec
var ValidDigestAlgorithms = map[string]bool{
	"sha256": true,
	"sha384": true,
	"sha512": true,
}

var _ sdk.Msg = &MsgCreateCredentialSchema{}

// NewMsgCreateCredentialSchema creates a new MsgCreateCredentialSchema instance
func NewMsgCreateCredentialSchema(
	authority string,
	operator string,
	trId uint64,
	jsonSchema string,
	issuerGrantorValidationValidityPeriod uint32,
	verifierGrantorValidationValidityPeriod uint32,
	issuerValidationValidityPeriod uint32,
	verifierValidationValidityPeriod uint32,
	holderValidationValidityPeriod uint32,
	issuerPermManagementMode uint32,
	verifierPermManagementMode uint32,
	pricingAssetType uint32,
	pricingAsset string,
	digestAlgorithm string,
) *MsgCreateCredentialSchema {
	msg := &MsgCreateCredentialSchema{
		Authority:                              authority,
		Operator:                               operator,
		TrId:                                   trId,
		JsonSchema:                             jsonSchema,
		IssuerGrantorValidationValidityPeriod:   &OptionalUInt32{Value: issuerGrantorValidationValidityPeriod},
		VerifierGrantorValidationValidityPeriod: &OptionalUInt32{Value: verifierGrantorValidationValidityPeriod},
		IssuerValidationValidityPeriod:          &OptionalUInt32{Value: issuerValidationValidityPeriod},
		VerifierValidationValidityPeriod:        &OptionalUInt32{Value: verifierValidationValidityPeriod},
		HolderValidationValidityPeriod:          &OptionalUInt32{Value: holderValidationValidityPeriod},
		IssuerPermManagementMode:                issuerPermManagementMode,
		VerifierPermManagementMode:              verifierPermManagementMode,
		PricingAssetType:                        pricingAssetType,
		PricingAsset:                            pricingAsset,
		DigestAlgorithm:                         digestAlgorithm,
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
	operator, err := sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{operator}
}

// ValidateBasic implements sdk.Msg
func (msg *MsgCreateCredentialSchema) ValidateBasic() error {
	// Validate authority address
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	// Validate operator address
	_, err = sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid operator address (%s)", err)
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

	// Validate pricing asset type and pricing asset
	if err := validatePricingAsset(msg); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	// Validate digest algorithm
	if err := validateDigestAlgorithm(msg.DigestAlgorithm); err != nil {
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

// InjectCanonicalID removes any existing $id from the JSON schema and injects the canonical $id
// as the first property, preserving the original property ordering of all other fields.
func InjectCanonicalID(schemaJSON string, chainID string, schemaID uint64) (string, error) {
	canonicalID := fmt.Sprintf("vpr:verana:%s/cs/v1/js/%d", chainID, schemaID)
	return injectOrReplaceID(schemaJSON, canonicalID)
}

// EnsureCanonicalID ensures the JSON schema has the canonical $id, updating it if needed.
// Short-circuits if the $id is already correct. Preserves original property ordering.
func EnsureCanonicalID(schemaJSON string, chainID string, schemaID uint64) (string, error) {
	canonicalID := fmt.Sprintf("vpr:verana:%s/cs/v1/js/%d", chainID, schemaID)

	// Short-circuit: if the $id is already correct, return as-is to avoid unnecessary work
	// and prevent any formatting changes on the hot query path.
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &doc); err == nil {
		if existingID, ok := doc["$id"].(string); ok && existingID == canonicalID {
			return schemaJSON, nil
		}
	}

	return injectOrReplaceID(schemaJSON, canonicalID)
}

// injectOrReplaceID performs in-place string manipulation to inject or replace the "$id" field
// in a JSON schema string without unmarshaling/remarshaling, preserving original property ordering.
// The canonical $id is always placed as the first property in the JSON object.
func injectOrReplaceID(schemaJSON string, canonicalID string) (string, error) {
	// Validate it's valid JSON first
	if !json.Valid([]byte(schemaJSON)) {
		return "", fmt.Errorf("invalid JSON schema")
	}

	// JSON-escape the canonical ID value to prevent injection
	escapedID, err := json.Marshal(canonicalID)
	if err != nil {
		return "", fmt.Errorf("failed to JSON-escape canonical ID: %w", err)
	}

	// Remove existing "$id" field if present
	cleaned, err := removeJSONField(schemaJSON, "$id")
	if err != nil {
		return "", fmt.Errorf("failed to remove existing $id: %w", err)
	}

	// Find the opening brace of the JSON object
	openBrace := -1
	for i, c := range cleaned {
		if c == '{' {
			openBrace = i
			break
		}
	}
	if openBrace == -1 {
		return "", fmt.Errorf("JSON schema is not an object")
	}

	// Build the $id entry to inject (using JSON-escaped value)
	idEntry := fmt.Sprintf(`"$id": %s`, string(escapedID))

	// Examine content after opening brace to determine formatting
	rest := cleaned[openBrace+1:]
	hasOtherProps := false
	for _, c := range rest {
		if c == '"' {
			hasOtherProps = true
			break
		}
		if c == '}' {
			break
		}
	}

	// Detect indentation style from existing content
	indent := detectIndent(cleaned)

	var result string
	if hasOtherProps {
		if indent != "" {
			// Pretty-printed: inject with matching indentation.
			// Ensure rest starts on its own line with proper indent.
			restTrimmed := trimLeadingWhitespace(rest)
			result = cleaned[:openBrace+1] + "\n" + indent + idEntry + ",\n" + indent + restTrimmed
		} else {
			// Compact: inject inline
			result = cleaned[:openBrace+1] + idEntry + "," + rest
		}
	} else {
		if indent != "" {
			result = cleaned[:openBrace+1] + "\n" + indent + idEntry + rest
		} else {
			result = cleaned[:openBrace+1] + idEntry + rest
		}
	}

	// Validate output is still valid JSON as a safety net
	if !json.Valid([]byte(result)) {
		return "", fmt.Errorf("internal error: produced invalid JSON after $id injection")
	}

	return result, nil
}

// removeJSONField removes a top-level field from a JSON object string by performing
// character-level scanning, preserving all other content exactly as-is.
func removeJSONField(jsonStr string, field string) (string, error) {
	target := fmt.Sprintf(`"%s"`, field)
	bytes := []byte(jsonStr)
	n := len(bytes)

	// Find the target key at the top level (depth == 1, i.e. inside the root object)
	depth := 0
	i := 0
	for i < n {
		c := bytes[i]

		if c == '"' {
			// Read the entire string (skip escaped chars)
			start := i
			i++
			for i < n && bytes[i] != '"' {
				if bytes[i] == '\\' {
					i++ // skip escaped character
				}
				i++
			}
			i++ // skip closing quote

			// Check if this is our target key at depth 1
			if depth == 1 {
				keyStr := string(bytes[start:i])
				if keyStr == target {
					// Found the key. Now find the colon and the value that follows.
					keyStart := start

					// Scan backwards to include any leading whitespace/newline
					ws := keyStart
					for ws > 0 && (bytes[ws-1] == ' ' || bytes[ws-1] == '\t' || bytes[ws-1] == '\n' || bytes[ws-1] == '\r') {
						ws--
					}

					// Find colon after key
					ci := i
					for ci < n && bytes[ci] != ':' {
						ci++
					}
					ci++ // skip colon

					// Skip whitespace after colon
					for ci < n && (bytes[ci] == ' ' || bytes[ci] == '\t' || bytes[ci] == '\n' || bytes[ci] == '\r') {
						ci++
					}

					// Skip the value
					valEnd, err := skipJSONValue(bytes, ci)
					if err != nil {
						return "", err
					}

					// Handle trailing comma: either remove a comma after the value, or before the key
					end := valEnd
					// Check for trailing comma
					ti := end
					for ti < n && (bytes[ti] == ' ' || bytes[ti] == '\t' || bytes[ti] == '\n' || bytes[ti] == '\r') {
						ti++
					}
					if ti < n && bytes[ti] == ',' {
						end = ti + 1
						result := string(bytes[:ws]) + string(bytes[end:])
						return result, nil
					}

					// No trailing comma — check for leading comma
					lc := ws
					for lc > 0 && (bytes[lc-1] == ' ' || bytes[lc-1] == '\t' || bytes[lc-1] == '\n' || bytes[lc-1] == '\r') {
						lc--
					}
					if lc > 0 && bytes[lc-1] == ',' {
						result := string(bytes[:lc-1]) + string(bytes[end:])
						return result, nil
					}

					// No commas at all — just remove the field
					result := string(bytes[:ws]) + string(bytes[end:])
					return result, nil
				}
			}
			continue
		}

		if c == '{' || c == '[' {
			depth++
		} else if c == '}' || c == ']' {
			depth--
		}
		i++
	}

	// Field not found, return original
	return jsonStr, nil
}

// skipJSONValue advances past a single JSON value starting at bytes[i] and returns the index after it.
func skipJSONValue(bytes []byte, i int) (int, error) {
	n := len(bytes)
	if i >= n {
		return 0, fmt.Errorf("unexpected end of JSON")
	}

	c := bytes[i]

	switch c {
	case '"':
		// String
		i++
		for i < n && bytes[i] != '"' {
			if bytes[i] == '\\' {
				i++
			}
			i++
		}
		i++ // closing quote
		return i, nil

	case '{', '[':
		// Object or array — track matching braces/brackets
		closer := byte('}')
		if c == '[' {
			closer = ']'
		}
		depth := 1
		i++
		for i < n && depth > 0 {
			if bytes[i] == '"' {
				i++
				for i < n && bytes[i] != '"' {
					if bytes[i] == '\\' {
						i++
					}
					i++
				}
			} else if bytes[i] == c {
				depth++
			} else if bytes[i] == closer {
				depth--
			}
			i++
		}
		return i, nil

	default:
		// Number, bool, null — advance until delimiter
		for i < n && bytes[i] != ',' && bytes[i] != '}' && bytes[i] != ']' && bytes[i] != ' ' && bytes[i] != '\t' && bytes[i] != '\n' && bytes[i] != '\r' {
			i++
		}
		return i, nil
	}
}

// detectIndent detects the indentation string used in a JSON document by looking
// at the first indented line after the opening brace.
func detectIndent(jsonStr string) string {
	inObject := false
	i := 0
	for i < len(jsonStr) {
		if jsonStr[i] == '{' {
			inObject = true
			i++
			continue
		}
		if inObject && jsonStr[i] == '\n' {
			i++
			// Collect whitespace
			start := i
			for i < len(jsonStr) && (jsonStr[i] == ' ' || jsonStr[i] == '\t') {
				i++
			}
			if i > start && i < len(jsonStr) && jsonStr[i] != '}' {
				return jsonStr[start:i]
			}
			continue
		}
		i++
	}
	return ""
}

// trimLeadingWhitespace removes leading whitespace characters (spaces, tabs, newlines)
// from a string.
func trimLeadingWhitespace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	return s[i:]
}

// CanonicalizeJCS serializes a JSON string using the JSON Canonicalization Scheme (JCS)
// as defined in RFC 8785: keys sorted alphabetically, no insignificant whitespace.
// json.Marshal on interface{} sorts map keys in Unicode code point order, satisfying JCS.
func CanonicalizeJCS(schemaJSON string) (string, error) {
	var doc interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &doc); err != nil {
		return "", fmt.Errorf("failed to parse JSON for JCS canonicalization: %w", err)
	}
	canonical, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to JCS-canonicalize JSON: %w", err)
	}
	return string(canonical), nil
}

func validateValidityPeriods(msg *MsgCreateCredentialSchema) error {
	// [MOD-CS-MSG-1-2-1] All validity period fields are mandatory
	if msg.GetIssuerGrantorValidationValidityPeriod() == nil {
		return fmt.Errorf("issuer_grantor_validation_validity_period is mandatory")
	}
	if msg.GetVerifierGrantorValidationValidityPeriod() == nil {
		return fmt.Errorf("verifier_grantor_validation_validity_period is mandatory")
	}
	if msg.GetIssuerValidationValidityPeriod() == nil {
		return fmt.Errorf("issuer_validation_validity_period is mandatory")
	}
	if msg.GetVerifierValidationValidityPeriod() == nil {
		return fmt.Errorf("verifier_validation_validity_period is mandatory")
	}
	if msg.GetHolderValidationValidityPeriod() == nil {
		return fmt.Errorf("holder_validation_validity_period is mandatory")
	}

	// Validate ranges: must be between 0 (never expire) and max_days
	val := msg.GetIssuerGrantorValidationValidityPeriod().GetValue()
	if val > 0 && val > DefaultCredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays {
		return fmt.Errorf("issuer grantor validation validity period exceeds maximum allowed days")
	}

	val = msg.GetVerifierGrantorValidationValidityPeriod().GetValue()
	if val > 0 && val > DefaultCredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays {
		return fmt.Errorf("verifier grantor validation validity period exceeds maximum allowed days")
	}

	val = msg.GetIssuerValidationValidityPeriod().GetValue()
	if val > 0 && val > DefaultCredentialSchemaIssuerValidationValidityPeriodMaxDays {
		return fmt.Errorf("issuer validation validity period exceeds maximum allowed days")
	}

	val = msg.GetVerifierValidationValidityPeriod().GetValue()
	if val > 0 && val > DefaultCredentialSchemaVerifierValidationValidityPeriodMaxDays {
		return fmt.Errorf("verifier validation validity period exceeds maximum allowed days")
	}

	val = msg.GetHolderValidationValidityPeriod().GetValue()
	if val > 0 && val > DefaultCredentialSchemaHolderValidationValidityPeriodMaxDays {
		return fmt.Errorf("holder validation validity period exceeds maximum allowed days")
	}

	return nil
}

func validatePermManagementModes(msg *MsgCreateCredentialSchema) error {
	if msg.IssuerPermManagementMode == 0 {
		return fmt.Errorf("issuer perm management mode must be specified")
	}
	if msg.IssuerPermManagementMode > 3 {
		return fmt.Errorf("invalid issuer perm management mode: must be between 1 and 3")
	}

	if msg.VerifierPermManagementMode == 0 {
		return fmt.Errorf("verifier perm management mode must be specified")
	}
	if msg.VerifierPermManagementMode > 3 {
		return fmt.Errorf("invalid verifier perm management mode: must be between 1 and 3")
	}

	return nil
}

func validatePricingAsset(msg *MsgCreateCredentialSchema) error {
	if msg.PricingAssetType == 0 {
		return fmt.Errorf("pricing_asset_type must be specified")
	}
	if msg.PricingAssetType > 3 {
		return fmt.Errorf("invalid pricing_asset_type: must be between 1 and 3")
	}

	if msg.PricingAsset == "" {
		return fmt.Errorf("pricing_asset is mandatory")
	}

	// If TU, pricing_asset must be "tu"
	if msg.PricingAssetType == uint32(PricingAssetType_TU) && msg.PricingAsset != "tu" {
		return fmt.Errorf("pricing_asset must be 'tu' when pricing_asset_type is TU")
	}

	return nil
}

func validateDigestAlgorithm(algorithm string) error {
	if algorithm == "" {
		return fmt.Errorf("digest_algorithm is mandatory")
	}
	if !ValidDigestAlgorithms[algorithm] {
		return fmt.Errorf("invalid digest_algorithm '%s': must be one of sha256, sha384, sha512", algorithm)
	}
	return nil
}

func (msg *MsgUpdateCredentialSchema) ValidateBasic() error {
	// Validate authority address
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	// Validate operator address
	_, err = sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid operator address (%s)", err)
	}

	if msg.Id == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "id cannot be 0")
	}

	if msg.GetIssuerGrantorValidationValidityPeriod() == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "issuer_grantor_validation_validity_period is mandatory")
	}
	if msg.GetVerifierGrantorValidationValidityPeriod() == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "verifier_grantor_validation_validity_period is mandatory")
	}
	if msg.GetIssuerValidationValidityPeriod() == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "issuer_validation_validity_period is mandatory")
	}
	if msg.GetVerifierValidationValidityPeriod() == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "verifier_validation_validity_period is mandatory")
	}
	if msg.GetHolderValidationValidityPeriod() == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "holder_validation_validity_period is mandatory")
	}

	return nil
}

func (msg *MsgArchiveCredentialSchema) ValidateBasic() error {
	// Validate authority address
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	// Validate operator address
	_, err = sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid operator address (%s)", err)
	}

	if msg.Id == 0 {
		return fmt.Errorf("credential schema id is required")
	}

	return nil
}
