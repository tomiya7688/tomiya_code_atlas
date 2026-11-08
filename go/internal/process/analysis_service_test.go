package process

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
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
		Objects:       []commonir.ObjectInstance{{Name: "widget", TypeName: "Widget"}},
		StateMachines: []commonir.StateMachine{{Owner: "Widget", StateVariable: "state", States: []string{"idle", "ready"}, Transitions: []commonir.StateTransition{{Source: "idle", Target: "ready"}}}},
		TimingFlows:   []commonir.TimingFlow{{Owner: "main", Events: []commonir.TimingEvent{{Order: 1, Kind: "await"}}}},
		InputEvents:   []commonir.InputEvent{{Actor: "user", Component: "app", Event: "start", Handler: "main"}},
	}
	moduleUnits := []analyzers.ModuleDependencyUnit{
		{Name: "frontend.main", CommonIR: commonir.Module{Imports: []string{"backend.models"}}},
		{Name: "backend.models", CommonIR: commonir.Module{}},
	}
	componentUnits := []analyzers.ComponentDependencyUnit{
		{Module: moduleUnits[0], Component: "frontend"},
		{Module: moduleUnits[1], Component: "backend"},
	}
	deploymentTopology := analyzers.ParseDockerfile("FROM golang:1.25", "Dockerfile")
	testCases := []struct {
		operation string
		contains  string
	}{
		{operation: "call-graph", contains: "flowchart LR"},
		{operation: "class-diagram", contains: "classDiagram"},
		{operation: "sequence-diagram", contains: "sequenceDiagram"},
		{operation: "object-diagram", contains: "widget : Widget"},
		{operation: "communication-diagram", contains: "flowchart LR"},
		{operation: "state-diagram", contains: "stateDiagram-v2"},
		{operation: "timing-chart", contains: "sequenceDiagram"},
		{operation: "use-case-diagram", contains: "flowchart LR"},
		{operation: "responsibility-table", contains: "| Class | Responsibility |"},
		{operation: "package-diagram", contains: "flowchart LR"},
		{operation: "component-diagram", contains: "flowchart LR"},
		{operation: "deployment-diagram", contains: "flowchart"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.operation, func(t *testing.T) {
			output, err := renderAnalysisOutput(module, testCase.operation, "sample", moduleUnits, componentUnits, deploymentTopology)
			if err != nil {
				t.Fatalf("renderAnalysisOutput returned error: %v", err)
			}
			if !strings.Contains(output, testCase.contains) {
				t.Fatalf("rendered output does not contain %q: %q", testCase.contains, output)
			}
			if testCase.operation != "responsibility-table" && !strings.Contains(output, "```mermaid") {
				t.Fatalf("rendered output does not contain a Mermaid block: %q", output)
			}
		})
	}
}

//	{
//	  責務: [TestDependencyUnitsForSourceUsesProjectRelativeNames: module・package・component識別子をsource pathから決定する]
//	}
func TestDependencyUnitsForSourceUsesProjectRelativeNames(t *testing.T) {
	projectRoot := t.TempDir()
	packageDirectory := filepath.Join(projectRoot, "frontend")
	if err := os.MkdirAll(packageDirectory, 0o755); err != nil {
		t.Fatalf("create package directory: %v", err)
	}
	sourcePath := filepath.Join(packageDirectory, "__init__.py")
	if err := os.WriteFile(sourcePath, []byte(""), 0o600); err != nil {
		t.Fatalf("create package source: %v", err)
	}
	moduleUnit, componentUnit, err := dependencyUnitsForSource(projectRoot, data.ProjectSourceFile{Path: sourcePath, Language: "python"}, commonir.Module{Language: "python"})
	if err != nil {
		t.Fatalf("dependencyUnitsForSource() error: %v", err)
	}
	if moduleUnit.Name != "frontend" || !moduleUnit.IsPackage || componentUnit.Component != "frontend" || componentUnit.Module.Name != moduleUnit.Name {
		t.Fatalf("dependency units = %#v, %#v", moduleUnit, componentUnit)
	}
}

//	{
//	  責務: [TestReadDeploymentTopologyMergesDockerfileAndComposeFacts: project内のDockerfile・Compose設定からconfiguration factを集める]
//	}
func TestReadDeploymentTopologyMergesDockerfileAndComposeFacts(t *testing.T) {
	projectRoot := t.TempDir()
	files := map[string]string{
		"Dockerfile":               "FROM golang:1.25\n",
		"compose.yaml":             "services:\n  api:\n    image: example/api\n  worker:\n    depends_on:\n      - api\n",
		".github/workflows/ci.yml": "name: CI\njobs: {}\n",
	}
	for relativePath, source := range files {
		path := filepath.Join(projectRoot, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create settings directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatalf("write settings file: %v", err)
		}
	}
	topology, err := readDeploymentTopology(projectRoot, true, data.NewProjectFileScanner())
	if err != nil {
		t.Fatalf("readDeploymentTopology() error: %v", err)
	}
	if len(topology.Nodes) != 4 || len(topology.Connections) != 2 {
		t.Fatalf("topology = %#v", topology)
	}
}

