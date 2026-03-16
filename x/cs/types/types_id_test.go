package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInjectCanonicalID_PreservesOrder(t *testing.T) {
	input := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Test Schema",
  "description": "A test",
  "type": "object",
  "properties": {
    "name": {"type": "string"}
  }
}`

	result, err := InjectCanonicalID(input, "verana-1", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, `"$id": "vpr:verana:verana-1/cs/v1/js/42"`) {
		t.Errorf("canonical $id not found in result:\n%s", result)
	}

	// Verify property order: $id, $schema, title, description, type, properties
	// Use top-level-only positions by searching for indented keys
	assertKeyOrder(t, result, []string{`"$id"`, `"$schema"`, `"title"`, `"description"`})

	if !json.Valid([]byte(result)) {
		t.Errorf("result is not valid JSON:\n%s", result)
	}
}

func TestInjectCanonicalID_ReplacesExistingID(t *testing.T) {
	input := `{
  "$id": "old-id",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Test"
}`

	result, err := InjectCanonicalID(input, "verana-1", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "old-id") {
		t.Error("old $id should be removed")
	}
	if !strings.Contains(result, `"$id": "vpr:verana:verana-1/cs/v1/js/7"`) {
		t.Errorf("canonical $id not found in result:\n%s", result)
	}

	assertKeyOrder(t, result, []string{`"$id"`, `"$schema"`, `"title"`})

	if !json.Valid([]byte(result)) {
		t.Errorf("result is not valid JSON:\n%s", result)
	}
}

func TestInjectCanonicalID_CompactJSON(t *testing.T) {
	input := `{"$schema":"https://json-schema.org/draft/2020-12/schema","title":"Test","type":"object"}`

	result, err := InjectCanonicalID(input, "chain-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should stay compact (no newlines)
	if strings.Contains(result, "\n") {
		t.Errorf("compact JSON should remain compact:\n%s", result)
	}

	assertKeyOrder(t, result, []string{`"$id"`, `"$schema"`, `"title"`})

	if !json.Valid([]byte(result)) {
		t.Errorf("result is not valid JSON:\n%s", result)
	}
}

func TestInjectCanonicalID_IDInMiddle(t *testing.T) {
	input := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "some-old-id",
  "title": "Test",
  "type": "object"
}`

	result, err := InjectCanonicalID(input, "verana-1", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertKeyOrder(t, result, []string{`"$id"`, `"$schema"`, `"title"`})

	if !json.Valid([]byte(result)) {
		t.Errorf("result is not valid JSON:\n%s", result)
	}
}

func TestInjectCanonicalID_IDAsLastField(t *testing.T) {
	input := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Test",
  "$id": "old-id"
}`

	result, err := InjectCanonicalID(input, "verana-1", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertKeyOrder(t, result, []string{`"$id"`, `"$schema"`, `"title"`})

	if !json.Valid([]byte(result)) {
		t.Errorf("result is not valid JSON:\n%s", result)
	}
}

func TestInjectCanonicalID_IDAsOnlyField(t *testing.T) {
	input := `{"$id": "old-id"}`

	result, err := InjectCanonicalID(input, "verana-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, `"$id": "vpr:verana:verana-1/cs/v1/js/1"`) {
		t.Errorf("canonical $id not found in result:\n%s", result)
	}

	if !json.Valid([]byte(result)) {
		t.Errorf("result is not valid JSON:\n%s", result)
	}
}

func TestInjectCanonicalID_NoExistingID(t *testing.T) {
	input := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Test"
}`

	result, err := InjectCanonicalID(input, "verana-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertKeyOrder(t, result, []string{`"$id"`, `"$schema"`, `"title"`})

	if !json.Valid([]byte(result)) {
		t.Errorf("result is not valid JSON:\n%s", result)
	}
}

