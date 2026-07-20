package arena

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRealAttemptFixtureAndGeneratedOutputMatchSchemas(t *testing.T) {
	root := repoRoot(t)
	manifestPath := filepath.Join(root, "examples", "real-attempts", "valid", "month5-ten-pair-manifest.json")
	manifestSchemaPath := filepath.Join(root, "docs", "contracts", "arena-real-attempt-manifest-v0.1.schema.json")
	portfolioSchemaPath := filepath.Join(root, "docs", "contracts", "arena-real-attempt-task-portfolio-v0.1.schema.json")
	evidenceSchemaPath := filepath.Join(root, "docs", "contracts", "arena-real-attempt-evidence-v0.1.schema.json")
	comparisonSchemaPath := filepath.Join(root, "docs", "contracts", "arena-real-attempt-comparison-v0.1.schema.json")

	validateJSONFileAgainstTestSchema(t, manifestPath, manifestSchemaPath)
	manifestBody, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifestReference RealAttemptManifest
	if err := json.Unmarshal(manifestBody, &manifestReference); err != nil {
		t.Fatal(err)
	}
	portfolioPath := filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(manifestReference.TaskPortfolio.Path))
	validateJSONFileAgainstTestSchema(t, portfolioPath, portfolioSchemaPath)
	manifest, err := LoadAndValidateRealAttemptManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range manifest.Pairs {
		for _, attempt := range []RealAttempt{pair.BareCodex, pair.AOOrchestration} {
			evidencePath := filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(attempt.Evidence.Path))
			validateJSONFileAgainstTestSchema(t, evidencePath, evidenceSchemaPath)
		}
	}

	out := filepath.Join(t.TempDir(), "comparison.json")
	if _, err := CompareRealAttempts(manifestPath, out); err != nil {
		t.Fatal(err)
	}
	validateJSONFileAgainstTestSchema(t, out, comparisonSchemaPath)
}

func TestRealAttemptSchemaTextPatternAndRuntimeSafetySuperset(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "docs", "contracts", "arena-real-attempt-manifest-v0.1.schema.json")
	body, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	var schema testJSONSchema
	if err := json.Unmarshal(body, &schema); err != nil {
		t.Fatal(err)
	}
	publicText := schema.Defs["public_text"]

	for _, value := range []string{" leading", "trailing ", "line one\nline two", "nul\x00byte"} {
		if err := validateTestSchemaValue(value, publicText, &schema, "$.text"); err == nil {
			t.Fatalf("schema accepted non-trimmed or multiline text %q", value)
		}
		if err := validatePublicText("text", value); err == nil {
			t.Fatalf("runtime accepted non-trimmed or multiline text %q", value)
		}
	}

	for _, value := range []string{"credential ghp_abcdefghijklmnopqrstuvwxyz", "artifact path:/root/private/result.json"} {
		if err := validateTestSchemaValue(value, publicText, &schema, "$.text"); err != nil {
			t.Fatalf("structural schema unexpectedly rejected semantic-safety vector %q: %v", value, err)
		}
		if err := validatePublicText("text", value); err == nil {
			t.Fatalf("runtime safety superset accepted %q", value)
		}
	}
}

func TestRealAttemptManifestSchemaMatchesRuntimePathAndLengthBounds(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "docs", "contracts", "arena-real-attempt-manifest-v0.1.schema.json")
	body, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	var schema testJSONSchema
	if err := json.Unmarshal(body, &schema); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		path string
		want bool
	}{
		{path: "evidence/result.json", want: true},
		{path: "evidence/nested/result-01.json", want: true},
		{path: "evidence/with space.json", want: false},
		{path: "evidence/Upper.json", want: false},
		{path: "evidence/結果.json", want: false},
		{path: "evidence//result.json", want: false},
		{path: "evidence/../result.json", want: false},
		{path: `.hidden.json`, want: false},
	} {
		runtimeErr := validateEvidenceReference(RealAttemptEvidenceReference{Path: tc.path, SHA256: strings.Repeat("a", 64)})
		schemaErr := validateTestSchemaValue(
			map[string]any{"path": tc.path, "sha256": strings.Repeat("a", 64)},
			schema.Defs["evidence_reference"],
			&schema,
			"$.evidence",
		)
		if (runtimeErr == nil) != tc.want || (schemaErr == nil) != tc.want {
			t.Fatalf("path %q runtimeErr=%v schemaErr=%v wantValid=%t", tc.path, runtimeErr, schemaErr, tc.want)
		}
	}

	for _, tc := range []struct {
		value string
		want  bool
	}{
		{value: strings.Repeat("界", 500), want: true},
		{value: strings.Repeat("界", 501), want: false},
	} {
		runtimeErr := validateRealAttemptAnnotations("limitations", []string{tc.value})
		schemaErr := validateTestSchemaValue([]any{tc.value}, schema.Defs["annotations"], &schema, "$.limitations")
		if (runtimeErr == nil) != tc.want || (schemaErr == nil) != tc.want {
			t.Fatalf("runes=%d runtimeErr=%v schemaErr=%v wantValid=%t", utf8.RuneCountInString(tc.value), runtimeErr, schemaErr, tc.want)
		}
	}
}

type testJSONSchema struct {
	Ref                  string                     `json:"$ref"`
	Type                 string                     `json:"type"`
	Const                any                        `json:"const"`
	Enum                 []any                      `json:"enum"`
	Required             []string                   `json:"required"`
	Properties           map[string]*testJSONSchema `json:"properties"`
	AdditionalProperties *bool                      `json:"additionalProperties"`
	Items                *testJSONSchema            `json:"items"`
	MinItems             *int                       `json:"minItems"`
	MaxItems             *int                       `json:"maxItems"`
	MinLength            *int                       `json:"minLength"`
	MaxLength            *int                       `json:"maxLength"`
	Pattern              string                     `json:"pattern"`
	Minimum              *float64                   `json:"minimum"`
	Maximum              *float64                   `json:"maximum"`
	OneOf                []*testJSONSchema          `json:"oneOf"`
	Defs                 map[string]*testJSONSchema `json:"$defs"`
}

