package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const supportedManifestVersion = "1"

//	{
//	  責務: [BackendManifest: 配布対象のparser helperと共通protocol versionを保持する]
//	  状態: [ManifestVersion: manifest形式のversion, ParserBackendContractVersion: helper通信契約のversion, Backends: helper定義]
//	}
type BackendManifest struct {
	ManifestVersion              string    `json:"manifest_version"`
	ParserBackendContractVersion string    `json:"parser_backend_contract_version"`
	Backends                     []Backend `json:"backends"`
}

//	{
//	  責務: [Backend: 一つのparser helperの識別情報と実行pathを保持する]
//	  状態: [BackendID: helperの一意な識別子, Language: 対象言語, Path: 既定path, Platforms: OS/architecture別path]
//	}
type Backend struct {
	BackendID string            `json:"backend_id"`
	Language  string            `json:"language"`
	Kind      string            `json:"kind"`
	Path      string            `json:"path"`
	Platforms map[string]string `json:"platforms"`
}

//	{
//	  責務: [LoadBackendManifest: 配布rootのmanifestを読み、versionとbackend識別子を検証する]
//	  処理: [1: manifest fileを開く, 2: JSONをdecodeする, 3: 末尾の余分な値と必須項目を検証する, 4: 検証済みmanifestを返す]
//	  引数: [applicationRoot: backends directoryを含む配布root]
//	  戻り値: [manifest: helper registry, error: 読込または契約違反の内容]
//	}
func LoadBackendManifest(applicationRoot string) (BackendManifest, error) {
	manifestPath := filepath.Join(applicationRoot, "backends", "manifest.json")
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return BackendManifest{}, fmt.Errorf("open backend manifest: %w", err)
	}
	defer manifestFile.Close()

	var manifest BackendManifest
	decoder := json.NewDecoder(manifestFile)
	if err := decoder.Decode(&manifest); err != nil {
		return BackendManifest{}, fmt.Errorf("decode backend manifest: %w", err)
	}
	if err := ensureManifestHasSingleJSONValue(decoder); err != nil {
		return BackendManifest{}, err
	}
	if err := validateBackendManifest(manifest); err != nil {
		return BackendManifest{}, err
	}
	return manifest, nil
}

//	{
//	  責務: [ensureManifestHasSingleJSONValue: manifestの末尾に別JSON値が続かないことを検証する]
//	  処理: [1: 二つ目のJSON値をdecodeする, 2: EOFなら成功し、余分な値または不正形式ならerrorにする]
//	  引数: [decoder: 一つ目のJSON値を読み終えたdecoder]
//	  戻り値: [error: 末尾に余分な値または不正データがある場合の内容]
//	}
func ensureManifestHasSingleJSONValue(decoder *json.Decoder) error {
	var trailingValue json.RawMessage
	if err := decoder.Decode(&trailingValue); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("backend manifest contains more than one JSON value")
		}
		return fmt.Errorf("read backend manifest trailer: %w", err)
	}
	return nil
}

//	{
//	  責務: [validateBackendManifest: manifest versionとhelper registryの不変条件を検証する]
//	  処理: [1: supported versionと非空registryを確認する, 2: 各helperの必須項目と実行pathを確認する, 3: backend_idの重複を拒否する]
//	  引数: [manifest: decode済みの配布manifest]
//	  戻り値: [error: 契約違反があればその内容]
//	}
func validateBackendManifest(manifest BackendManifest) error {
	if manifest.ManifestVersion != supportedManifestVersion {
		return fmt.Errorf("unsupported backend manifest version %q", manifest.ManifestVersion)
	}
	if strings.TrimSpace(manifest.ParserBackendContractVersion) == "" {
		return errors.New("backend manifest has no parser backend contract version")
	}
	if len(manifest.Backends) == 0 {
		return errors.New("backend manifest has no backends")
	}

	seenBackendIDs := make(map[string]struct{}, len(manifest.Backends))
	for _, backend := range manifest.Backends {
		if strings.TrimSpace(backend.BackendID) == "" || strings.TrimSpace(backend.Language) == "" {
			return errors.New("backend manifest contains a backend without an id or language")
		}
		if _, exists := seenBackendIDs[backend.BackendID]; exists {
			return fmt.Errorf("backend manifest contains duplicate backend_id %q", backend.BackendID)
		}
		seenBackendIDs[backend.BackendID] = struct{}{}
		if backend.Path == "" && len(backend.Platforms) == 0 {
			return fmt.Errorf("backend %q has no executable path", backend.BackendID)
		}
		for target, path := range backend.Platforms {
			if strings.TrimSpace(target) == "" || strings.TrimSpace(path) == "" {
				return fmt.Errorf("backend %q has an empty platform or executable path", backend.BackendID)
			}
		}
	}
	return nil
}

