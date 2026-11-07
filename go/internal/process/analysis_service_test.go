package process

import (
	"context"
	"os"
	"path/filepath"
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
		{operation: "sequence-diagram", contains: "sequenceDiagram"},
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

//	{
//	  責務: [TestValidateAnalysisRequestAllowsSiblingCommentOutput: 入力projectと重ならない兄弟folderをcomment出力先として許可する,]
//	  処理: [1: projectと兄弟output folderのpathを作る, 2: directory入力とcomments operationを検証する, 3: errorなくdirectory requestとして受理されることを確認する,]
//	  引数: [t: Go test runner,]
//	  戻り値: []
//	}
func TestValidateAnalysisRequestAllowsSiblingCommentOutput(t *testing.T) {
	parentPath := t.TempDir()
	projectPath := filepath.Join(parentPath, "project")
	outputPath := filepath.Join(parentPath, "atlas-output")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("create project folder: %v", err)
	}
	operations, isDirectory, err := validateAnalysisRequest(AnalysisRequest{
		SourcePath: projectPath, Operations: []string{"comments"}, OutputDirectory: outputPath,
	})
	if err != nil {
		t.Fatalf("validateAnalysisRequest() error = %v", err)
	}
	if !isDirectory {
		t.Fatal("validateAnalysisRequest() isDirectory = false, want true")
	}
	if _, found := operations["comments"]; !found {
		t.Fatal("validateAnalysisRequest() did not retain the comments operation")
	}
}

//	{
//	  責務: [TestAnalysisServiceRejectsCommentOutputInsideProject: 入出力が重なり生成済みcomment copyを再走査する配置を拒否する,]
//	  処理: [1: projectと既存のcomments copyを作る, 2: 同じfolder・配下folder・包含folder・symlink経由の出力を指定する, 3: parser起動前に拒否されることを確認する,]
//	  引数: [t: Go test runner,]
//	  戻り値: []
//	}
func TestAnalysisServiceRejectsCommentOutputInsideProject(t *testing.T) {
	projectPath := filepath.Join(t.TempDir(), "project")
	generatedCommentPath := filepath.Join(projectPath, "comments", "sample.py")
	if err := os.MkdirAll(filepath.Dir(generatedCommentPath), 0o755); err != nil {
		t.Fatalf("create generated comments folder: %v", err)
	}
	if err := os.WriteFile(generatedCommentPath, []byte("# generated comment copy\n"), 0o600); err != nil {
		t.Fatalf("create generated comment copy: %v", err)
	}
	testCases := []struct {
		name            string
		outputDirectory string
	}{
		{name: "same folder", outputDirectory: projectPath},
		{name: "nested folder", outputDirectory: filepath.Join(projectPath, "atlas-output")},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := (AnalysisService{}).Generate(context.Background(), AnalysisRequest{
				SourcePath: projectPath, Operations: []string{"comments"}, OutputDirectory: testCase.outputDirectory,
			})
			if err == nil || !strings.Contains(err.Error(), "入力project folderと重ならない") {
				t.Fatalf("Generate error = %v, want source/output overlap error", err)
			}
		})
	}
	t.Run("source nested under comments destination", func(t *testing.T) {
		outputPath := t.TempDir()
		nestedProjectPath := filepath.Join(outputPath, "comments", "project")
		if err := os.MkdirAll(nestedProjectPath, 0o755); err != nil {
			t.Fatalf("create project nested under output: %v", err)
		}
		_, err := (AnalysisService{}).Generate(context.Background(), AnalysisRequest{
			SourcePath: nestedProjectPath, Operations: []string{"comments"}, OutputDirectory: outputPath,
		})
		if err == nil || !strings.Contains(err.Error(), "入力project folderと重ならない") {
			t.Fatalf("Generate error = %v, want source/output overlap error", err)
		}
	})
	t.Run("comments symlink into project", func(t *testing.T) {
		outputPath := filepath.Join(t.TempDir(), "output")
		if err := os.MkdirAll(outputPath, 0o755); err != nil {
			t.Fatalf("create external output folder: %v", err)
		}
		if err := os.Symlink(projectPath, filepath.Join(outputPath, "comments")); err != nil {
			t.Skipf("directory symlink is unavailable: %v", err)
		}
		_, err := (AnalysisService{}).Generate(context.Background(), AnalysisRequest{
			SourcePath: projectPath, Operations: []string{"comments"}, OutputDirectory: outputPath,
		})
		if err == nil || !strings.Contains(err.Error(), "入力project folderと重ならない") {
			t.Fatalf("Generate error = %v, want source/output overlap error", err)
		}
	})
}

