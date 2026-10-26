package ui

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
)

//	{
//	  責務: [
//	    fakeDiagramGenerator: GUIから受けたrequestを記録し、テスト用pathまたはerrorを返す,
//	  ],
//	}
type fakeDiagramGenerator struct {
	request process.AnalysisRequest
	paths   []string
	err     error
}

//	{
//	  責務: [Generate: GUIが渡した要求を記録し、指定された結果またはエラーを返す,],
//	  引数: [request: 画面から渡された解析要求,],
//	  戻り値: [paths: テスト用の出力先, error: 指定された失敗,],
//	}
func (generator *fakeDiagramGenerator) Generate(_ context.Context, request process.AnalysisRequest) ([]string, error) {
	generator.request = request
	return generator.paths, generator.err
}

//	{
//	  責務: [TestAnalyzePassesMultipleDiagramSelections: 複数選択した図と出力先が一度の解析要求として渡ることを検証する,],
//	}
func TestAnalyzePassesMultipleDiagramSelections(t *testing.T) {
	generator := &fakeDiagramGenerator{paths: []string{"out/sample_call-graph.md", "out/sample_class-diagram.md"}}
	application := &Application{generator: generator}
	result, err := application.Analyze("src/sample.py", []string{"call-graph", "class-diagram"}, "out")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if generator.request.SourcePath != "src/sample.py" || generator.request.OutputDirectory != "out" || !reflect.DeepEqual(generator.request.Operations, []string{"call-graph", "class-diagram"}) {
		t.Fatalf("request = %#v", generator.request)
	}
	if !reflect.DeepEqual(result.OutputPaths, generator.paths) || !strings.Contains(result.Message, "2件") {
		t.Fatalf("result = %#v", result)
	}
}

//	{
//	  責務: [TestAnalyzeRejectsInvalidSelectionAndReturnsProcessError: 不正な選択の拒否と解析失敗の伝達を検証する,],
//	}
func TestAnalyzeRejectsInvalidSelectionAndReturnsProcessError(t *testing.T) {
	for _, testCase := range []struct {
		name, source, output string
		operations           []string
		want                 string
	}{
		{name: "missing source", output: "out", operations: []string{"call-graph"}, want: "ファイルまたはプロジェクト"},
		{name: "missing operation", source: "src", output: "out", want: "一つ以上"},
		{name: "unsupported operation", source: "src", output: "out", operations: []string{"unknown"}, want: "未対応"},
		{name: "duplicate operation", source: "src", output: "out", operations: []string{"call-graph", "call-graph"}, want: "重複"},
		{name: "missing output", source: "src", operations: []string{"call-graph"}, want: "保存フォルダー"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			application := &Application{generator: &fakeDiagramGenerator{}}
			if _, err := application.Analyze(testCase.source, testCase.operations, testCase.output); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}
	processError := errors.New("parserが失敗しました")
	application := &Application{generator: &fakeDiagramGenerator{err: processError}}
	if _, err := application.Analyze("src", []string{"call-graph"}, "out"); !errors.Is(err, processError) {
		t.Fatalf("process error = %v", err)
	}
}

//	{
//	  責務: [TestSourceFileFilterMatchesSupportedLanguages: 各ソース言語の選択肢とscannerの判定が一致することを検証する,],
//	}
func TestSourceFileFilterMatchesSupportedLanguages(t *testing.T) {
	scanner := data.NewProjectFileScanner()
	patterns := strings.Split(sourceFileFilterPattern, ";")
	sourceLanguages := []string{"python", "gdscript", "csharp", "cpp", "java", "go"}
	for _, extension := range []string{".py", ".gd", ".cs", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".hxx", ".java", ".go", ".yml", ".yaml"} {
		language := scanner.DetectLanguage("sample" + extension)
		supported := slices.Contains(sourceLanguages, language)
		selectable := slices.Contains(patterns, "*"+extension)
		if selectable != supported {
			t.Errorf("extension %q: selectable = %t, supported = %t", extension, selectable, supported)
		}
		if supported && scanner.DetectLanguage("sample"+strings.ToUpper(extension)) != language {
			t.Errorf("uppercase extension %q has a different language", extension)
		}
	}
}
