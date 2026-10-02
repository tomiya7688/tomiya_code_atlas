package conformance

import (
	"os"
	"strings"
	"testing"
)

const fixturePath = "../../../tests/fixtures/backend_conformance/serialized_common_ir_v1.json"

func TestSerializedCommonIRFixture(t *testing.T) {
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSerializedFixture(data); err != nil {
		t.Fatalf("shared fixture does not conform: %v", err)
	}
}

func TestSerializedCommonIRFixtureRejectsContractErrors(t *testing.T) {
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		from string
		to   string
		want string
	}{
		{"missing version", `"schema_version": "1",`, "", "fixture.schema_version is required"},
		{"unsupported version", `"schema_version": "1"`, `"schema_version": "2"`, "unsupported Common IR schema_version"},
		{"invalid diagnostic position", `"diagnostic_example": {` + "\n    " + `"kind": "parse_warning",` + "\n    " + `"message": "Example normalized diagnostic",` + "\n    " + `"line": 3`, `"diagnostic_example": {` + "\n    " + `"kind": "parse_warning",` + "\n    " + `"message": "Example normalized diagnostic",` + "\n    " + `"line": 0`, "diagnostic_example.line must be a positive integer"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := string(data)
			if tc.from != "" {
				mutated = strings.Replace(mutated, tc.from, tc.to, 1)
			}
			err := ValidateSerializedFixture([]byte(mutated))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got error %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestSerializedCommonIRFixtureAllowsUnknownOptionalFields(t *testing.T) {
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(data), `"schema_version": "1",`, `"schema_version": "1", "future_optional": true,`, 1)
	if err := ValidateSerializedFixture([]byte(mutated)); err != nil {
		t.Fatalf("unknown optional field should be accepted: %v", err)
	}
}
