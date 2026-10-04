package staticdoc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

func TestRenderStaticSpecificationIncludesResolvedLocalCalls(t *testing.T) {
	got := Render(commonir.Module{
		Language: "python",
		Entities: []commonir.Entity{
			{Kind: "class", Name: "Base", SourceLocation: commonir.SourceLocation{Line: 2, EndLine: 2}},
			{Kind: "class", Name: "Worker", SourceLocation: commonir.SourceLocation{Line: 3, EndLine: 7}, Bases: []string{"Base"}},
			{Kind: "method", Name: "run", SourceLocation: commonir.SourceLocation{Line: 4, EndLine: 5}, Parent: stringPointer("Worker"), Parameters: []string{"self", "value"}, ParameterTypes: [][]string{{"value", "int"}}, ReturnType: stringPointer("str"), Calls: []string{"self.format"}, CallSequence: []string{"self.format"}, ResolvedCalls: []string{"Worker.format"}},
			{Kind: "method", Name: "format", SourceLocation: commonir.SourceLocation{Line: 6, EndLine: 7}, Parent: stringPointer("Worker"), Parameters: []string{"self", "value"}, ParameterTypes: [][]string{{"value", "int"}}, ReturnType: stringPointer("str"), Calls: []string{"str"}, CallSequence: []string{"str"}},
		},
		Imports: []string{}, Diagnostics: []commonir.Diagnostic{},
	}, "sample.py")
	for _, want := range []string{"# sample.py", "class: `Base`", "class: `Worker`", "### method: `Worker.run`", "run(value: int) -> str", "同一ファイル内で解決した呼び出し先: `Worker.format`", "呼び出し元: `Worker.run`"} {
		if !strings.Contains(got, want) {
			t.Errorf("render missing %q:\n%s", want, got)
		}
	}
}

func TestEmptyModuleHasLocalizedNotice(t *testing.T) {
	got := Render(commonir.Module{Language: "python", Entities: []commonir.Entity{}, Imports: []string{}, Diagnostics: []commonir.Diagnostic{}}, "empty.py")
	if !strings.Contains(got, "宣言されたクラス、関数、メソッドはありません。") {
		t.Fatalf("unexpected empty-module document: %s", got)
	}
}

func TestGoldenMatchesPythonStaticSpecification(t *testing.T) {
	got := Render(loadPythonFixture(t), "sample.py")
	want, err := os.ReadFile("../../../tests/fixtures/static_specification/sample-spec.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("static specification differs from golden\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func stringPointer(value string) *string { return &value }

func loadPythonFixture(t *testing.T) commonir.Module {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean("../../../tests/fixtures/static_specification/sample-common-ir.json"))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := commonir.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return payload.Module
}
