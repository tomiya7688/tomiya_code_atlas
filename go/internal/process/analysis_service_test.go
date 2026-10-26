package process

import (
	"context"
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

//	{
//	  責務: [TestRenderAnalysisOutputProducesSelectedDiagrams: Common IRから選択diagram別のMermaid Markdownを生成する]
//	}
func TestRenderAnalysisOutputProducesSelectedDiagrams(t *testing.T) {
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
			output, err := renderAnalysisOutput(module, testCase.operation, "sample")
			if err != nil {
				t.Fatalf("renderAnalysisOutput returned error: %v", err)
			}
			if !strings.Contains(output, testCase.contains) || !strings.Contains(output, "```mermaid") {
				t.Fatalf("rendered output does not contain %q Mermaid block: %q", testCase.contains, output)
			}
		})
	}
}

//	{
//	  責務: [TestRenderAnalysisOutputProducesStaticSpecificationAndDesignQualityDocuments: Common IRから静的仕様書・設計品質のMarkdownを生成する]
//	}
func TestRenderAnalysisOutputProducesStaticSpecificationAndDesignQualityDocuments(t *testing.T) {
	module := commonir.Module{
		Language: "go",
		Entities: []commonir.Entity{
			{Kind: "class", Name: "Runner", SourceLocation: commonir.SourceLocation{Line: 1, EndLine: 4}},
			{Kind: "function", Name: "Run", Calls: []string{"execute"}, SourceLocation: commonir.SourceLocation{Line: 6, EndLine: 8}},
			{Kind: "function", Name: "execute", SourceLocation: commonir.SourceLocation{Line: 10, EndLine: 12}},
		},
	}
	testCases := []struct {
		operation string
		contains  []string
	}{
		{operation: "static-spec", contains: []string{"# sample", "## 宣言一覧", "Runner"}},
		{operation: "design-quality", contains: []string{"# 設計品質レポート", "## 指標", "## 指摘"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.operation, func(t *testing.T) {
			output, err := renderAnalysisOutput(module, testCase.operation, "sample")
			if err != nil {
				t.Fatalf("renderAnalysisOutput returned error: %v", err)
			}
			for _, marker := range testCase.contains {
				if !strings.Contains(output, marker) {
					t.Errorf("%s output does not contain %q: %q", testCase.operation, marker, output)
				}
			}
		})
	}
}

//	{
//	  責務: [TestAppendModuleCombinesFactsAndMarksMixedLanguages: 複数sourceのCommon IR factを保持し言語混在を表す]
//	  処理: [1: 異なるlanguageのmoduleを追加する, 2: collectionの件数と言語IDを検証する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestAppendModuleCombinesFactsAndMarksMixedLanguages(t *testing.T) {
	combined := commonir.Module{}
	appendModule(&combined, commonir.Module{
		Language: "go", Entities: []commonir.Entity{{Name: "Run"}},
		Diagnostics: []commonir.Diagnostic{{Kind: "warning"}}, Imports: []string{"fmt"},
	})
	appendModule(&combined, commonir.Module{
		Language: "python", Entities: []commonir.Entity{{Name: "start"}},
		Objects: []commonir.ObjectInstance{{Name: "client"}}, StateMachines: []commonir.StateMachine{{Owner: "app"}},
		TimingFlows: []commonir.TimingFlow{{Owner: "start"}}, InputEvents: []commonir.InputEvent{{Handler: "start"}},
		Signals: []commonir.Signal{{Name: "changed"}}, Imports: []string{"pathlib"},
	})
	if combined.Language != "mixed" || len(combined.Entities) != 2 || len(combined.Objects) != 1 || len(combined.StateMachines) != 1 || len(combined.TimingFlows) != 1 || len(combined.InputEvents) != 1 || len(combined.Signals) != 1 || len(combined.Diagnostics) != 1 || len(combined.Imports) != 2 {
		t.Fatalf("combined module facts were not preserved: %#v", combined)
	}
}

//	{
//	  責務: [TestAnalysisServiceRejectsDuplicateOrUnsupportedOperations: 不正operationをparse前に拒否する]
//	}
func TestAnalysisServiceRejectsDuplicateOrUnsupportedOperations(t *testing.T) {
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
			service := AnalysisService{}
			_, err := service.Generate(context.Background(), AnalysisRequest{
				SourcePath: "unused.py", Operations: testCase.operations, OutputDirectory: "unused-output",
			})
			if err == nil || !strings.Contains(err.Error(), testCase.wantError) {
				t.Fatalf("Generate error = %v, want %q", err, testCase.wantError)
			}
		})
	}
}
