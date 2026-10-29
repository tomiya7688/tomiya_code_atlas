package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
)

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run returned %d; want 0", code)
	}
	if !strings.Contains(stdout.String(), "使用方法:") {
		t.Fatalf("help output is missing usage: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "コマンドを指定せずに起動するとGUIが開きます。") {
		t.Fatalf("help is missing the GUI entry: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("help wrote to stderr: %q", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run returned %d; want 0", code)
	}
	if !strings.Contains(stdout.String(), "Tomiya Code Atlas (Go)") {
		t.Fatalf("version output is missing product name: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("version wrote to stderr: %q", stderr.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"class-diagram"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run returned %d; want 2", code)
	}
	if !strings.Contains(stderr.String(), "[エラー]") {
		t.Fatalf("error output is not Japanese: %q", stderr.String())
	}
}

func TestRunWithGUIOpensDesktopAndSmokeMode(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		args      []string
		wantSmoke bool
	}{
		{name: "default opens GUI", wantSmoke: false},
		{name: "hidden smoke opens and exits", args: []string{"--gui-smoke"}, wantSmoke: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			called := false
			code := runWithGUI(testCase.args, &stdout, &stderr, func(smoke bool) error {
				called = true
				if smoke != testCase.wantSmoke {
					t.Fatalf("smoke mode = %v, want %v", smoke, testCase.wantSmoke)
				}
				return nil
			})
			if code != exitSuccess || !called || stderr.Len() != 0 {
				t.Fatalf("code=%d called=%v stderr=%q", code, called, stderr.String())
			}
		})
	}
}

