package process

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
)

const (
	// {
	//   責務: [helperProcessEnvironmentName: test executableをfake helper processとして動かすmarker名を保持する]
	//   値: [test helper process markerの環境変数名]
	// }
	helperProcessEnvironmentName = "TOMIYA_ATLAS_TEST_HELPER_PROCESS"
	helperModeEnvironmentName    = "TOMIYA_ATLAS_TEST_HELPER_MODE"
)

//	{
//	  責務: [TestHelperClientParsesValidatedCommonIR: helper requestと正常responseのcontract・IR schema・languageが検証されることを確認する]
//	  処理: [1: fake helperを成功response modeで起動する, 2: source pathとCommon IRを受け取る, 3: payloadとrequest対応を検証する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestHelperClientParsesValidatedCommonIR(t *testing.T) {
	t.Setenv(helperModeEnvironmentName, "success")
	client := newTestHelperClient(time.Second)
	sourcePath := "src/main.go"
	payload, err := client.Parse(context.Background(), "fake-helper", "go-stdlib-types-helper", "go", "package demo", &sourcePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if payload.SchemaVersion != commonir.SchemaVersion || payload.Language != "go" {
		t.Fatalf("Parse() payload = %#v, want schema v1 Go module", payload)
	}
	if payload.Entities == nil || payload.Diagnostics == nil || payload.Imports == nil {
		t.Fatalf("Parse() returned nil required Common IR collections: %#v", payload.Module)
	}
}

//	{
//	  責務: [TestHelperClientNormalizesProcessAndProtocolFailures: timeout・終了code・JSON・version・IR異常を安定したfailure kindへ変換することを確認する]
//	  処理: [1: failure modeごとにfake helperを起動する, 2: BackendErrorへ変換された結果を検証する, 3: request元backend IDが維持されることを確認する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestHelperClientNormalizesProcessAndProtocolFailures(t *testing.T) {
	testCases := []struct {
		name          string
		mode          string
		wantKind      string
		wantRetryable bool
	}{
		{name: "non-zero exit", mode: "non-zero", wantKind: "failure"},
		{name: "invalid JSON", mode: "invalid-json", wantKind: "protocol_error"},
		{name: "unsupported helper version", mode: "unsupported-contract", wantKind: "protocol_error"},
		{name: "wrong request ID", mode: "wrong-request", wantKind: "protocol_error"},
		{name: "unsupported Common IR version", mode: "unsupported-ir", wantKind: "protocol_error"},
		{name: "missing Common IR", mode: "missing-ir", wantKind: "protocol_error"},
		{name: "helper unsupported syntax", mode: "unsupported-syntax", wantKind: "unsupported_syntax"},
		{name: "unknown helper failure", mode: "unknown-failure", wantKind: "protocol_error"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv(helperModeEnvironmentName, testCase.mode)
			client := newTestHelperClient(time.Second)
			_, err := client.Parse(context.Background(), "fake-helper", "csharp-roslyn-helper", "csharp", "class Worker {}", nil)
			var backendError *BackendError
			if !errors.As(err, &backendError) {
				t.Fatalf("Parse() error = %v, want BackendError", err)
			}
			if backendError.Failure.Kind != testCase.wantKind {
				t.Errorf("failure kind = %q, want %q", backendError.Failure.Kind, testCase.wantKind)
			}
			if backendError.Failure.BackendID != "csharp-roslyn-helper" {
				t.Errorf("failure backend ID = %q, want request backend ID", backendError.Failure.BackendID)
			}
			if backendError.Failure.Retryable != testCase.wantRetryable {
				t.Errorf("failure retryable = %t, want %t", backendError.Failure.Retryable, testCase.wantRetryable)
			}
		})
	}
}

//	{
//	  責務: [TestHelperClientTimesOut: helperが設定時間を超過したときtimeout診断と再試行可能性を返すことを確認する]
//	  処理: [1: delay modeを選ぶ, 2: 短いtimeoutでhelperを起動する, 3: timeout failureを検証する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestHelperClientTimesOut(t *testing.T) {
	t.Setenv(helperModeEnvironmentName, "delay")
	client := newTestHelperClient(20 * time.Millisecond)
	_, err := client.Parse(context.Background(), "fake-helper", "python-stdlib-ast", "python", "value = 1", nil)
	var backendError *BackendError
	if !errors.As(err, &backendError) {
		t.Fatalf("Parse() error = %v, want BackendError", err)
	}
	if backendError.Failure.Kind != "timeout" || !backendError.Failure.Retryable {
		t.Fatalf("timeout failure = %#v, want retryable timeout", backendError.Failure)
	}
}

