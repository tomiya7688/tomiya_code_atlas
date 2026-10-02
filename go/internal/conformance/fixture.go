// Package conformance validates the versioned, renderer-neutral Common IR golden.
package conformance

import (
	"encoding/json"
	"fmt"
)

// ValidateSerializedFixture checks required Common IR and logical-output fields.
// Unknown fields are intentionally accepted so optional additions remain compatible.
func ValidateSerializedFixture(data []byte) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("decode conformance fixture JSON: %w", err)
	}
	version, err := stringField(root, "schema_version", "fixture")
	if err != nil {
		return err
	}
	if version != "1" {
		return fmt.Errorf("unsupported Common IR schema_version %q (supported: %q)", version, "1")
	}

	module, err := objectField(root, "common_ir", "fixture")
	if err != nil {
		return err
	}
	if _, err := stringField(module, "language", "common_ir"); err != nil {
		return err
	}
	entities, err := arrayField(module, "entities", "common_ir")
	if err != nil {
		return err
	}
	for index, raw := range entities {
		entity, err := decodeObject(raw, fmt.Sprintf("common_ir.entities[%d]", index))
		if err != nil {
			return err
		}
		where := fmt.Sprintf("common_ir.entities[%d]", index)
		if _, err := stringField(entity, "kind", where); err != nil {
			return err
		}
		if _, err := stringField(entity, "name", where); err != nil {
			return err
		}
		line, err := intField(entity, "line", where)
		if err != nil {
			return err
		}
		endLine, err := intField(entity, "end_line", where)
		if err != nil {
			return err
		}
		if line < 1 || endLine < line {
			return fmt.Errorf("%s has invalid source range line=%d end_line=%d", where, line, endLine)
		}
	}
	if _, err := arrayField(module, "imports", "common_ir"); err != nil {
		return err
	}
	diagnostics, err := arrayField(module, "diagnostics", "common_ir")
	if err != nil {
		return err
	}
	for index, raw := range diagnostics {
		diagnostic, err := decodeObject(raw, fmt.Sprintf("common_ir.diagnostics[%d]", index))
		if err != nil {
			return err
		}
		where := fmt.Sprintf("common_ir.diagnostics[%d]", index)
		if _, err := stringField(diagnostic, "kind", where); err != nil {
			return err
		}
		if _, err := stringField(diagnostic, "message", where); err != nil {
			return err
		}
		if rawLine, ok := diagnostic["line"]; ok {
			var line int
			if err := json.Unmarshal(rawLine, &line); err != nil || line < 1 {
				return fmt.Errorf("%s.line must be a positive integer when present", where)
			}
		}
	}
	diagnosticExample, err := objectField(root, "diagnostic_example", "fixture")
	if err != nil {
		return err
	}
	if _, err := stringField(diagnosticExample, "kind", "diagnostic_example"); err != nil {
		return err
	}
	if _, err := stringField(diagnosticExample, "message", "diagnostic_example"); err != nil {
		return err
	}
	if line, err := intField(diagnosticExample, "line", "diagnostic_example"); err != nil || line < 1 {
		return fmt.Errorf("diagnostic_example.line must be a positive integer")
	}

	logicalOutput, err := objectField(root, "logical_output", "fixture")
	if err != nil {
		return err
	}
	callGraph, err := objectField(logicalOutput, "call_graph", "logical_output")
	if err != nil {
		return err
	}
	if _, err := arrayField(callGraph, "nodes", "logical_output.call_graph"); err != nil {
		return err
	}
	edges, err := arrayField(callGraph, "edges", "logical_output.call_graph")
	if err != nil {
		return err
	}
	for index, raw := range edges {
		edge, err := decodeObject(raw, fmt.Sprintf("logical_output.call_graph.edges[%d]", index))
		if err != nil {
			return err
		}
		where := fmt.Sprintf("logical_output.call_graph.edges[%d]", index)
		for _, field := range []string{"caller", "callee", "call_type"} {
			if _, err := stringField(edge, field, where); err != nil {
				return err
			}
		}
	}
	return nil
}

func decodeObject(raw json.RawMessage, where string) (map[string]json.RawMessage, error) {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, fmt.Errorf("%s must be a JSON object", where)
	}
	return value, nil
}

func objectField(object map[string]json.RawMessage, field, where string) (map[string]json.RawMessage, error) {
	raw, ok := object[field]
	if !ok {
		return nil, fmt.Errorf("%s.%s is required", where, field)
	}
	return decodeObject(raw, where+"."+field)
}

func arrayField(object map[string]json.RawMessage, field, where string) ([]json.RawMessage, error) {
	raw, ok := object[field]
	if !ok {
		return nil, fmt.Errorf("%s.%s is required", where, field)
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return nil, fmt.Errorf("%s.%s must be a JSON array", where, field)
	}
	return values, nil
}

func stringField(object map[string]json.RawMessage, field, where string) (string, error) {
	raw, ok := object[field]
	if !ok {
		return "", fmt.Errorf("%s.%s is required", where, field)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || value == "" {
		return "", fmt.Errorf("%s.%s must be a non-empty string", where, field)
	}
	return value, nil
}

func intField(object map[string]json.RawMessage, field, where string) (int, error) {
	raw, ok := object[field]
	if !ok {
		return 0, fmt.Errorf("%s.%s is required", where, field)
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("%s.%s must be an integer", where, field)
	}
	return value, nil
}
