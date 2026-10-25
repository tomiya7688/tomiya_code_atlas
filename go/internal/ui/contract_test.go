package ui

import (
	"context"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
)

//	{
//	  責務: [TestResolvePathFromDirectoryUsesStableWorkingDirectory: 相対パスと「.」をGUI起動時のフォルダー基準で解決することを検証する,],
//	}
func TestResolvePathFromDirectoryUsesStableWorkingDirectory(t *testing.T) {
	resolved, err := resolvePathFromDirectory(" atlas-output ", filepath.Clean("project-root"))
	if err != nil {
		t.Fatalf("resolvePathFromDirectory returned error: %v", err)
	}
	want := filepath.Join(filepath.Clean("project-root"), "atlas-output")
	if resolved != want {
		t.Fatalf("resolved relative path = %q, want %q", resolved, want)
	}
	if empty, err := resolvePathFromDirectory("   ", filepath.Clean("project-root")); err != nil || empty != "" {
		t.Fatalf("empty input path = %q, want empty", empty)
	}
	if currentDirectory, err := resolvePathFromDirectory(".", filepath.Clean("project-root")); err != nil || currentDirectory != filepath.Clean("project-root") {
		t.Fatalf("current directory path = %q, want %q", currentDirectory, filepath.Clean("project-root"))
	}
	quotedPath := `"` + filepath.Join(t.TempDir(), "main.py") + `"`
	if resolvedQuotedPath, err := resolvePathFromDirectory(quotedPath, filepath.Clean("project-root")); err != nil || resolvedQuotedPath != filepath.Clean(quotedPath[1:len(quotedPath)-1]) {
		t.Fatalf("quoted absolute path = %q, error %v", resolvedQuotedPath, err)
	}
	if runtime.GOOS == "windows" {
		for _, ambiguousPath := range []string{`C:project\main.py`, `\project\main.py`, `/project/main.py`} {
			if _, err := resolvePathFromDirectory(ambiguousPath, filepath.Clean(`C:\workspace`)); err == nil {
				t.Fatalf("ambiguous Windows path was accepted: %q", ambiguousPath)
			}
		}
	}
}

//	{
//	  責務: [TestDefaultOutputDirectoryUsesWritableUserLocation: 既定の保存先がホームフォルダーを優先し、取得できない場合は実行時フォルダーを使うことを検証する,],
//	  処理: [1: ホームフォルダーがある場合とない場合の保存先を作る, 2: 製品専用フォルダー配下になることを確認する,],
//	}
func TestDefaultOutputDirectoryUsesWritableUserLocation(t *testing.T) {
	userHomeDirectory := t.TempDir()
	gotHomeOutput := defaultOutputDirectory(userHomeDirectory, filepath.Clean("application"))
	if filepath.Dir(filepath.Dir(gotHomeOutput)) != userHomeDirectory || filepath.Base(filepath.Dir(gotHomeOutput)) != "Tomiya Code Atlas" || filepath.Base(gotHomeOutput) != "atlas-output" {
		t.Fatalf("home output directory has unexpected components: %q", gotHomeOutput)
	}
	workingDirectory := t.TempDir()
	gotFallbackOutput := defaultOutputDirectory("", workingDirectory)
	if filepath.Dir(filepath.Dir(gotFallbackOutput)) != workingDirectory || filepath.Base(filepath.Dir(gotFallbackOutput)) != "Tomiya Code Atlas" || filepath.Base(gotFallbackOutput) != "atlas-output" {
		t.Fatalf("fallback output directory has unexpected components: %q", gotFallbackOutput)
	}
}

//	{
//	  責務: [panicDiagramGenerator: UI panic回復テストで予期しない解析停止を再現する,],
//	}
type panicDiagramGenerator struct{}

//	{
//	  責務: [Generate: UI panic回復テスト用に解析中のpanicを発生させる,],
//	}
func (panicDiagramGenerator) Generate(context.Context, process.DiagramRequest) ([]string, error) {
	panic("解析panicのテスト")
}

//	{
//	  責務: [TestGenerateDiagramsSafelyConvertsPanicIntoDiagnostic: 解析サービスのpanicが画面へ返せるerrorに変わることを検証する,],
//	}
func TestGenerateDiagramsSafelyConvertsPanicIntoDiagnostic(t *testing.T) {
	_, err := generateDiagramsSafely(context.Background(), panicDiagramGenerator{}, process.DiagramRequest{})
	if err == nil || !strings.Contains(err.Error(), "予期せず停止") {
		t.Fatalf("panic recovery error = %v, want a Japanese diagnostic", err)
	}
}

