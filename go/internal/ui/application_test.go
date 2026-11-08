package ui

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
)

//	{
//	  責務: [
//	    fakeAnalysisGenerator: GUIから受けたrequestを記録し、テスト用pathまたはerrorを返す,
//	  ],
//	}
type fakeAnalysisGenerator struct {
	request process.AnalysisRequest
	paths   []string
	err     error
}

//	{
//	  責務: [TestRemoveSmokeProfileRetriesUntilReleased: WebView2 profile削除を成功まで再試行する,],
//	}
func TestRemoveSmokeProfileRetriesUntilReleased(t *testing.T) {
	attempts := 0
	waits := 0
	err := removeSmokeProfileWithRetry("profile", func(string) error {
		attempts++
		if attempts < 3 {
			return errors.New("profile is still in use")
		}
		return nil
	}, func(delay time.Duration) {
		waits++
		if delay != smokeProfileCleanupInterval {
			t.Errorf("retry delay = %s, want %s", delay, smokeProfileCleanupInterval)
		}
	})
	if err != nil {
		t.Fatalf("removeSmokeProfileWithRetry returned error: %v", err)
	}
	if attempts != 3 || waits != 2 {
		t.Fatalf("attempts = %d, waits = %d; want 3 attempts and 2 waits", attempts, waits)
	}
}

//	{
//	  責務: [TestRemoveSmokeProfileReportsPersistentLock: profile削除が再試行上限後も失敗した場合にerrorを返す,],
//	}
func TestRemoveSmokeProfileReportsPersistentLock(t *testing.T) {
	cleanupError := errors.New("profile is still in use")
	attempts := 0
	err := removeSmokeProfileWithRetry("profile", func(string) error {
		attempts++
		return cleanupError
	}, func(time.Duration) {})
	if !errors.Is(err, cleanupError) {
		t.Fatalf("removeSmokeProfileWithRetry error = %v, want %v", err, cleanupError)
	}
	if attempts != smokeProfileCleanupAttempts {
		t.Fatalf("attempts = %d, want %d", attempts, smokeProfileCleanupAttempts)
	}
}

//	{
//	  責務: [Generate: GUIが渡した要求を記録し、指定された結果またはエラーを返す,],
//	  引数: [request: 画面から渡された解析要求,],
//	  戻り値: [paths: テスト用の出力先, error: 指定された失敗,],
//	}
func (generator *fakeAnalysisGenerator) Generate(_ context.Context, request process.AnalysisRequest) ([]string, error) {
	generator.request = request
	return generator.paths, generator.err
}

//	{
//	  責務: [TestAnalyzePassesAllSupportedOperations: 対応する解析操作一覧と出力先が一度の解析要求として渡ることを検証する,],
//	}
func TestAnalyzePassesAllSupportedOperations(t *testing.T) {
	operations := process.SupportedOperationIDs()
	generator := &fakeAnalysisGenerator{paths: []string{
		"out/sample_call-graph.md",
		"out/sample_class-diagram.md",
		"out/sample_sequence-diagram.md",
		"out/sample_object-diagram.md",
		"out/sample_communication-diagram.md",
		"out/sample_state-diagram.md",
		"out/sample_timing-chart.md",
		"out/sample_use-case-diagram.md",
		"out/sample_responsibility-table.md",
		"out/sample_package-diagram.md",
		"out/sample_component-diagram.md",
		"out/sample_deployment-diagram.md",
		"out/sample_ci-graph.md",
		"out/comments/sample_commented.py",
		"out/sample_static-spec.md",
		"out/sample_design-quality.md",
	}}
	application := &Application{generator: generator}
	result, err := application.Analyze("src/sample.py", operations, "out")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if generator.request.SourcePath != "src/sample.py" || generator.request.OutputDirectory != "out" || !reflect.DeepEqual(generator.request.Operations, operations) {
		t.Fatalf("request = %#v", generator.request)
	}
	if !reflect.DeepEqual(result.OutputPaths, generator.paths) || !strings.Contains(result.Message, "16件") || !strings.Contains(result.Message, "成果物") {
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
			application := &Application{generator: &fakeAnalysisGenerator{}}
			if _, err := application.Analyze(testCase.source, testCase.operations, testCase.output); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}
	processError := errors.New("parserが失敗しました")
	application := &Application{generator: &fakeAnalysisGenerator{err: processError}}
	if _, err := application.Analyze("src", []string{"call-graph"}, "out"); !errors.Is(err, processError) {
		t.Fatalf("process error = %v", err)
	}
}

//	{
//	  責務: [TestSourceFileFilterMatchesSupportedLanguages: 各ソース言語の選択肢とscannerの判定が一致することを検証する,],
//	}
func TestSourceFileFilterMatchesSupportedLanguagesAndCIWorkflows(t *testing.T) {
	scanner := data.NewProjectFileScanner()
	patterns := strings.Split(sourceFileFilterPattern, ";")
	sourceLanguages := []string{"python", "gdscript", "csharp", "cpp", "java", "go"}
	for _, extension := range []string{".py", ".gd", ".cs", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".hxx", ".java", ".go"} {
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
	for _, extension := range []string{".yml", ".yaml"} {
		if scanner.DetectLanguage("workflow"+extension) != "yaml" || !slices.Contains(patterns, "*"+extension) {
			t.Errorf("workflow extension %q must be selectable as CI input without being a source language", extension)
		}
	}
}
