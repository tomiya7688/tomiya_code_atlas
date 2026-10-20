package process

import (
	"context"
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

//	{
//	  責務: [TestRenderDiagramProducesMermaidForSelectedDiagrams: Common IRから選択operation別のMermaid Markdownを生成する]
//	}
func TestRenderDiagramProducesMermaidForSelectedDiagrams(t *testing.T) {
	module := commonir.Module{
		Language: "python",
		Entities: []commonir.Entity{
			{Kind: "function", Name: "main", Calls: []string{"helper"}},
			{Kind: "function", Name: "helper"},
			{Kind: "class", Name: "Widget"},
		},
	}
	testCases := []struct {
		operation string
		contains  string
	}{
		{operation: "call-graph", contains: "flowchart LR"},
		{operation: "class-diagram", contains: "classDiagram"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.operation, func(t *testing.T) {
			output, err := renderDiagram(module, testCase.operation)
			if err != nil {
				t.Fatalf("renderDiagram returned error: %v", err)
			}
			if !strings.Contains(output, testCase.contains) || !strings.Contains(output, "```mermaid") {
				t.Fatalf("rendered output does not contain %q Mermaid block: %q", testCase.contains, output)
			}
		})
	}
}

//	{
//	  責務: [TestDiagramServiceRejectsDuplicateOrUnsupportedOperations: 不正operationをparse前に拒否する]
//	}
func TestDiagramServiceRejectsDuplicateOrUnsupportedOperations(t *testing.T) {
	testCases := []struct {
		name       string
		operations []string
		wantError  string
	}{
		{name: "duplicate operation", operations: []string{"call-graph", "call-graph"}, wantError: "重複"},
		{name: "unsupported operation", operations: []string{"deployment"}, wantError: "未対応"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			service := DiagramService{}
			_, err := service.Generate(context.Background(), DiagramRequest{
				SourcePath: "unused.py", Operations: testCase.operations, OutputDirectory: "unused-output",
			})
			if err == nil || !strings.Contains(err.Error(), testCase.wantError) {
				t.Fatalf("Generate error = %v, want %q", err, testCase.wantError)
			}
		})
	}
}