//	{
//	  責務: [TestRenderCallGraphKeepsSeriesAndSharedBoundaries: CLI call graph出力で系列分割後のrelationと共有境界を保持する]
//	  処理: [1: 三つの独立系列から共有nodeへ接続するCommon IRを作る, 2: call graph Markdownを生成する, 3: series/shared図とrelation参照の保持を確認する]
//	}
func TestRenderCallGraphKeepsSeriesAndSharedBoundaries(t *testing.T) {
	module := commonir.Module{
		Language: "go",
		Entities: []commonir.Entity{
			{Kind: "function", Name: "AlphaStart", Calls: []string{"AlphaMiddle"}},
			{Kind: "function", Name: "AlphaMiddle", Calls: []string{"AlphaWrite"}},
			{Kind: "function", Name: "AlphaWrite", Calls: []string{"SharedLogger"}},
			{Kind: "function", Name: "BetaStart", Calls: []string{"BetaMiddle"}},
			{Kind: "function", Name: "BetaMiddle", Calls: []string{"BetaWrite"}},
			{Kind: "function", Name: "BetaWrite", Calls: []string{"SharedLogger"}},
			{Kind: "function", Name: "GammaStart", Calls: []string{"GammaMiddle"}},
			{Kind: "function", Name: "GammaMiddle", Calls: []string{"GammaWrite"}},
			{Kind: "function", Name: "GammaWrite", Calls: []string{"SharedLogger"}},
			{Kind: "function", Name: "SharedLogger"},
		},
	}
	output, err := renderAnalysisOutput(module, "call-graph", "partition")
	if err != nil {
		t.Fatalf("renderAnalysisOutput returned error: %v", err)
	}
	if strings.Count(output, "## series_") != 3 || !strings.Contains(output, "## shared_") {
		t.Fatalf("call graph does not include three series and one shared diagram: %q", output)
	}
	if !strings.Contains(output, "続き:") {
		t.Fatalf("call graph does not include boundary references: %q", output)
	}
	for _, edge := range []string{
		"n_AlphaStart --> n_AlphaMiddle",
		"n_AlphaMiddle --> n_AlphaWrite",
		"n_BetaStart --> n_BetaMiddle",
		"n_BetaMiddle --> n_BetaWrite",
		"n_GammaStart --> n_GammaMiddle",
		"n_GammaMiddle --> n_GammaWrite",
	} {
		if !strings.Contains(output, edge) {
			t.Errorf("call graph does not preserve relation %q", edge)
		}
	}
	for _, edge := range []string{
		"n_AlphaWrite --> n_SharedLogger",
		"n_BetaWrite --> n_SharedLogger",
		"n_GammaWrite --> n_SharedLogger",
	} {
		if count := strings.Count(output, edge); count != 2 {
			t.Errorf("shared boundary relation %q appears %d times, want series and shared diagrams", edge, count)
		}
	}
}

//	{
//	  責務: [TestRenderCallGraphPreservesEmptyDiagram: nodeがない入力でも空のMermaid call graphを維持する]
//	  処理: [1: 空のCommon IRからcall graph Markdownを生成する, 2: 空のflowchart blockが出力されることを確認する]
//	}
func TestRenderCallGraphPreservesEmptyDiagram(t *testing.T) {
	output, err := renderAnalysisOutput(commonir.Module{Language: "go"}, "call-graph", "empty")
	if err != nil {
		t.Fatalf("renderAnalysisOutput returned error: %v", err)
	}
	if !strings.Contains(output, "```mermaid\nflowchart LR\n```") {
		t.Fatalf("empty call graph Mermaid block was not preserved: %q", output)
	}
}

//	{
//	  責務: [TestRenderCallGraphKeepsSingleDiagramFormat: 分割されないcall graphの従来Markdown形式を維持する]
//	  処理: [1: 小さなCommon IRからcall graphを生成する, 2: トップレベルMermaid blockと見出しの不在を確認する]
//	}
func TestRenderCallGraphKeepsSingleDiagramFormat(t *testing.T) {
	module := commonir.Module{
		Language: "go",
		Entities: []commonir.Entity{
			{Kind: "function", Name: "Main", Calls: []string{"Helper"}},
			{Kind: "function", Name: "Helper"},
		},
	}
	output, err := renderAnalysisOutput(module, "call-graph", "single")
	if err != nil {
		t.Fatalf("renderAnalysisOutput returned error: %v", err)
	}
	if !strings.Contains(output, "```mermaid\nflowchart LR\n") || strings.Contains(output, "## ") {
		t.Fatalf("single call graph did not preserve its top-level Mermaid block: %q", output)
	}
}