//	{
//	  責務: [ResolveBackendExecutable: backend_idと実行環境から配布root内の実在するhelperを解決する]
//	  処理: [1: backend_idを検索する, 2: OS/architecture固有pathを優先し既定pathへfallbackする, 3: 配布root外へのpath逸脱を拒否する, 4: 通常fileの絶対pathを返す]
//	  引数: [applicationRoot: 配布root, manifest: 検証済みregistry, backendID: 起動するhelper, operatingSystem: runtime.GOOS相当, architecture: runtime.GOARCH相当]
//	  戻り値: [executablePath: 実在するhelperの絶対path, error: 未登録・未配布・path違反の内容]
//	}
func ResolveBackendExecutable(applicationRoot string, manifest BackendManifest, backendID string, operatingSystem string, architecture string) (string, error) {
	for _, backend := range manifest.Backends {
		if backend.BackendID != backendID {
			continue
		}
		relativePath, exists := backend.Platforms[operatingSystem+"/"+architecture]
		if !exists {
			relativePath = backend.Path
		}
		if relativePath == "" {
			return "", fmt.Errorf("backend %q has no executable for %s/%s", backendID, operatingSystem, architecture)
		}
		return resolveManifestPath(applicationRoot, relativePath)
	}
	return "", fmt.Errorf("backend %q is not registered", backendID)
}

//	{
//	  責務: [resolveManifestPath: manifest pathを配布directory内に限定して実在確認する]
//	  処理: [1: backends rootと候補pathを絶対化する, 2: 相対pathの逸脱を拒否する, 3: 通常fileであることを確認して返す]
//	  引数: [applicationRoot: 配布root, relativePath: manifest内の相対path]
//	  戻り値: [resolvedPath: helper fileの絶対path, error: path不正またはfile未配布の内容]
//	}
func resolveManifestPath(applicationRoot string, relativePath string) (string, error) {
	backendRoot, err := filepath.Abs(filepath.Join(applicationRoot, "backends"))
	if err != nil {
		return "", fmt.Errorf("resolve backend asset root: %w", err)
	}
	if filepath.IsAbs(relativePath) {
		return "", errors.New("backend executable path must be relative to the backends directory")
	}
	resolvedPath, err := filepath.Abs(filepath.Join(backendRoot, filepath.FromSlash(relativePath)))
	if err != nil {
		return "", fmt.Errorf("resolve backend executable path: %w", err)
	}
	relativeToRoot, err := filepath.Rel(backendRoot, resolvedPath)
	if err != nil || relativeToRoot == ".." || strings.HasPrefix(relativeToRoot, ".."+string(filepath.Separator)) {
		return "", errors.New("backend executable path escapes the backends directory")
	}
	fileInfo, err := os.Stat(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("backend executable is unavailable at %q: %w", resolvedPath, err)
	}
	if !fileInfo.Mode().IsRegular() {
		return "", fmt.Errorf("backend executable is not a file: %q", resolvedPath)
	}
	return resolvedPath, nil
}
