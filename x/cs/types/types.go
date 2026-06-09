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
      "pattern": "^vpr:verana:VPR_CHAIN_ID:cs:VPR_CREDENTIAL_SCHEMA_ID$",
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
      "$id": "vpr:verana:mainnet:cs:1",
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
	ecosystemId uint64,
	jsonSchema string,
	issuerGrantorValidationValidityPeriod uint32,
	verifierGrantorValidationValidityPeriod uint32,
	issuerValidationValidityPeriod uint32,
	verifierValidationValidityPeriod uint32,
	holderValidationValidityPeriod uint32,
	issuerOnboardingMode uint32,
	verifierOnboardingMode uint32,
	holderOnboardingMode uint32,
	pricingAssetType uint32,
	pricingAsset string,
	digestAlgorithm string,
) *MsgCreateCredentialSchema {
	msg := &MsgCreateCredentialSchema{
		Corporation:                             authority,
		Operator:                                operator,
		EcosystemId:                             ecosystemId,
		JsonSchema:                              jsonSchema,
		IssuerGrantorValidationValidityPeriod:   &OptionalUInt32{Value: issuerGrantorValidationValidityPeriod},
		VerifierGrantorValidationValidityPeriod: &OptionalUInt32{Value: verifierGrantorValidationValidityPeriod},
		IssuerValidationValidityPeriod:          &OptionalUInt32{Value: issuerValidationValidityPeriod},
		VerifierValidationValidityPeriod:        &OptionalUInt32{Value: verifierValidationValidityPeriod},
		HolderValidationValidityPeriod:          &OptionalUInt32{Value: holderValidationValidityPeriod},
		IssuerOnboardingMode:                    issuerOnboardingMode,
		VerifierOnboardingMode:                  verifierOnboardingMode,
		HolderOnboardingMode:                    holderOnboardingMode,
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
	// Validate corporation address
	_, err := sdk.AccAddressFromBech32(msg.Corporation)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid corporation address (%s)", err)
	}

	// Validate operator address
	_, err = sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid operator address (%s)", err)
	}

	// Check mandatory parameters
	if msg.EcosystemId == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "ecosystem_id cannot be 0")
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
	canonicalID := fmt.Sprintf("vpr:verana:%s:cs:%d", chainID, schemaID)
	return injectOrReplaceID(schemaJSON, canonicalID)
}

// EnsureCanonicalID ensures the JSON schema has the canonical $id, updating it if needed.
// Short-circuits if the $id is already correct. Preserves original property ordering.
func EnsureCanonicalID(schemaJSON string, chainID string, schemaID uint64) (string, error) {
	canonicalID := fmt.Sprintf("vpr:verana:%s:cs:%d", chainID, schemaID)

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

	return nil
}

func validatePermManagementModes(msg *MsgCreateCredentialSchema) error {
	if msg.IssuerOnboardingMode == 0 {
		return fmt.Errorf("issuer onboarding mode must be specified")
	}
	if msg.IssuerOnboardingMode > 3 {
		return fmt.Errorf("invalid issuer onboarding mode: must be between 1 and 3")
	}

	if msg.VerifierOnboardingMode == 0 {
		return fmt.Errorf("verifier onboarding mode must be specified")
	}
	if msg.VerifierOnboardingMode > 3 {
		return fmt.Errorf("invalid verifier onboarding mode: must be between 1 and 3")
	}

	// [MOD-CS-MSG-1-2-1] holder_onboarding_mode MUST be a valid HolderOnboardingMode.
	// Enum values: ISSUER_VALIDATION_PROCESS=1, PERMISSIONLESS=2. UNSPECIFIED=0 is invalid.
	if msg.HolderOnboardingMode == 0 {
		return fmt.Errorf("holder onboarding mode must be specified")
	}
	if msg.HolderOnboardingMode > 2 {
		return fmt.Errorf("invalid holder onboarding mode: must be between 1 and 2")
	}

	return nil
}