func validateJSONFileAgainstTestSchema(t *testing.T, documentPath, schemaPath string) {
	t.Helper()
	documentBody, err := os.ReadFile(documentPath)
	if err != nil {
		t.Fatal(err)
	}
	var document any
	if err := json.Unmarshal(documentBody, &document); err != nil {
		t.Fatalf("parse %s: %v", documentPath, err)
	}
	schemaBody, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	var schema testJSONSchema
	if err := json.Unmarshal(schemaBody, &schema); err != nil {
		t.Fatalf("parse %s: %v", schemaPath, err)
	}
	if err := validateTestSchemaValue(document, &schema, &schema, "$"); err != nil {
		t.Fatalf("%s does not match %s: %v", documentPath, schemaPath, err)
	}
}

func validateTestSchemaValue(value any, schema, root *testJSONSchema, location string) error {
	if schema.Ref != "" {
		const prefix = "#/$defs/"
		if !strings.HasPrefix(schema.Ref, prefix) {
			return fmt.Errorf("%s unsupported schema reference %q", location, schema.Ref)
		}
		resolved := root.Defs[strings.TrimPrefix(schema.Ref, prefix)]
		if resolved == nil {
			return fmt.Errorf("%s unresolved schema reference %q", location, schema.Ref)
		}
		return validateTestSchemaValue(value, resolved, root, location)
	}
	if schema.Const != nil && !reflect.DeepEqual(value, schema.Const) {
		return fmt.Errorf("%s value %#v does not equal const %#v", location, value, schema.Const)
	}
	if len(schema.Enum) > 0 {
		matched := false
		for _, candidate := range schema.Enum {
			matched = matched || reflect.DeepEqual(value, candidate)
		}
		if !matched {
			return fmt.Errorf("%s value %#v is not in enum", location, value)
		}
	}
	if len(schema.OneOf) > 0 {
		matches := 0
		for _, candidate := range schema.OneOf {
			if validateTestSchemaValue(value, candidate, root, location) == nil {
				matches++
			}
		}
		if matches != 1 {
			return fmt.Errorf("%s matched %d oneOf branches", location, matches)
		}
	}

	switch schema.Type {
	case "":
		return validateTestSchemaObjectConstraints(value, schema, root, location)
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s is not an object", location)
		}
		for _, name := range schema.Required {
			if _, ok := object[name]; !ok {
				return fmt.Errorf("%s missing required field %q", location, name)
			}
		}
		for name, field := range object {
			propertySchema, ok := schema.Properties[name]
			if !ok {
				if schema.AdditionalProperties != nil && !*schema.AdditionalProperties {
					return fmt.Errorf("%s has unknown field %q", location, name)
				}
				continue
			}
			if err := validateTestSchemaValue(field, propertySchema, root, location+"."+name); err != nil {
				return err
			}
		}
	case "array":
		array, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s is not an array", location)
		}
		if schema.MinItems != nil && len(array) < *schema.MinItems {
			return fmt.Errorf("%s has fewer than %d items", location, *schema.MinItems)
		}
		if schema.MaxItems != nil && len(array) > *schema.MaxItems {
			return fmt.Errorf("%s has more than %d items", location, *schema.MaxItems)
		}
		if schema.Items != nil {
			for i, item := range array {
				if err := validateTestSchemaValue(item, schema.Items, root, fmt.Sprintf("%s[%d]", location, i)); err != nil {
					return err
				}
			}
		}
	case "string":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s is not a string", location)
		}
		length := utf8.RuneCountInString(text)
		if schema.MinLength != nil && length < *schema.MinLength {
			return fmt.Errorf("%s is shorter than %d", location, *schema.MinLength)
		}
		if schema.MaxLength != nil && length > *schema.MaxLength {
			return fmt.Errorf("%s is longer than %d", location, *schema.MaxLength)
		}
		if schema.Pattern != "" && !regexp.MustCompile(schema.Pattern).MatchString(text) {
			return fmt.Errorf("%s does not match %q", location, schema.Pattern)
		}
	case "integer", "number":
		number, ok := value.(float64)
		if !ok || (schema.Type == "integer" && math.Trunc(number) != number) {
			return fmt.Errorf("%s is not a %s", location, schema.Type)
		}
		if schema.Minimum != nil && number < *schema.Minimum {
			return fmt.Errorf("%s is below %v", location, *schema.Minimum)
		}
		if schema.Maximum != nil && number > *schema.Maximum {
			return fmt.Errorf("%s is above %v", location, *schema.Maximum)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s is not a boolean", location)
		}
	default:
		return fmt.Errorf("%s unsupported schema type %q", location, schema.Type)
	}
	return nil
}

func validateTestSchemaObjectConstraints(value any, schema, root *testJSONSchema, location string) error {
	if len(schema.Properties) == 0 && len(schema.Required) == 0 {
		return nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%s is not an object", location)
	}
	for _, name := range schema.Required {
		if _, ok := object[name]; !ok {
			return fmt.Errorf("%s missing required field %q", location, name)
		}
	}
	for name, propertySchema := range schema.Properties {
		if field, ok := object[name]; ok {
			if err := validateTestSchemaValue(field, propertySchema, root, location+"."+name); err != nil {
				return err
			}
		}
	}
	return nil
}
