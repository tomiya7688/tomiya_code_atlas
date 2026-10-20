package process

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
)

//	{
//	  責務: [TestParserHostSelectsManifestBackendAndReturnsCommonIR: manifestからplatform helperを選択し、JSON protocol経由でparseする流れを確認する]
//	  処理: [1: 一時配布rootとregistryを作る, 2: OS/architecture固有pathを登録する, 3: ParserHostでparseし選択pathと言語結果を検証する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestParserHostSelectsManifestBackendAndReturnsCommonIR(t *testing.T) {
	t.Setenv(helperModeEnvironmentName, "success")
	applicationRoot := t.TempDir()
	platformExecutable := writeParserHostTestExecutable(t, applicationRoot, "platform/atlas-helper")
	writeParserHostTestExecutable(t, applicationRoot, "default/atlas-helper")
	manifest := data.BackendManifest{
		ManifestVersion:              "1",
		ParserBackendContractVersion: ParserBackendContractVersion,
		Backends: []data.Backend{{
			BackendID: "go-test-helper",
			Language:  "go",
			Kind:      "helper",
			Path:      "default/atlas-helper",
			Platforms: map[string]string{runtime.GOOS + "/" + runtime.GOARCH: "platform/atlas-helper"},
		}},
	}
	writeParserHostTestManifest(t, applicationRoot, manifest)
	host, err := NewParserHost(applicationRoot, time.Second)
	if err != nil {
		t.Fatalf("NewParserHost() error = %v", err)
	}
	var invokedPath string
	host.helperClient.createCommand = func(ctx context.Context, executablePath string) *exec.Cmd {
		invokedPath = executablePath
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperClientProcess$")
		command.Env = append(os.Environ(), helperProcessEnvironmentName+"=1")
		return command
	}

	payload, err := host.Parse(context.Background(), "go-test-helper", "go", "package demo", nil)
	if err != nil {
		t.Fatalf("ParserHost.Parse() error = %v", err)
	}
	if invokedPath != platformExecutable {
		t.Fatalf("selected helper path = %q, want platform path %q", invokedPath, platformExecutable)
	}
	if payload.Language != "go" {
		t.Fatalf("parsed module language = %q, want go", payload.Language)
	}
}

//	{
//	  責務: [TestParserHostRunsRepositoryGoParserHelper: repository Go parser helperをbuildし、Process hostの実際のJSON boundaryでCommon IRへ変換できることを検証する]
//	  処理: [1: test sourceからrepository rootを決める, 2: Go helper executableをbuildする, 3: helper registryからhostを作る, 4: 実sourceをparseして宣言を確認する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestParserHostRunsRepositoryGoParserHelper(t *testing.T) {
	_, testSourcePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() did not return the test source path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(testSourcePath), "..", "..", ".."))
	helperModuleDirectory := filepath.Join(repositoryRoot, "backends", "go")
	helperName := "tomiya-go-backend"
	if runtime.GOOS == "windows" {
		helperName += ".exe"
	}

	applicationRoot := t.TempDir()
	helperPath := filepath.Join(applicationRoot, "backends", helperName)
	if err := os.MkdirAll(filepath.Dir(helperPath), 0o755); err != nil {
		t.Fatalf("create backend asset directory: %v", err)
	}
	buildCommand := exec.Command("go", "build", "-o", helperPath, ".")
	buildCommand.Dir = helperModuleDirectory
	buildOutput, err := buildCommand.CombinedOutput()
	if err != nil {
		t.Fatalf("build Go parser helper: %v\n%s", err, buildOutput)
	}
	manifest := data.BackendManifest{
		ManifestVersion:              "1",
		ParserBackendContractVersion: ParserBackendContractVersion,
		Backends: []data.Backend{{
			BackendID: "go-stdlib-types-helper",
			Language:  "go",
			Kind:      "helper",
			Path:      helperName,
			Platforms: map[string]string{runtime.GOOS + "/" + runtime.GOARCH: helperName},
		}},
	}
	writeParserHostTestManifest(t, applicationRoot, manifest)
	host, err := NewParserHost(applicationRoot, 10*time.Second)
	if err != nil {
		t.Fatalf("NewParserHost() error = %v", err)
	}
	sourcePath := "demo.go"
	source := "package demo\n\nimport \"fmt\"\n\nfunc Run() { fmt.Println(\"ready\") }\n"
	payload, err := host.Parse(context.Background(), "go-stdlib-types-helper", "go", source, &sourcePath)
	if err != nil {
		t.Fatalf("ParserHost.Parse() error = %v", err)
	}
	for _, entity := range payload.Entities {
		if entity.Name == "Run" {
			return
		}
	}
	t.Fatalf("Go parser response omitted function Run: %#v", payload.Entities)
}

