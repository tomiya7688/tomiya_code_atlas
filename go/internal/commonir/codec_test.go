package commonir

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

const fixturePath = "../../../tests/fixtures/backend_conformance/serialized_common_ir_v1.json"

func TestPythonSerializedFixtureDecodesAndRoundTrips(t *testing.T) {
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	want, err := DecodeFixture(data)
	if err != nil {
		t.Fatalf("decode Python fixture: %v", err)
	}
	if want.CommonIR.Language != "python" || len(want.CommonIR.Entities) != 2 {
		t.Fatalf("unexpected Common IR contents: %#v", want.CommonIR)
	}
	if got := want.CommonIR.Entities[0]; got.Name != "run" || got.Line != 1 || got.EndLine != 2 || !slices.Equal(got.CallSequence, []string{"helper"}) {
		t.Fatalf("source entity/relation facts did not decode: %#v", got)
	}
	if len(want.LogicalOutput.CallGraph.Edges) != 1 || want.LogicalOutput.CallGraph.Edges[0].Caller != "run" || want.LogicalOutput.CallGraph.Edges[0].Callee != "helper" || want.LogicalOutput.CallGraph.Edges[0].CallType != "direct" {
		t.Fatalf("logical relation did not decode: %#v", want.LogicalOutput.CallGraph)
	}

	encoded, err := EncodeFixture(want)
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	got, err := DecodeFixture(encoded)
	if err != nil {
		t.Fatalf("decode encoded fixture: %v", err)
	}
	if got.SchemaVersion != want.SchemaVersion || got.Source != want.Source || got.CommonIR.Language != want.CommonIR.Language || !sameEntities(got.CommonIR.Entities, want.CommonIR.Entities) || !slices.Equal(got.CommonIR.Imports, want.CommonIR.Imports) || !slices.Equal(got.CommonIR.Diagnostics, want.CommonIR.Diagnostics) || !sameCallGraph(got.LogicalOutput.CallGraph, want.LogicalOutput.CallGraph) {
		t.Fatalf("round-trip changed Common IR meaning\n got: %#v\nwant: %#v", got, want)
	}
}

func TestHelperIRPayloadDecodesIntoCommonModel(t *testing.T) {
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := DecodeFixture(data)
	if err != nil {
		t.Fatal(err)
	}
	payload := Payload{SchemaVersion: SchemaVersion, Module: fixture.CommonIR}
	wire, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(wire)
	if err != nil {
		t.Fatalf("decode backend helper IR payload: %v", err)
	}
	if decoded.Language != "python" || decoded.Entities[0].Name != "run" {
		t.Fatalf("unexpected helper payload: %#v", decoded)
	}
	encoded, err := Encode(decoded)
	if err != nil {
		t.Fatal(err)
	}
	decodedAgain, err := Decode(encoded)
	if err != nil || decodedAgain.SchemaVersion != decoded.SchemaVersion || decodedAgain.Language != decoded.Language || !sameEntities(decodedAgain.Entities, decoded.Entities) || !slices.Equal(decodedAgain.Imports, decoded.Imports) || !slices.Equal(decodedAgain.Diagnostics, decoded.Diagnostics) {
		t.Fatalf("helper payload round-trip mismatch: err=%v got=%#v want=%#v", err, decodedAgain, decoded)
	}
}

func TestCommonIRPreservesTypedPropertyFacts(t *testing.T) {
	data := []byte(`{"schema_version":"1","language":"csharp","entities":[{"kind":"property","name":"Target","line":3,"end_line":3,"visibility":"public","type_name":"Transform"}],"imports":[],"diagnostics":[]}`)
	payload, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload.Entities) != 1 || payload.Entities[0].Kind != "property" || payload.Entities[0].TypeName == nil || *payload.Entities[0].TypeName != "Transform" {
		t.Fatalf("property type was not preserved: %#v", payload.Entities)
	}
	wire, err := Encode(payload)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Decode(wire)
	if err != nil || roundTrip.Entities[0].TypeName == nil || *roundTrip.Entities[0].TypeName != "Transform" {
		t.Fatalf("property type round-trip failed: err=%v payload=%s", err, wire)
	}
}

func sameEntities(left, right []Entity) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		a, b := left[index], right[index]
		if a.Kind != b.Kind || a.Name != b.Name || a.SourceLocation != b.SourceLocation || a.Indent != b.Indent || a.Visibility != b.Visibility || a.IsAsync != b.IsAsync || !sameString(a.Parent, b.Parent) || !sameString(a.Docstring, b.Docstring) || !sameString(a.DeclarationKind, b.DeclarationKind) || !sameString(a.SymbolID, b.SymbolID) || !sameString(a.ReturnType, b.ReturnType) || !sameString(a.TypeName, b.TypeName) || !slices.Equal(a.Parameters, b.Parameters) || !slices.Equal(a.Decorators, b.Decorators) || !slices.Equal(a.Calls, b.Calls) || !slices.Equal(a.CallSequence, b.CallSequence) || !slices.Equal(a.Bases, b.Bases) || !slices.Equal(a.TypeParameters, b.TypeParameters) || !slices.Equal(a.TypeConstraints, b.TypeConstraints) || !slices.Equal(a.ResolvedCalls, b.ResolvedCalls) || !slices.EqualFunc(a.ParameterTypes, b.ParameterTypes, func(left, right []string) bool { return slices.Equal(left, right) }) {
			return false
		}
	}
	return true
}

func sameString(left, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func sameCallGraph(left, right CallGraph) bool {
	return slices.Equal(left.Nodes, right.Nodes) && slices.Equal(left.Edges, right.Edges)
}

func TestDecodeRejectsUnsupportedVersionAndRequiredFieldErrors(t *testing.T) {
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
		{"unsupported version", `"schema_version": "1"`, `"schema_version": "2"`, "unsupported Common IR schema_version"},
		{"invalid source range", `"end_line": 2`, `"end_line": 0`, "invalid source range"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := strings.Replace(string(data), tc.from, tc.to, 1)
			_, err := DecodeFixture([]byte(mutated))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got error %v, want it to contain %q", err, tc.want)
			}
		})
	}
	_, err = Decode([]byte(`{"schema_version":"1","language":"python","imports":[],"diagnostics":[]}`))
	if err == nil || !strings.Contains(err.Error(), "common_ir.entities must be a JSON array") {
		t.Fatalf("missing helper entity array: got error %v", err)
	}
}