//	{
//	  責務: [TestGenerateCIWorkflowFromSelectedYAML: YAML file単独の選択からCI図Markdownを生成する]
//	}
func TestGenerateCIWorkflowFromSelectedYAML(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "ci.yml")
	source := "name: CI\non: [push, pull_request]\njobs:\n  build:\n    steps:\n      - name: Compile\n        run: go build ./...\n  test:\n    needs: build\n    steps:\n      - uses: actions/checkout@v4\n"
	if err := os.WriteFile(workflowPath, []byte(source), 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	outputDirectory := filepath.Join(root, "results")
	service := AnalysisService{}
	paths, err := service.Generate(context.Background(), AnalysisRequest{
		SourcePath: workflowPath, Operations: []string{"ci-graph"}, OutputDirectory: outputDirectory,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	wantPath := filepath.Join(outputDirectory, "ci_ci-graph.md")
	if !reflect.DeepEqual(paths, []string{wantPath}) {
		t.Fatalf("output paths = %#v, want %#v", paths, []string{wantPath})
	}
	output, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read CI graph: %v", err)
	}
	for _, marker := range []string{"# GitHub Actions / CI図", "## ci.yml", "ci_trigger[\"pull_request, push\"]", "ci_job_6275696c64 --> ci_job_74657374"} {
		if !strings.Contains(string(output), marker) {
			t.Errorf("CI graph does not contain %q: %s", marker, output)
		}
	}
}

//	{
//	  責務: [TestCompatibleOperationsReflectInputLanguageAndProjectFiles: 入力言語・workflow・設定fileに応じた操作だけを返す]
//	}
func TestCompatibleOperationsReflectInputLanguageAndProjectFiles(t *testing.T) {
	root := t.TempDir()
	languageFiles := map[string]string{
		"App.cs":   "class App {}\n",
		"app.cpp":  "class App {};\n",
		"app.gd":   "extends Node\n",
		"App.java": "class App {}\n",
		"app.go":   "package main\n",
		"app.py":   "def run(): pass\n",
	}
	for fileName, source := range languageFiles {
		path := filepath.Join(root, fileName)
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatalf("write source %s: %v", path, err)
		}
	}
	service := AnalysisService{}
	genericOperations := []string{
		"call-graph", "class-diagram", "sequence-diagram", "communication-diagram",
		"responsibility-table", "comments", "static-spec", "design-quality",
	}
	pythonSpecificOperations := []string{"object-diagram", "state-diagram", "timing-chart", "use-case-diagram"}
	for fileName := range languageFiles {
		path := filepath.Join(root, fileName)
		operations, err := service.CompatibleOperations(path)
		if err != nil {
			t.Fatalf("CompatibleOperations(%s) error: %v", fileName, err)
		}
		for _, operation := range genericOperations {
			if !slices.Contains(operations, operation) {
				t.Errorf("%s operations = %#v; expected generic operation %s", fileName, operations, operation)
			}
		}
		for _, operation := range pythonSpecificOperations {
			if slices.Contains(operations, operation) != (filepath.Ext(fileName) == ".py") {
				t.Errorf("%s operation %s has incorrect language compatibility: %#v", fileName, operation, operations)
			}
		}
	}
	csharpPath := filepath.Join(root, "App.cs")
	_, err := service.Generate(context.Background(), AnalysisRequest{
		SourcePath: csharpPath, Operations: []string{"timing-chart"}, OutputDirectory: filepath.Join(root, "output"),
	})
	if err == nil || !strings.Contains(err.Error(), "選択した入力では実行できない操作") {
		t.Fatalf("Generate(C# timing-chart) error = %v, want incompatible operation error", err)
	}
	projectOperations, err := service.CompatibleOperations(root)
	if err != nil {
		t.Fatalf("CompatibleOperations(project) error: %v", err)
	}
	if !slices.Contains(projectOperations, "package-diagram") || !slices.Contains(projectOperations, "object-diagram") {
		t.Fatalf("project operations = %#v; expected folder and Python outputs", projectOperations)
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
			output, err := renderAnalysisOutput(module, testCase.operation, "sample", nil, nil, analyzers.DeploymentTopology{})
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
	output, err := renderAnalysisOutput(module, "call-graph", "partition", nil, nil, analyzers.DeploymentTopology{})
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
	output, err := renderAnalysisOutput(commonir.Module{Language: "go"}, "call-graph", "empty", nil, nil, analyzers.DeploymentTopology{})
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
	output, err := renderAnalysisOutput(module, "call-graph", "single", nil, nil, analyzers.DeploymentTopology{})
	if err != nil {
		t.Fatalf("renderAnalysisOutput returned error: %v", err)
	}
	if !strings.Contains(output, "```mermaid\nflowchart LR\n") || strings.Contains(output, "## ") {
		t.Fatalf("single call graph did not preserve its top-level Mermaid block: %q", output)
	}
}
