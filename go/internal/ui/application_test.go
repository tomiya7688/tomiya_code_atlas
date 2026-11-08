package ui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
	"golang.org/x/net/html"
)

//	{
//	  責務: [TestEmbeddedUIListsEverySupportedOperation: GUIの成果物選択肢とCLI operation registryの一致を確認する,]
//	}
func TestEmbeddedUIListsEverySupportedOperation(t *testing.T) {
	guiHTML, err := embeddedWebFiles.ReadFile("web/index.html")
	if err != nil {
		t.Fatalf("read embedded GUI: %v", err)
	}
	document, err := html.Parse(bytes.NewReader(guiHTML))
	if err != nil {
		t.Fatalf("parse embedded GUI: %v", err)
	}
	operationIDs := process.SupportedOperationIDs()
	supportedOperationIDs := make(map[string]struct{}, len(operationIDs))
	for _, operationID := range operationIDs {
		supportedOperationIDs[operationID] = struct{}{}
	}
	operationCheckboxes := make(map[string]int, len(operationIDs))
	var inspectNode func(*html.Node)
	inspectNode = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "label" && hasClass(node, "output-option") {
			var inspectOption func(*html.Node)
			inspectOption = func(optionNode *html.Node) {
				if optionNode.Type == html.ElementNode && optionNode.Data == "input" {
					inputType := ""
					inputValue := ""
					for _, attribute := range optionNode.Attr {
						switch attribute.Key {
						case "type":
							inputType = attribute.Val
						case "value":
							inputValue = attribute.Val
						}
					}
					if strings.EqualFold(inputType, "checkbox") {
						if inputValue == "" {
							t.Errorf("GUI operation checkbox has no value")
							return
						}
						operationCheckboxes[inputValue]++
					}
				}
				for child := optionNode.FirstChild; child != nil; child = child.NextSibling {
					inspectOption(child)
				}
			}
			inspectOption(node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			inspectNode(child)
		}
	}
	inspectNode(document)
	if actualCount := len(operationCheckboxes); actualCount != len(operationIDs) {
		t.Fatalf("GUI operation checkbox count = %d, registry count = %d", actualCount, len(operationIDs))
	}
	for operationID, actualCount := range operationCheckboxes {
		if _, supported := supportedOperationIDs[operationID]; !supported {
			t.Errorf("GUI checkbox contains unsupported operation %q", operationID)
		}
		if actualCount != 1 {
			t.Errorf("GUI checkbox for %q occurs %d times, want once", operationID, actualCount)
		}
	}
	for _, operationID := range operationIDs {
		if actualCount := operationCheckboxes[operationID]; actualCount != 1 {
			t.Errorf("GUI checkbox for %q occurs %d times, want once", operationID, actualCount)
		}
	}
}

//	{
//	  責務: [hasClass: HTML要素に指定したclassが含まれるか確認する,]
//	}
func hasClass(node *html.Node, className string) bool {
	for _, attribute := range node.Attr {
		if attribute.Key == "class" && slices.Contains(strings.Fields(attribute.Val), className) {
			return true
		}
	}
	return false
}

