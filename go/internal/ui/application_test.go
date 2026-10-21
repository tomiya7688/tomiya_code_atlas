package ui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
)

//	{
//	  責務: [
//	    fakeDiagramGenerator: GUIから受けたrequestを記録し、テスト用pathまたはerrorを返す,
//	  ],
//	}
type fakeDiagramGenerator struct {
	request process.DiagramRequest
	paths   []string
	err     error
}

func (generator *fakeDiagramGenerator) Generate(_ context.Context, request process.DiagramRequest) ([]string, error) {
	generator.request = request
	return generator.paths, generator.err
}

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
