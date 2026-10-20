package process

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
)

//	{
//	  責務: [ParserHost: 言語backend registryを保持し、要求に応じて登録済みhelperへCommon IR解析を委譲する]
//	  状態: [applicationRoot: parser assetとmanifestを含む配布root, manifest: 起動時に検証したbackend registry, operatingSystem: helperを選ぶOS, architecture: helperを選ぶCPU architecture, helperClient: JSON helper protocol host]
//	}
type ParserHost struct {
	applicationRoot string
	manifest        data.BackendManifest
	operatingSystem string
	architecture    string
	helperClient    HelperClient
}

//	{
//	  責務: [NewParserHost: manifestを検証し、現在の実行環境に対応するParser Hostを準備する]
//	  処理: [1: backend manifestを読み込む, 2: OS/architectureとtimeoutを設定する, 3: ParserHostを返す]
//	  引数: [applicationRoot: 配布root, timeout: helper一回あたりの実行時間上限]
//	  戻り値: [host: helperを起動するProcess service, error: manifest読込・検証に失敗した場合の内容]
//	}
func NewParserHost(applicationRoot string, timeout time.Duration) (ParserHost, error) {
	manifest, err := data.LoadBackendManifest(applicationRoot)
	if err != nil {
		return ParserHost{}, fmt.Errorf("load parser backend registry: %w", err)
	}
	if manifest.ParserBackendContractVersion != ParserBackendContractVersion {
		return ParserHost{}, fmt.Errorf("unsupported parser backend contract version %q (supported: %q)", manifest.ParserBackendContractVersion, ParserBackendContractVersion)
	}
	return ParserHost{
		applicationRoot: applicationRoot,
		manifest:        manifest,
		operatingSystem: runtime.GOOS,
		architecture:    runtime.GOARCH,
		helperClient:    NewHelperClient(timeout),
	}, nil
}

//	{
//	  責務: [Parse: backend IDと言語IDを照合し、manifestで選択したhelperからCommon IRを取得する]
//	  処理: [1: registryでbackend IDを検索する, 2: request languageを確認する, 3: OS/architecture向けhelper pathを解決する, 4: helper processでparseしてCommon IRを返す]
//	  引数: [ctx: 呼出元のcancel・deadline, backendID: manifest上のhelper ID, language: request language ID, source: 解析対象source, sourcePath: 任意のsource path]
//	  戻り値: [payload: 言語非依存Common IR, error: 未登録backend・path・process・protocolの正規化error]
//	}
func (host ParserHost) Parse(ctx context.Context, backendID string, language string, source string, sourcePath *string) (commonir.Payload, error) {
	backend, found := host.findBackend(backendID)
	if !found {
		return commonir.Payload{}, newBackendError("unsupported_language", fmt.Sprintf("Parser backend %q is not registered.", backendID), backendID, false)
	}
	if backend.Language != language {
		return commonir.Payload{}, newBackendError("unsupported_language", fmt.Sprintf("Parser backend %q handles %q, not %q.", backendID, backend.Language, language), backendID, false)
	}

	executablePath, err := data.ResolveBackendExecutable(host.applicationRoot, host.manifest, backendID, host.operatingSystem, host.architecture)
	if err != nil {
		return commonir.Payload{}, newBackendError("failure", err.Error(), backendID, false)
	}
	return host.helperClient.Parse(ctx, executablePath, backendID, language, source, sourcePath)
}

//	{
//	  責務: [findBackend: registry内からbackend_idに一致する定義を検索する]
//	  処理: [1: backend定義を順に確認する, 2: 最初のbackend ID一致を返す, 3: 見つからない場合はfalseを返す]
//	  引数: [backendID: 検索するparser helperの一意なID]
//	  戻り値: [backend: 一致したbackend定義, found: registryに登録済みならtrue]
//	}
func (host ParserHost) findBackend(backendID string) (data.Backend, bool) {
	for _, backend := range host.manifest.Backends {
		if backend.BackendID == backendID {
			return backend, true
		}
	}
	return data.Backend{}, false
}