//	{
//	  責務: [TestHelperClientProcess: 親test processから起動された場合だけprotocol応答を返すfake helper役を提供する]
//	  処理: [1: helper markerがなければ通常のtestとして戻る, 2: stdin requestをdecodeする, 3: modeに応じてresponse・stderr・exit codeを出す]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestHelperClientProcess(t *testing.T) {
	if os.Getenv(helperProcessEnvironmentName) != "1" {
		return
	}
	requestBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(51)
	}
	var request parseRequest
	if json.Unmarshal(requestBytes, &request) != nil {
		os.Exit(52)
	}
	mode := os.Getenv(helperModeEnvironmentName)
	if mode == "delay" {
		time.Sleep(250 * time.Millisecond)
	}
	if mode == "non-zero" {
		_, _ = os.Stderr.WriteString("fake parser process failed")
		os.Exit(7)
	}
	if mode == "invalid-json" {
		_, _ = os.Stdout.WriteString("{invalid response")
		os.Exit(0)
	}

	response := map[string]any{
		"contract_version": ParserBackendContractVersion,
		"request_id":       request.RequestID,
		"ok":               true,
		"ir": map[string]any{
			"schema_version": commonir.SchemaVersion,
			"language":       request.Language,
			"entities":       []commonir.Entity{},
			"diagnostics":    []commonir.Diagnostic{},
			"imports":        []string{},
		},
	}
	switch mode {
	case "unsupported-contract":
		response["contract_version"] = "2"
	case "wrong-request":
		response["request_id"] = "different-request"
	case "unsupported-ir":
		response["ir"] = map[string]any{"schema_version": "2", "language": request.Language, "entities": []any{}, "diagnostics": []any{}, "imports": []any{}}
	case "missing-ir":
		delete(response, "ir")
	case "unsupported-syntax":
		response["ok"] = false
		response["error"] = map[string]any{"kind": "unsupported_syntax", "message": "syntax is not supported", "backend_id": "untrusted-helper-id"}
		delete(response, "ir")
	case "unknown-failure":
		response["ok"] = false
		response["error"] = map[string]any{"kind": "future_kind", "message": "unknown error"}
		delete(response, "ir")
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		os.Exit(53)
	}
	_, _ = os.Stdout.Write(append(responseBytes, '\n'))
	os.Exit(0)
}

//	{
//	  責務: [newTestHelperClient: current Go test executableをfake helperとして起動するclientを作る]
//	  処理: [1: production clientのtimeout設定を再利用する, 2: test processをhelper executableに差し替える]
//	  引数: [timeout: helper testのtimeout上限]
//	  戻り値: [client: fake helperを起動するprocess client]
//	}
func newTestHelperClient(timeout time.Duration) HelperClient {
	client := NewHelperClient(timeout)
	client.createCommand = func(ctx context.Context, _ string) *exec.Cmd {
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperClientProcess$")
		command.Env = append(os.Environ(), helperProcessEnvironmentName+"=1")
		return command
	}
	return client
}

//	{
//	  責務: [TestHelperClientNormalizesProcessFailuresErrorText: stderrがある場合は終了codeより具体的な内容を返すことを検証する]
//	  処理: [1: nonzero modeでhelperを起動する, 2: error messageにstderrが含まれることを確認する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestHelperClientNormalizesProcessFailuresErrorText(t *testing.T) {
	t.Setenv(helperModeEnvironmentName, "non-zero")
	client := newTestHelperClient(time.Second)
	_, err := client.Parse(context.Background(), "fake-helper", "test-backend", "go", "package x", nil)
	if err == nil || !strings.Contains(err.Error(), "fake parser process failed") {
		t.Fatalf("Parse() error = %v, want helper stderr", err)
	}
}