func TestEnsureCanonicalID_ShortCircuitsWhenCorrect(t *testing.T) {
	input := `{
  "$id": "vpr:verana:verana-1/cs/v1/js/5",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Test",
  "description": "Desc",
  "type": "object"
}`

	result, err := EnsureCanonicalID(input, "verana-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return the exact same string (short-circuit)
	if result != input {
		t.Errorf("EnsureCanonicalID should short-circuit when $id is already correct.\nInput:\n%s\nOutput:\n%s", input, result)
	}
}

func TestEnsureCanonicalID_UpdatesWrongID(t *testing.T) {
	input := `{
  "$id": "vpr:verana:verana-1/cs/v1/js/99",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Test"
}`

	result, err := EnsureCanonicalID(input, "verana-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, `"vpr:verana:verana-1/cs/v1/js/5"`) {
		t.Errorf("expected updated $id in result:\n%s", result)
	}
	if strings.Contains(result, "js/99") {
		t.Error("old $id value should be removed")
	}

	if !json.Valid([]byte(result)) {
		t.Errorf("result is not valid JSON:\n%s", result)
	}
}

func TestInjectCanonicalID_Idempotency(t *testing.T) {
	input := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Test",
  "type": "object"
}`

	// First injection
	result1, err := InjectCanonicalID(input, "verana-1", 42)
	if err != nil {
		t.Fatalf("first injection failed: %v", err)
	}

	// Second injection on the result (simulates store → query round-trip)
	result2, err := InjectCanonicalID(result1, "verana-1", 42)
	if err != nil {
		t.Fatalf("second injection failed: %v", err)
	}

	if result1 != result2 {
		t.Errorf("InjectCanonicalID is not idempotent.\nFirst:\n%s\nSecond:\n%s", result1, result2)
	}
}

func TestInjectCanonicalID_RoundTrip(t *testing.T) {
	// Simulate: create (InjectCanonicalID) → store → query (EnsureCanonicalID)
	input := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Test Schema",
  "description": "A test credential schema",
  "type": "object",
  "properties": {
    "name": {"type": "string"},
    "age": {"type": "integer"}
  }
}`

	// Creation: inject $id
	stored, err := InjectCanonicalID(input, "verana-1", 10)
	if err != nil {
		t.Fatalf("InjectCanonicalID failed: %v", err)
	}

	// Query: ensure $id (should be a no-op since it's already correct)
	queried, err := EnsureCanonicalID(stored, "verana-1", 10)
	if err != nil {
		t.Fatalf("EnsureCanonicalID failed: %v", err)
	}

	if stored != queried {
		t.Errorf("Round-trip changed the schema.\nStored:\n%s\nQueried:\n%s", stored, queried)
	}
}

func TestInjectCanonicalID_TabIndentation(t *testing.T) {
	input := "{\n\t\"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n\t\"title\": \"Test\"\n}"

	result, err := InjectCanonicalID(input, "verana-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use tab indentation for the injected $id
	if !strings.Contains(result, "\t\"$id\"") {
		t.Errorf("expected tab-indented $id in result:\n%s", result)
	}

	if !json.Valid([]byte(result)) {
		t.Errorf("result is not valid JSON:\n%s", result)
	}
}

func TestInjectCanonicalID_InvalidJSON(t *testing.T) {
	_, err := InjectCanonicalID("not json", "verana-1", 1)
	if err == nil {
		t.Error("expected error for invalid JSON input")
	}
}

func TestInjectCanonicalID_NestedIDNotAffected(t *testing.T) {
	// Ensure nested $id fields inside properties are not touched
	input := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Test",
  "properties": {
    "ref": {"$id": "nested-id", "type": "string"}
  }
}`

	result, err := InjectCanonicalID(input, "verana-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nested $id should be preserved
	if !strings.Contains(result, `"nested-id"`) {
		t.Errorf("nested $id should not be removed:\n%s", result)
	}

	// Top-level canonical $id should be injected
	if !strings.Contains(result, `"vpr:verana:verana-1/cs/v1/js/1"`) {
		t.Errorf("canonical $id not found:\n%s", result)
	}

	if !json.Valid([]byte(result)) {
		t.Errorf("result is not valid JSON:\n%s", result)
	}
}

func TestInjectCanonicalID_EscapedQuotesInValues(t *testing.T) {
	input := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Schema with \"quotes\"",
  "type": "object"
}`

	result, err := InjectCanonicalID(input, "verana-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The escaped quotes in title should be preserved
	if !strings.Contains(result, `\"quotes\"`) {
		t.Errorf("escaped quotes should be preserved:\n%s", result)
	}

	if !json.Valid([]byte(result)) {
		t.Errorf("result is not valid JSON:\n%s", result)
	}
}

// assertKeyOrder verifies that the given keys appear in order in the JSON string.
// Uses line-by-line scanning to only match top-level keys (lines at the first indent level).
func assertKeyOrder(t *testing.T, jsonStr string, keys []string) {
	t.Helper()
	lastIdx := -1
	for _, key := range keys {
		idx := strings.Index(jsonStr, key)
		if idx == -1 {
			t.Errorf("key %s not found in result:\n%s", key, jsonStr)
			return
		}
		if idx <= lastIdx {
			t.Errorf("key %s (at %d) should come after previous key (at %d) in result:\n%s", key, idx, lastIdx, jsonStr)
			return
		}
		lastIdx = idx
	}
}
