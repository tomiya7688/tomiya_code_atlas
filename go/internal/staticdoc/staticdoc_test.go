package staticdoc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/pythonadapter"
)

func TestRenderStaticSpecificationIncludesResolvedLocalCalls(t *testing.T) {
	source := `"""Summary."""
class Base: pass
class Worker(Base):
    def run(self, value: int) -> str:
        return self.format(value)
    def format(self, value: int) -> str:
        return str(value)
`
	payload := pythonadapter.Parse(source)
	got := Render(payload.Module, "sample.py")
	for _, want := range []string{"# sample.py", "class: `Base`", "class: `Worker`", "### method: `Worker.run`", "run(value: int) -> str", "同一ファイル内で解決した呼び出し先: `Worker.format`", "呼び出し元: `Worker.run`"} {
		if !strings.Contains(got, want) {
			t.Errorf("render missing %q:\n%s", want, got)
		}
	}
}

func TestEmptyModuleHasLocalizedNotice(t *testing.T) {
	got := Render(pythonadapter.Parse("\"\"\"empty\"\"\"\n").Module, "empty.py")
	if !strings.Contains(got, "宣言されたクラス、関数、メソッドはありません。") {
		t.Fatalf("unexpected empty-module document: %s", got)
	}
}

func TestGoldenMatchesPythonStaticSpecification(t *testing.T) {
	fixture := filepath.Clean("../../../tests/fixtures/static_specification/sample.py")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	got := Render(pythonadapter.Parse(string(data)).Module, "sample.py")
	want, err := os.ReadFile("../../../tests/fixtures/static_specification/sample-spec.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("static specification differs from golden\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