//	{
//	  責務: [TestResolveWebviewUserDataPathSeparatesInstallations: 同じ配置先は同一profile、異なる配置先は別profileを使うことを検証する,]
//	}
func TestResolveWebviewUserDataPathSeparatesInstallations(t *testing.T) {
	configDirectory := filepath.Join(t.TempDir(), "config")
	firstInstallDirectory := filepath.Join(t.TempDir(), "first")
	if err := os.MkdirAll(firstInstallDirectory, 0o700); err != nil {
		t.Fatalf("create first installation directory: %v", err)
	}
	firstExecutable := filepath.Join(firstInstallDirectory, "tomiya-code-atlas.exe")
	if err := os.WriteFile(firstExecutable, []byte("test executable"), 0o600); err != nil {
		t.Fatalf("create first executable: %v", err)
	}
	firstPath, err := resolveWebviewUserDataPath(firstExecutable, configDirectory)
	if err != nil {
		t.Fatalf("resolve first WebView2 profile: %v", err)
	}
	firstPathAgain, err := resolveWebviewUserDataPath(firstExecutable, configDirectory)
	if err != nil {
		t.Fatalf("resolve repeated WebView2 profile: %v", err)
	}
	if runtime.GOOS == "windows" {
		firstPathDifferentCase, err := resolveWebviewUserDataPath(strings.ToUpper(firstExecutable), configDirectory)
		if err != nil {
			t.Fatalf("resolve case-insensitive WebView2 profile: %v", err)
		}
		if firstPath != firstPathDifferentCase {
			t.Fatalf("profile changed with Windows path case: first %q, different case %q", firstPath, firstPathDifferentCase)
		}
	}
	secondInstallDirectory := filepath.Join(t.TempDir(), "second")
	if err := os.MkdirAll(secondInstallDirectory, 0o700); err != nil {
		t.Fatalf("create second installation directory: %v", err)
	}
	secondExecutable := filepath.Join(secondInstallDirectory, "tomiya-code-atlas.exe")
	if err := os.WriteFile(secondExecutable, []byte("test executable"), 0o600); err != nil {
		t.Fatalf("create second executable: %v", err)
	}
	secondPath, err := resolveWebviewUserDataPath(secondExecutable, configDirectory)
	if err != nil {
		t.Fatalf("resolve second WebView2 profile: %v", err)
	}
	if !filepath.IsAbs(firstPath) {
		t.Fatalf("profile path = %q, want an absolute path", firstPath)
	}
	if firstPath == secondPath {
		t.Fatalf("profiles for different installation folders are equal: %q", firstPath)
	}
	if firstPath != firstPathAgain {
		t.Fatalf("same installation profile changed: first %q, repeated %q", firstPath, firstPathAgain)
	}
	if !strings.HasPrefix(firstPath, configDirectory) {
		t.Fatalf("profile path %q is outside config directory %q", firstPath, configDirectory)
	}
}

//	{
//	  責務: [TestResolveWebviewUserDataPathResolvesInstallAliases: 同じ実体を指すsymlink経由でも同じWebView2 profileを使うことを検証する,]
//	}
func TestResolveWebviewUserDataPathResolvesInstallAliases(t *testing.T) {
	installDirectory := filepath.Join(t.TempDir(), "install")
	if err := os.MkdirAll(installDirectory, 0o700); err != nil {
		t.Fatalf("create installation directory: %v", err)
	}
	installExecutable := filepath.Join(installDirectory, "tomiya-code-atlas.exe")
	if err := os.WriteFile(installExecutable, []byte("test executable"), 0o600); err != nil {
		t.Fatalf("create installation executable: %v", err)
	}
	aliasExecutable := filepath.Join(t.TempDir(), "tomiya-code-atlas.exe")
	if err := os.Symlink(installExecutable, aliasExecutable); err != nil {
		t.Skipf("create installation alias: %v", err)
	}
	configDirectory := filepath.Join(t.TempDir(), "config")
	installProfile, err := resolveWebviewUserDataPath(installExecutable, configDirectory)
	if err != nil {
		t.Fatalf("resolve installation WebView2 profile: %v", err)
	}
	aliasProfile, err := resolveWebviewUserDataPath(aliasExecutable, configDirectory)
	if err != nil {
		t.Fatalf("resolve alias WebView2 profile: %v", err)
	}
	if installProfile != aliasProfile {
		t.Fatalf("profile differs for the same installation: direct %q, alias %q", installProfile, aliasProfile)
	}
}