//	{
//	  責務: [TestParserHostRejectsUnknownAndMismatchedBackends: 未登録helperまたは異なる言語のhelperを起動前に拒否することを検証する]
//	  処理: [1: Python backendだけをregistryへ登録する, 2: 未登録IDを指定して拒否を確認する, 3: 言語不一致を指定して拒否を確認する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestParserHostRejectsUnknownAndMismatchedBackends(t *testing.T) {
	applicationRoot := t.TempDir()
	writeParserHostTestExecutable(t, applicationRoot, "python-helper")
	manifest := data.BackendManifest{
		ManifestVersion:              "1",
		ParserBackendContractVersion: ParserBackendContractVersion,
		Backends: []data.Backend{{
			BackendID: "python-helper",
			Language:  "python",
			Kind:      "helper",
			Path:      "python-helper",
		}},
	}
	writeParserHostTestManifest(t, applicationRoot, manifest)
	host, err := NewParserHost(applicationRoot, time.Second)
	if err != nil {
		t.Fatalf("NewParserHost() error = %v", err)
	}

	for _, testCase := range []struct {
		backendID string
		language  string
	}{
		{backendID: "missing-helper", language: "python"},
		{backendID: "python-helper", language: "go"},
	} {
		_, err := host.Parse(context.Background(), testCase.backendID, testCase.language, "pass", nil)
		backendError, isBackendError := err.(*BackendError)
		if !isBackendError || backendError.Failure.Kind != "unsupported_language" {
			t.Errorf("ParserHost.Parse(%q, %q) error = %v, want unsupported_language", testCase.backendID, testCase.language, err)
		}
	}
}

//	{
//	  責務: [TestParserHostRejectsUnsupportedManifestContract: registryが要求する未知のhelper contract versionを起動前に拒否することを検証する]
//	  処理: [1: 未対応contract versionを含むmanifestを書く, 2: ParserHostを作る, 3: supported versionを示すerrorを確認する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestParserHostRejectsUnsupportedManifestContract(t *testing.T) {
	applicationRoot := t.TempDir()
	writeParserHostTestManifest(t, applicationRoot, data.BackendManifest{
		ManifestVersion:              "1",
		ParserBackendContractVersion: "99",
		Backends: []data.Backend{{
			BackendID: "python-helper",
			Language:  "python",
			Path:      "python-helper",
		}},
	})
	if _, err := NewParserHost(applicationRoot, time.Second); err == nil || !strings.Contains(err.Error(), `contract version "99"`) {
		t.Fatalf("NewParserHost() error = %v, want unsupported contract version error", err)
	}
}

//	{
//	  責務: [writeParserHostTestManifest: test用配布rootへbackend manifestをJSON形式で書き込む]
//	  処理: [1: manifest directoryを作る, 2: versioned registryをencodeする, 3: UTF-8 JSON fileを保存する]
//	  引数: [t: Go test runner, applicationRoot: test配布root, manifest: registry fixture]
//	  戻り値: []
//	}
func writeParserHostTestManifest(t *testing.T, applicationRoot string, manifest data.BackendManifest) {
	t.Helper()
	backendDirectory := filepath.Join(applicationRoot, "backends")
	if err := os.MkdirAll(backendDirectory, 0o755); err != nil {
		t.Fatalf("create backend manifest directory: %v", err)
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("encode backend manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDirectory, "manifest.json"), manifestBytes, 0o600); err != nil {
		t.Fatalf("write backend manifest: %v", err)
	}
}

//	{
//	  責務: [writeParserHostTestExecutable: manifest path検証用に通常file assetを作る]
//	  処理: [1: backend asset directoryを作る, 2: test process bytesを保存する, 3: manifest-relative pathを返す]
//	  引数: [t: Go test runner, applicationRoot: test配布root, relativePath: backends directoryからの相対path]
//	  戻り値: [executablePath: 作成したassetの絶対path]
//	}
func writeParserHostTestExecutable(t *testing.T, applicationRoot string, relativePath string) string {
	t.Helper()
	executablePath := filepath.Join(applicationRoot, "backends", relativePath)
	if err := os.MkdirAll(filepath.Dir(executablePath), 0o755); err != nil {
		t.Fatalf("create backend asset directory: %v", err)
	}
	executableBytes, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatalf("read test executable: %v", err)
	}
	if err := os.WriteFile(executablePath, executableBytes, 0o700); err != nil {
		t.Fatalf("write backend test asset: %v", err)
	}
	return executablePath
}
