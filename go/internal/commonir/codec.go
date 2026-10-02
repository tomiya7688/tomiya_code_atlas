package commonir

import (
	"encoding/json"
	"fmt"
)

// Decode parses a helper IR payload and validates its version and required v1 fields.
func Decode(data []byte) (Payload, error) {
	var payload Payload
	if err := json.Unmarshal(data, &payload); err != nil {
		return Payload{}, fmt.Errorf("decode Common IR payload: %w", err)
	}
	if payload.SchemaVersion == "" {
		return Payload{}, fmt.Errorf("payload.schema_version is required")
	}
	if err := validateVersion(payload.SchemaVersion); err != nil {
		return Payload{}, err
	}
	if err := validateModule(payload.Module, "common_ir"); err != nil {
		return Payload{}, err
	}
	return payload, nil
}

// Encode validates and serializes a helper IR payload using the v1 wire shape.
func Encode(payload Payload) ([]byte, error) {
	if err := validateVersion(payload.SchemaVersion); err != nil {
		return nil, err
	}
	if err := validateModule(payload.Module, "common_ir"); err != nil {
		return nil, err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode Common IR payload: %w", err)
	}
	return data, nil
}

// DecodeFixture parses the Python-produced serialized golden, validating the
// nested Common IR as well as the diagnostic and logical relation examples.
func DecodeFixture(data []byte) (Fixture, error) {
	var fixture Fixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return Fixture{}, fmt.Errorf("decode serialized Common IR fixture: %w", err)
	}
	if fixture.SchemaVersion == "" {
		return Fixture{}, fmt.Errorf("fixture.schema_version is required")
	}
	if err := validateVersion(fixture.SchemaVersion); err != nil {
		return Fixture{}, err
	}
	if err := validateModule(fixture.CommonIR, "common_ir"); err != nil {
		return Fixture{}, err
	}
	if fixture.DiagnosticExample.Kind == "" || fixture.DiagnosticExample.Message == "" {
		return Fixture{}, fmt.Errorf("diagnostic_example.kind and diagnostic_example.message are required")
	}
	if fixture.DiagnosticExample.Line == nil || *fixture.DiagnosticExample.Line < 1 {
		return Fixture{}, fmt.Errorf("diagnostic_example.line must be a positive integer")
	}
	if fixture.LogicalOutput.CallGraph.Nodes == nil || fixture.LogicalOutput.CallGraph.Edges == nil {
		return Fixture{}, fmt.Errorf("logical_output.call_graph.nodes and edges are required arrays")
	}
	for index, relation := range fixture.LogicalOutput.CallGraph.Edges {
		if relation.Caller == "" || relation.Callee == "" || relation.CallType == "" {
			return Fixture{}, fmt.Errorf("logical_output.call_graph.edges[%d] requires caller, callee, and call_type", index)
		}
	}
	return fixture, nil
}

// EncodeFixture validates and serializes the versioned conformance fixture.
func EncodeFixture(fixture Fixture) ([]byte, error) {
	if err := validateVersion(fixture.SchemaVersion); err != nil {
		return nil, err
	}
	if err := validateModule(fixture.CommonIR, "common_ir"); err != nil {
		return nil, err
	}
	if fixture.DiagnosticExample.Kind == "" || fixture.DiagnosticExample.Message == "" || fixture.DiagnosticExample.Line == nil || *fixture.DiagnosticExample.Line < 1 {
		return nil, fmt.Errorf("diagnostic_example requires kind, message, and a positive line")
	}
	if fixture.LogicalOutput.CallGraph.Nodes == nil || fixture.LogicalOutput.CallGraph.Edges == nil {
		return nil, fmt.Errorf("logical_output.call_graph.nodes and edges are required arrays")
	}
	data, err := json.Marshal(fixture)
	if err != nil {
		return nil, fmt.Errorf("encode serialized Common IR fixture: %w", err)
	}
	return data, nil
}

func validateVersion(version string) error {
	if version != SchemaVersion {
		return fmt.Errorf("unsupported Common IR schema_version %q (supported: %q)", version, SchemaVersion)
	}
	return nil
}

func validateModule(module Module, where string) error {
	if module.Language == "" {
		return fmt.Errorf("%s.language is required", where)
	}
	if module.Entities == nil {
		return fmt.Errorf("%s.entities must be a JSON array", where)
	}
	if module.Imports == nil {
		return fmt.Errorf("%s.imports must be a JSON array", where)
	}
	if module.Diagnostics == nil {
		return fmt.Errorf("%s.diagnostics must be a JSON array", where)
	}
	for index, entity := range module.Entities {
		if entity.Kind == "" || entity.Name == "" {
			return fmt.Errorf("%s.entities[%d].kind and name are required", where, index)
		}
		if entity.Line < 1 || entity.EndLine < entity.Line {
			return fmt.Errorf("%s.entities[%d] has invalid source range line=%d end_line=%d", where, index, entity.Line, entity.EndLine)
		}
	}
	for index, diagnostic := range module.Diagnostics {
		if diagnostic.Kind == "" || diagnostic.Message == "" {
			return fmt.Errorf("%s.diagnostics[%d].kind and message are required", where, index)
		}
		if diagnostic.Line != nil && *diagnostic.Line < 1 {
			return fmt.Errorf("%s.diagnostics[%d].line must be a positive integer when present", where, index)
		}
	}
	return nil
}