//	{
//	  責務: [TestResolveWebviewUserDataPathRejectsEmptyPath: 必須pathが空ならWebView2 profileを決めないことを検証する,]
//	}
func TestResolveWebviewUserDataPathRejectsEmptyPath(t *testing.T) {
	for _, paths := range [][2]string{{"", "config"}, {"app.exe", ""}} {
		if _, err := resolveWebviewUserDataPath(paths[0], paths[1]); err == nil {
			t.Errorf("resolveWebviewUserDataPath(%q, %q) succeeded, want error", paths[0], paths[1])
		}
	}
}

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
//	  責務: [TestGetCompatibleOperationsUsesAnalysisService: GUI bindingが実際のProcess serviceから入力対応操作を返すことを検証する,]
//	}
func TestGetCompatibleOperationsUsesAnalysisService(t *testing.T) {
	root := t.TempDir()
	projectFiles := map[string]string{
		"App.cs":                   "class App {}\n",
		"player.gd":                "extends Node\nfunc run():\n    pass\n",
		"engine.cpp":               "class Engine {};\n",
		"Main.java":                "class Main {}\n",
		"main.go":                  "package main\nfunc main() {}\n",
		"sample.py":                "def run(): pass\n",
		"compose.yaml":             "services:\n  app:\n    image: example/app\n",
		".github/workflows/ci.yml": "name: CI\non: [push, pull_request]\njobs:\n  build:\n    steps:\n      - name: Compile\n        run: go build ./...\n  test:\n    needs: build\n    steps:\n      - uses: actions/checkout@v4\n",
		"settings.yaml":            "theme: dark\n",
	}
	for relativePath, source := range projectFiles {
		path := filepath.Join(root, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("create input directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatalf("write input file %q: %v", relativePath, err)
		}
	}
	application := &Application{generator: process.AnalysisService{}}
	pythonOperations := []string{
		"call-graph", "class-diagram", "sequence-diagram", "object-diagram",
		"communication-diagram", "state-diagram", "timing-chart", "use-case-diagram",
		"responsibility-table", "comments", "static-spec", "design-quality",
	}
	standardSourceOperations := []string{
		"call-graph", "class-diagram", "sequence-diagram", "communication-diagram",
		"responsibility-table", "comments", "static-spec", "design-quality",
	}
	projectOperations := []string{
		"call-graph", "class-diagram", "sequence-diagram", "object-diagram",
		"communication-diagram", "state-diagram", "timing-chart", "use-case-diagram",
		"responsibility-table", "package-diagram", "component-diagram", "deployment-diagram",
		"ci-graph", "comments", "static-spec", "design-quality",
	}
	for _, testCase := range []struct {
		name string
		path string
		want []string
	}{
		{name: "Python source", path: filepath.Join(root, "sample.py"), want: pythonOperations},
		{name: "GDScript source", path: filepath.Join(root, "player.gd"), want: standardSourceOperations},
		{name: "C# source", path: filepath.Join(root, "App.cs"), want: standardSourceOperations},
		{name: "C++ source", path: filepath.Join(root, "engine.cpp"), want: standardSourceOperations},
		{name: "Java source", path: filepath.Join(root, "Main.java"), want: standardSourceOperations},
		{name: "Go source", path: filepath.Join(root, "main.go"), want: standardSourceOperations},
		{name: "workflow YAML", path: filepath.Join(root, filepath.FromSlash(".github/workflows/ci.yml")), want: []string{"ci-graph"}},
		{name: "ordinary YAML", path: filepath.Join(root, "settings.yaml"), want: []string{}},
		{name: "complete project", path: root, want: projectOperations},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := application.GetCompatibleOperations(testCase.path)
			if err != nil {
				t.Fatalf("GetCompatibleOperations() error = %v", err)
			}
			got = slices.Clone(got)
			want := slices.Clone(testCase.want)
			slices.Sort(got)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("GetCompatibleOperations() = %#v, want %#v", got, testCase.want)
			}
		})
	}
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
//	  責務: [TestReportSmokeResultRejectsRegularGUIAndCompletesOnce: smoke mode限定通知と多重完了の抑止を検証する]
//	}
func TestReportSmokeResultRejectsRegularGUIAndCompletesOnce(t *testing.T) {
	regularApplication := &Application{smokeResult: make(chan error, 1)}
	if err := regularApplication.ReportSmokeResult(""); err == nil {
		t.Fatal("regular GUI accepted a smoke result")
	}

	smokeApplication := &Application{
		smokeMode:   true,
		smokeResult: make(chan error, 1),
	}
	if err := smokeApplication.ReportSmokeResult("解析に失敗しました"); err != nil {
		t.Fatalf("ReportSmokeResult returned error: %v", err)
	}
	if err := smokeApplication.ReportSmokeResult(""); err != nil {
		t.Fatalf("duplicate ReportSmokeResult returned error: %v", err)
	}
	if result := <-smokeApplication.smokeResult; result == nil || result.Error() != "解析に失敗しました" {
		t.Fatalf("first smoke result = %v, want analysis error", result)
	}
	if len(smokeApplication.smokeResult) != 0 {
		t.Fatal("duplicate smoke result was enqueued")
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