//	{
//	  責務: [TestCancelPendingAnalysisStopsGenerationBeforeWindowExit: ウィンドウを閉じる時に解析をキャンセルし、終了を待つことを検証する,],
//	  処理: [1: キャンセルに応じて終了する解析を用意する, 2: 終了待ち後にcancel関数と完了channelが解放されることを確認する,],
//	}
func TestCancelPendingAnalysisStopsGenerationBeforeWindowExit(t *testing.T) {
	cancelRequested := make(chan struct{})
	analysisCanFinish := make(chan struct{})
	analysisDone := make(chan struct{})
	go func() {
		<-cancelRequested
		<-analysisCanFinish
		close(analysisDone)
	}()
	time.AfterFunc(25*time.Millisecond, func() { close(analysisCanFinish) })
	application := &desktopApp{
		analysisCancel: func() { close(cancelRequested) },
		analysisDone:   analysisDone,
	}
	startedAt := time.Now()
	application.cancelPendingAnalysis()
	if elapsed := time.Since(startedAt); elapsed < 20*time.Millisecond {
		t.Fatalf("cancelPendingAnalysis returned after %s without waiting for analysis completion", elapsed)
	}
	select {
	case <-analysisDone:
	default:
		t.Fatal("cancelPendingAnalysis returned before analysis completed")
	}
	if application.analysisCancel != nil || application.analysisDone != nil {
		t.Fatal("analysis cancellation resources were not released")
	}
}

//	{
//	  責務: [TestBuildDiagramRequestAllowsMultipleOperations: 画面で選んだ複数の図を一つのProcess依頼へ渡せることを検証する,],
//	  処理: [1: 対応する二種類の図と入出力パスから依頼を作る, 2: 値と順序を確認する,],
//	}
func TestBuildDiagramRequestAllowsMultipleOperations(t *testing.T) {
	request, err := buildDiagramRequest(" sample.py ", " output ", []string{
		process.OperationCallGraph,
		process.OperationClassDiagram,
	})
	if err != nil {
		t.Fatalf("buildDiagramRequest returned error: %v", err)
	}
	if request.SourcePath != "sample.py" || request.OutputDirectory != "output" {
		t.Fatalf("request paths = (%q, %q)", request.SourcePath, request.OutputDirectory)
	}
	wantOperations := []string{process.OperationCallGraph, process.OperationClassDiagram}
	if !reflect.DeepEqual(request.Operations, wantOperations) {
		t.Fatalf("request operations = %#v, want %#v", request.Operations, wantOperations)
	}
}

//	{
//	  責務: [TestBuildDiagramRequestRejectsInvalidInput: 入力不足・未対応の図・重複選択をProcessへ渡す前に拒否する,],
//	  処理: [1: 不正な入力例を順に試す, 2: 依頼が返らず日本語のエラーになることを確認する,],
//	}
func TestBuildDiagramRequestRejectsInvalidInput(t *testing.T) {
	testCases := []struct {
		name          string
		sourcePath    string
		outputPath    string
		operations    []string
		wantErrorText string
	}{
		{name: "missing input", outputPath: "out", operations: []string{process.OperationCallGraph}, wantErrorText: "解析するファイル"},
		{name: "missing output", sourcePath: "sample.py", operations: []string{process.OperationCallGraph}, wantErrorText: "保存先フォルダー"},
		{name: "missing operations", sourcePath: "sample.py", outputPath: "out", wantErrorText: "1つ以上"},
		{name: "unsupported operation", sourcePath: "sample.py", outputPath: "out", operations: []string{"deployment"}, wantErrorText: "利用できない"},
		{name: "duplicate operation", sourcePath: "sample.py", outputPath: "out", operations: []string{process.OperationCallGraph, process.OperationCallGraph}, wantErrorText: "重複"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request, err := buildDiagramRequest(testCase.sourcePath, testCase.outputPath, testCase.operations)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrorText) {
				t.Fatalf("buildDiagramRequest error = %v, want text %q", err, testCase.wantErrorText)
			}
			if request.SourcePath != "" || len(request.Operations) != 0 {
				t.Fatalf("invalid input returned a partial request: %#v", request)
			}
		})
	}
}

//	{
//	  責務: [TestSupportedOperationDefinitionsMatchProcessContract: 画面がProcessと同じ順序で対応済みの図を表示することを検証する,],
//	}
func TestSupportedOperationDefinitionsMatchProcessContract(t *testing.T) {
	definitions := supportedOperationDefinitions()
	if len(definitions) != len(process.SupportedDiagramOperations()) {
		t.Fatalf("UI operations = %#v, Process operations = %#v", definitions, process.SupportedDiagramOperations())
	}
	for index, operation := range process.SupportedDiagramOperations() {
		if definitions[index].id != operation || definitions[index].label == "" {
			t.Errorf("UI operation %d = %#v, want ID %q and a Japanese label", index, definitions[index], operation)
		}
	}
}