// iso4217Currencies holds the active ISO-4217 alpha-3 currency codes accepted
// for FIAT-priced credential schemas. Kept as a package-level set so spec
// [MOD-CS-MSG-1] NOTE "pricing_asset MUST be an ISO-4217 currency code" is
// enforced stateless-ly in ValidateBasic.
var iso4217Currencies = map[string]struct{}{
	"AED": {}, "AFN": {}, "ALL": {}, "AMD": {}, "ANG": {}, "AOA": {}, "ARS": {},
	"AUD": {}, "AWG": {}, "AZN": {}, "BAM": {}, "BBD": {}, "BDT": {}, "BGN": {},
	"BHD": {}, "BIF": {}, "BMD": {}, "BND": {}, "BOB": {}, "BRL": {}, "BSD": {},
	"BTN": {}, "BWP": {}, "BYN": {}, "BZD": {}, "CAD": {}, "CDF": {}, "CHF": {},
	"CLP": {}, "CNY": {}, "COP": {}, "CRC": {}, "CUP": {}, "CVE": {}, "CZK": {},
	"DJF": {}, "DKK": {}, "DOP": {}, "DZD": {}, "EGP": {}, "ERN": {}, "ETB": {},
	"EUR": {}, "FJD": {}, "FKP": {}, "GBP": {}, "GEL": {}, "GHS": {}, "GIP": {},
	"GMD": {}, "GNF": {}, "GTQ": {}, "GYD": {}, "HKD": {}, "HNL": {}, "HTG": {},
	"HUF": {}, "IDR": {}, "ILS": {}, "INR": {}, "IQD": {}, "IRR": {}, "ISK": {},
	"JMD": {}, "JOD": {}, "JPY": {}, "KES": {}, "KGS": {}, "KHR": {}, "KMF": {},
	"KPW": {}, "KRW": {}, "KWD": {}, "KYD": {}, "KZT": {}, "LAK": {}, "LBP": {},
	"LKR": {}, "LRD": {}, "LSL": {}, "LYD": {}, "MAD": {}, "MDL": {}, "MGA": {},
	"MKD": {}, "MMK": {}, "MNT": {}, "MOP": {}, "MRU": {}, "MUR": {}, "MVR": {},
	"MWK": {}, "MXN": {}, "MYR": {}, "MZN": {}, "NAD": {}, "NGN": {}, "NIO": {},
	"NOK": {}, "NPR": {}, "NZD": {}, "OMR": {}, "PAB": {}, "PEN": {}, "PGK": {},
	"PHP": {}, "PKR": {}, "PLN": {}, "PYG": {}, "QAR": {}, "RON": {}, "RSD": {},
	"RUB": {}, "RWF": {}, "SAR": {}, "SBD": {}, "SCR": {}, "SDG": {}, "SEK": {},
	"SGD": {}, "SHP": {}, "SLE": {}, "SOS": {}, "SRD": {}, "SSP": {}, "STN": {},
	"SVC": {}, "SYP": {}, "SZL": {}, "THB": {}, "TJS": {}, "TMT": {}, "TND": {},
	"TOP": {}, "TRY": {}, "TTD": {}, "TWD": {}, "TZS": {}, "UAH": {}, "UGX": {},
	"USD": {}, "UYU": {}, "UZS": {}, "VES": {}, "VND": {}, "VUV": {}, "WST": {},
	"XAF": {}, "XCD": {}, "XOF": {}, "XPF": {}, "YER": {}, "ZAR": {}, "ZMW": {},
	"ZWL": {},
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

	// [MOD-CS-MSG-1] pricing_asset semantics by pricing_asset_type.
	switch msg.PricingAssetType {
	case uint32(PricingAssetType_TU):
		// If TU, pricing_asset must be "tu"
		if msg.PricingAsset != "tu" {
			return fmt.Errorf("pricing_asset must be 'tu' when pricing_asset_type is TU")
		}
	case uint32(PricingAssetType_FIAT):
		// Spec NOTE: "When pricing_currency is set to FIAT, pricing_asset MUST
		// be an ISO-4217 currency code." Validated against the alpha-3 set.
		if _, ok := iso4217Currencies[msg.PricingAsset]; !ok {
			return fmt.Errorf("pricing_asset %q is not a valid ISO-4217 currency code", msg.PricingAsset)
		}
	case uint32(PricingAssetType_COIN):
		// Spec shows examples like "uvna", "ibc/...", "factory/...". Cosmos SDK
		// denom format accepts lowercase alphanumeric with separators; a full
		// denom-regex check happens at bank level, but we can reject obvious
		// garbage early by enforcing the canonical denom pattern.
		if err := sdk.ValidateDenom(msg.PricingAsset); err != nil {
			return fmt.Errorf("pricing_asset must be a valid Cosmos denom when pricing_asset_type is COIN: %w", err)
		}
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

func (m *MsgCreateSchemaAuthorizationPolicy) Route() string { return ModuleName }
func (m *MsgIncreaseActiveSchemaAuthorizationPolicyVersion) Route() string { return ModuleName }
func (m *MsgRevokeSchemaAuthorizationPolicy) Route() string { return ModuleName }

func (m *MsgCreateSchemaAuthorizationPolicy) ValidateBasic() error {
	if m.Corporation == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "corporation is required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Corporation); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "invalid corporation address")
	}
	if m.Operator == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "operator is required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Operator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "invalid operator address")
	}
	if m.SchemaId == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "schema_id is required")
	}
	if m.Role == SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_UNSPECIFIED {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "role is required")
	}
	if m.Url == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "url is required")
	}
	if m.DigestSri == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "digest_sri is required")
	}
	// [MOD-CS-MSG-5-3] effective_from/effective_until are set to null at creation per spec v4 draft 13.
	return nil
}

func (m *MsgIncreaseActiveSchemaAuthorizationPolicyVersion) ValidateBasic() error {
	if m.Corporation == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "corporation is required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Corporation); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "invalid corporation address")
	}
	if m.Operator == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "operator is required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Operator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "invalid operator address")
	}
	if m.SchemaId == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "schema_id is required")
	}
	if m.Role == SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_UNSPECIFIED {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "role is required")
	}
	return nil
}

func (m *MsgRevokeSchemaAuthorizationPolicy) ValidateBasic() error {
	if m.Corporation == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "corporation is required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Corporation); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "invalid corporation address")
	}
	if m.Operator == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "operator is required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Operator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "invalid operator address")
	}
	if m.SchemaId == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "schema_id is required")
	}
	if m.Role == SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_UNSPECIFIED {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "role is required")
	}
	if m.Version == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "version is required")
	}
	return nil
}

func (msg *MsgUpdateCredentialSchema) ValidateBasic() error {
	// Validate corporation address
	_, err := sdk.AccAddressFromBech32(msg.Corporation)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid corporation address (%s)", err)
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
	// Validate corporation address
	_, err := sdk.AccAddressFromBech32(msg.Corporation)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid corporation address (%s)", err)
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
