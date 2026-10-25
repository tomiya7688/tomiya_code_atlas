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

//	{
//	  責務: [TestRunAnalyzeReportsStableUsageAndRuntimeExitCodes: 引数誤りと解析開始時の失敗で終了コードが分かれることを検証する,],
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
//	  責務: [cliContractFixture: CLI契約に記録されたコマンド・指定値・終了コードを保持する,],
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
//	  責務: [TestCLIContractFixtureMatchesCommandAndExitCodes: CLI契約の入力・処理ID・終了コードが実装と一致することを検証する,],
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
	if len(command.SupportedOperation) != len(supportedOperations) {
		t.Fatalf("fixture operations = %#v, implementation operations = %#v", command.SupportedOperation, supportedOperations)
	}
	for _, operation := range command.SupportedOperation {
		if _, supported := supportedOperations[operation]; !supported {
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
//	  責務: [TestCLIUsesProcessDiagramOperationCatalog: CLIがProcessの対応する図一覧を検証とエラー表示に使うことを確認する,],
//	  処理: [1: CLIの対応一覧とProcessの一覧を比べる, 2: 未対応値のエラーに最新一覧が含まれることを確認する,],
//	}
func TestCLIUsesProcessDiagramOperationCatalog(t *testing.T) {
	operations := process.SupportedDiagramOperations()
	if len(supportedOperations) != len(operations) {
		t.Fatalf("CLI operations = %#v, Process operations = %#v", supportedOperations, operations)
	}
	for _, operation := range operations {
		if _, supported := supportedOperations[operation]; !supported {
			t.Errorf("Process operation %q is missing from CLI validation", operation)
		}
	}
	_, err := parseAnalyzeArguments([]string{"main.py", "--operation", "unsupported", "--output-dir", "out"})
	if err == nil || !strings.Contains(err.Error(), strings.Join(operations, ", ")) {
		t.Fatalf("unsupported operation error = %v, want Process operation list %q", err, strings.Join(operations, ", "))
	}
}

//	{
//	  責務: [containsString: 文字列一覧に指定した値があるか調べる,],
//	  処理: [1: 一覧の要素を順に比較する, 2: 一致した時点でtrueを返す,],
//	  引数: [values: 検索する文字列の一覧, expected: 探す文字列,],
//	  戻り値: [bool: 一致した場合はtrue,],
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
//	  責務: [TestParseAnalyzeArgumentsSupportsMultipleOperations: analyzeコマンドで複数の図を指定できることを検証する,],
//	}
func TestParseAnalyzeArgumentsSupportsMultipleOperations(t *testing.T) {
	options, err := parseAnalyzeArguments([]string{
		"src/main.py",
		"--operation", "call-graph",
		"--operation", "class-diagram",
		"--output-dir", "atlas-output",
	})
	if err != nil {
		t.Fatalf("parseAnalyzeArguments returned error: %v", err)
	}
	if options.sourcePath != "src/main.py" || options.outputDirectory != "atlas-output" {
		t.Fatalf("parsed paths = (%q, %q)", options.sourcePath, options.outputDirectory)
	}
	if len(options.operations) != 2 || options.operations[0] != "call-graph" || options.operations[1] != "class-diagram" {
		t.Fatalf("parsed operations = %#v", options.operations)
	}
}

//	{
//	  責務: [TestParseAnalyzeArgumentsRejectsIncompleteOrUnsafeSelection: 必須値の不足・未対応の図・重複指定を拒否する,],
//	}
func TestParseAnalyzeArgumentsRejectsIncompleteOrUnsafeSelection(t *testing.T) {
	testCases := []struct {
		name      string
		arguments []string
		wantError string
	}{
		{name: "missing source", arguments: []string{"--operation", "call-graph", "--output-dir", "out"}, wantError: "ソースファイル"},
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
