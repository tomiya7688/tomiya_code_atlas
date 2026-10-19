package data

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

//	{
//	  責務: [TestLoadRepositoryBackendManifest: repository manifestが現在のGo契約と互換なregistryとして読めることを検証する]
//	  処理: [1: repository rootからmanifestを読む, 2: versionと登録件数を検証する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestLoadRepositoryBackendManifest(t *testing.T) {
	// テストのworking directoryはGoの実行環境で異なるため、固定の相対階層を使わずsource pathを取得する。
	_, testSourcePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() did not return the test source path")
	}
	// test sourceはgo/internal/dataにあるため、そこから三階層上をrepository rootとしてmanifestを読む。
	applicationRoot := filepath.Clean(filepath.Join(filepath.Dir(testSourcePath), "..", "..", ".."))
	manifest, err := LoadBackendManifest(applicationRoot)
	if err != nil {
		t.Fatalf("LoadBackendManifest() error = %v", err)
	}
	if manifest.ManifestVersion != supportedManifestVersion || manifest.ParserBackendContractVersion != "1" {
		t.Fatalf("manifest contract = (%q, %q), want (1, 1)", manifest.ManifestVersion, manifest.ParserBackendContractVersion)
	}
	if len(manifest.Backends) < 6 {
		t.Fatalf("registered backend count = %d, want at least 6", len(manifest.Backends))
	}
}

//	{
//	  責務: [TestResolveBackendExecutableUsesPlatformPath: OS/architecture固有pathを既定pathより優先することを検証する]
//	  処理: [1: 配布rootにhelper fileを作る, 2: platform固有pathから解決する, 3: 選択pathを検証する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestResolveBackendExecutableUsesPlatformPath(t *testing.T) {
	applicationRoot := t.TempDir()
	platformHelper := filepath.Join(applicationRoot, "backends", "windows", "atlas.exe")
	writeBackendTestFile(t, platformHelper)
	manifest := BackendManifest{Backends: []Backend{{
		BackendID: "test-helper",
		Path:      "default/atlas.exe",
		Platforms: map[string]string{"windows/amd64": "windows/atlas.exe"},
	}}}

	resolvedPath, err := ResolveBackendExecutable(applicationRoot, manifest, "test-helper", "windows", "amd64")
	if err != nil {
		t.Fatalf("ResolveBackendExecutable() error = %v", err)
	}
	if resolvedPath != platformHelper {
		t.Fatalf("resolved path = %q, want %q", resolvedPath, platformHelper)
	}
}

//	{
//	  責務: [TestResolveBackendExecutableUsesDefaultPath: platform別pathがない環境で既定pathを使うことを検証する]
//	  処理: [1: 配布rootに既定helper fileを作る, 2: 別platformから解決する, 3: 既定pathを検証する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestResolveBackendExecutableUsesDefaultPath(t *testing.T) {
	applicationRoot := t.TempDir()
	defaultHelper := filepath.Join(applicationRoot, "backends", "default", "atlas")
	writeBackendTestFile(t, defaultHelper)
	manifest := BackendManifest{Backends: []Backend{{BackendID: "test-helper", Path: "default/atlas"}}}

	resolvedPath, err := ResolveBackendExecutable(applicationRoot, manifest, "test-helper", "linux", "arm64")
	if err != nil {
		t.Fatalf("ResolveBackendExecutable() error = %v", err)
	}
	if resolvedPath != defaultHelper {
		t.Fatalf("resolved path = %q, want %q", resolvedPath, defaultHelper)
	}
}

//	{
//	  責務: [TestResolveBackendExecutableRejectsPathEscape: manifestからbackends directory外を参照できないことを検証する]
//	  処理: [1: 親directoryを含むpathをregistryへ設定する, 2: helperを解決する, 3: path逸脱errorを検証する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestResolveBackendExecutableRejectsPathEscape(t *testing.T) {
	manifest := BackendManifest{Backends: []Backend{{BackendID: "test-helper", Path: "../outside.exe"}}}
	_, err := ResolveBackendExecutable(t.TempDir(), manifest, "test-helper", "windows", "amd64")
	if err == nil || !strings.Contains(err.Error(), "escapes the backends directory") {
		t.Fatalf("ResolveBackendExecutable() error = %v, want path escape rejection", err)
	}
}

//	{
//	  責務: [TestResolveBackendExecutableReportsMissingAsset: manifest登録済みでも未配布fileならerrorになることを検証する]
//	  処理: [1: 存在しないpathをregistryへ設定する, 2: helperを解決する, 3: 不在fileの診断を検証する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestResolveBackendExecutableReportsMissingAsset(t *testing.T) {
	manifest := BackendManifest{Backends: []Backend{{BackendID: "test-helper", Path: "missing/atlas.exe"}}}
	_, err := ResolveBackendExecutable(t.TempDir(), manifest, "test-helper", "windows", "amd64")
	if err == nil || !strings.Contains(err.Error(), "backend executable is unavailable") {
		t.Fatalf("ResolveBackendExecutable() error = %v, want missing asset diagnostic", err)
	}
}

//	{
//	  責務: [TestLoadBackendManifestRejectsDuplicateBackendIDs: registry内のbackend_id重複を拒否することを検証する]
//	  処理: [1: 重複IDを含むmanifestを書く, 2: manifestを読み込む, 3: 重複ID errorを検証する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestLoadBackendManifestRejectsDuplicateBackendIDs(t *testing.T) {
	applicationRoot := t.TempDir()
	manifestPath := filepath.Join(applicationRoot, "backends", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("create backend directory: %v", err)
	}
	manifestJSON := `{"manifest_version":"1","parser_backend_contract_version":"1","backends":[{"backend_id":"duplicate","language":"python","path":"one"},{"backend_id":"duplicate","language":"python","path":"two"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o600); err != nil {
		t.Fatalf("write backend manifest: %v", err)
	}
	if _, err := LoadBackendManifest(applicationRoot); err == nil || !strings.Contains(err.Error(), "duplicate backend_id") {
		t.Fatalf("LoadBackendManifest() error = %v, want duplicate id rejection", err)
	}
}

//	{
//	  責務: [writeBackendTestFile: test用helper assetを親directory込みで作成する]
//	  処理: [1: 親directoryを作る, 2: file内容を書き込む]
//	  引数: [t: Go test runner, filePath: 作成するasset path]
//	  戻り値: []
//	}
func writeBackendTestFile(t *testing.T, filePath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create helper directory: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("test helper"), 0o600); err != nil {
		t.Fatalf("write helper file: %v", err)
	}
}