//	{
//	  責務: [TestRunAnalyzeReportsStableUsageAndRuntimeExitCodes: command引数誤りと実行環境失敗のexit codeを検証する]
//	}
func TestRunAnalyzeReportsStableUsageAndRuntimeExitCodes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"analyze"}, &stdout, &stderr); code != exitUsageError {
		t.Fatalf("usage failure exit code = %d, want %d", code, exitUsageError)
	}
	if !strings.Contains(stderr.String(), "[エラー]") {
		t.Fatalf("usage failure is not Japanese: %q", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	arguments := []string{"analyze", "unused.py", "--operation", "call-graph", "--output-dir", "unused-output"}
	if code := run(arguments, &stdout, &stderr); code != exitAnalysisError {
		t.Fatalf("runtime failure exit code = %d, want %d", code, exitAnalysisError)
	}
	if !strings.Contains(stderr.String(), "[エラー]") {
		t.Fatalf("runtime failure is not Japanese: %q", stderr.String())
	}
}

//	{
//	  責務: [TestWriteAnalyzeHelpListsEverySupportedOperation: helpに各operationの用途を日本語で表示する]
//	}
func TestWriteAnalyzeHelpListsEverySupportedOperation(t *testing.T) {
	var output bytes.Buffer
	writeAnalyzeHelp(&output)
	for _, operation := range process.SupportedOperationIDs() {
		if !strings.Contains(output.String(), operation) {
			t.Errorf("analyze help does not list %q: %q", operation, output.String())
		}
		if strings.Contains(analysisOperationDescription(operation), "未対応") {
			t.Errorf("operation %q has no Japanese description", operation)
		}
	}
	if !strings.Contains(output.String(), "元ソースを変更せず") || !strings.Contains(output.String(), "設計品質レポート") {
		t.Fatalf("analyze help omits user-facing operation details: %q", output.String())
	}
}

//	{
//	  責務: [cliContractFixture: versioned CLI契約fixtureのcommand・option・exit codeを受け取る]
//	}
type cliContractFixture struct {
	SchemaVersion string `json:"schema_version"`
	Commands      []struct {
		Name               string   `json:"name"`
		RequiredOptions    []string `json:"required_options"`
		RepeatableOptions  []string `json:"repeatable_options"`
		SupportedOperation []string `json:"supported_operations"`
		InputPath          struct {
			Accepts       []string `json:"accepts"`
			DirectoryScan string   `json:"directory_scan"`
		} `json:"input_path"`
	} `json:"commands"`
	ExitCodes map[string]int `json:"exit_codes"`
}

//	{
//	  責務: [TestCLIContractFixtureMatchesCommandAndExitCodes: versioned CLI contract fixtureを実装のoperation・exit codeと照合する]
//	}
func TestCLIContractFixtureMatchesCommandAndExitCodes(t *testing.T) {
	contents, err := os.ReadFile("cli_contract_v1.json")
	if err != nil {
		t.Fatalf("CLI contract fixture is unavailable: %v", err)
	}
	var fixture cliContractFixture
	if err := json.Unmarshal(contents, &fixture); err != nil {
		t.Fatalf("CLI contract fixture is invalid JSON: %v", err)
	}
	if fixture.SchemaVersion != "1" || len(fixture.Commands) != 1 || fixture.Commands[0].Name != "analyze" {
		t.Fatalf("unexpected CLI contract fixture shape: %#v", fixture)
	}
	command := fixture.Commands[0]
	if !containsString(command.InputPath.Accepts, "file") || !containsString(command.InputPath.Accepts, "directory") || command.InputPath.DirectoryScan != "recursive" {
		t.Fatalf("input path contract = %#v", command.InputPath)
	}
	if len(command.RequiredOptions) != 2 || !containsString(command.RequiredOptions, "--operation") || !containsString(command.RequiredOptions, "--output-dir") {
		t.Fatalf("required options = %#v", command.RequiredOptions)
	}
	if len(command.RepeatableOptions) != 1 || command.RepeatableOptions[0] != "--operation" {
		t.Fatalf("repeatable options = %#v", command.RepeatableOptions)
	}
	if len(command.SupportedOperation) != len(process.SupportedOperationIDs()) {
		t.Fatalf("fixture operations = %#v, implementation operations = %#v", command.SupportedOperation, process.SupportedOperationIDs())
	}
	for _, operation := range command.SupportedOperation {
		if !process.IsSupportedOperation(operation) {
			t.Fatalf("fixture operation %q has no implementation", operation)
		}
	}
	wantExitCodes := map[string]int{"success": exitSuccess, "analysis_error": exitAnalysisError, "usage_error": exitUsageError}
	for name, code := range wantExitCodes {
		if fixture.ExitCodes[name] != code {
			t.Errorf("exit code %s = %d, want %d", name, fixture.ExitCodes[name], code)
		}
	}
}

//	{
//	  責務: [containsString: string sliceが指定値を含むか確認する]
//	  処理: [1: sliceの要素を順に比較する, 2: 一致有無を返す]
//	  引数: [values: 検索対象, expected: 探す文字列]
//	  戻り値: [bool: 一致すればtrue]
//	}
func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

//	{
//	  責務: [TestParseAnalyzeArgumentsSupportsMultipleOperations: 一つのanalyze commandで複数operationを選べる契約を検証する]
//	}
func TestParseAnalyzeArgumentsSupportsMultipleOperations(t *testing.T) {
	options, err := parseAnalyzeArguments([]string{
		"src/main.py",
		"--operation", "call-graph",
		"--operation", "class-diagram",
		"--operation", "comments",
		"--operation", "static-spec",
		"--operation", "design-quality",
		"--output-dir", "atlas-output",
	})
	if err != nil {
		t.Fatalf("parseAnalyzeArguments returned error: %v", err)
	}
	if options.sourcePath != "src/main.py" || options.outputDirectory != "atlas-output" {
		t.Fatalf("parsed paths = (%q, %q)", options.sourcePath, options.outputDirectory)
	}
	if len(options.operations) != len(process.SupportedOperationIDs()) {
		t.Fatalf("parsed operations = %#v", options.operations)
	}
	for index, operation := range process.SupportedOperationIDs() {
		if options.operations[index] != operation {
			t.Fatalf("parsed operations = %#v, want %#v", options.operations, process.SupportedOperationIDs())
		}
	}
}

//	{
//	  責務: [TestParseAnalyzeArgumentsRejectsIncompleteOrUnsafeSelection: 必須値・未対応operation・重複operationを利用者へ拒否する]
//	}
func TestParseAnalyzeArgumentsRejectsIncompleteOrUnsafeSelection(t *testing.T) {
	testCases := []struct {
		name      string
		arguments []string
		wantError string
	}{
		{name: "missing source", arguments: []string{"--operation", "call-graph", "--output-dir", "out"}, wantError: "source file"},
		{name: "missing operation", arguments: []string{"main.py", "--output-dir", "out"}, wantError: "--operation"},
		{name: "missing output directory", arguments: []string{"main.py", "--operation", "call-graph"}, wantError: "--output-dir"},
		{name: "unsupported operation", arguments: []string{"main.py", "--operation", "use-cases", "--output-dir", "out"}, wantError: "未対応"},
		{name: "duplicate operation", arguments: []string{"main.py", "--operation", "call-graph", "--operation", "call-graph", "--output-dir", "out"}, wantError: "重複"},
		{name: "duplicate output directory", arguments: []string{"main.py", "--operation", "call-graph", "--output-dir", "first", "--output-dir", "second"}, wantError: "一度だけ"},
		{name: "unknown option", arguments: []string{"main.py", "--unknown", "value"}, wantError: "認識できないオプション"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := parseAnalyzeArguments(testCase.arguments)
			if err == nil || !strings.Contains(err.Error(), testCase.wantError) {
				t.Fatalf("parseAnalyzeArguments error = %v, want text %q", err, testCase.wantError)
			}
		})
	}
}
