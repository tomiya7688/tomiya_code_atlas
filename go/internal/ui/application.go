package ui

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//	{
//	  責務: [sourceFileFilterPattern: GUIで選択できる解析対象の拡張子を表す,],
//	}
const sourceFileFilterPattern = "*.py;*.gd;*.cs;*.cpp;*.cc;*.cxx;*.hpp;*.hxx;*.java;*.go;*.yml;*.yaml"

const smokeProfileCleanupAttempts = 40
const smokeProfileCleanupInterval = 250 * time.Millisecond

//go:embed web/*
var embeddedWebFiles embed.FS

//	{
//	  責務: [
//	    AnalysisGenerator: UIとProcess間で解析結果生成の要求と結果を受け渡す契約,
//	  ],
//	}
type AnalysisGenerator interface {
	Generate(context.Context, process.AnalysisRequest) ([]string, error)
}

//	{
//	  責務: [
//	    AnalysisResult: 解析完了時に画面へ表示する結果をまとめる,
//	  ],
//	  状態: [
//	    OutputPaths: 生成したfileのpath一覧,
//	    Message: 利用者向けの完了案内,
//	  ],
//	}
type AnalysisResult struct {
	OutputPaths []string `json:"outputPaths"`
	Message     string   `json:"message"`
}

//	{
//	  責務: [
//	    Application: 画面入力を受け、解析serviceとWails runtimeをつなぐ,
//	  ],
//	  状態: [generator: Processの解析結果生成契約, context: file dialogとwindow制御に使うruntime情報, smokeMode: CI smoke時にDOM読込後GUIを閉じる設定, smokeAnalysis: GUI起動後に実行するsmoke解析要求, smokeResult: smoke解析の終了結果をRunへ返すchannel]
//	}
type Application struct {
	generator     AnalysisGenerator
	context       context.Context
	smokeMode     bool
	smokeAnalysis *process.AnalysisRequest
	smokeResult   chan error
}

//	{
//	  責務: [
//	    Run: 埋込み画面を解析serviceへ接続し、desktop windowまたはsmoke解析を起動する,
//	  ],
//	}
func Run(generator AnalysisGenerator, smokeMode bool, smokeAnalysis *process.AnalysisRequest) (runError error) {
	if smokeAnalysis != nil && !smokeMode {
		return fmt.Errorf("GUI解析smoke testはsmoke modeでのみ実行できます")
	}
	webFiles, err := fs.Sub(embeddedWebFiles, "web")
	if err != nil {
		return fmt.Errorf("GUIの埋込みassetを開けません: %w", err)
	}
	webviewUserDataPath := ""
	if smokeMode {
		webviewUserDataPath, err = os.MkdirTemp("", "tomiya-code-atlas-webview-smoke-")
		if err != nil {
			return fmt.Errorf("smoke test用WebView2 profileを準備できません: %w", err)
		}
		defer func() {
			if cleanupError := removeSmokeProfileWithRetry(
				webviewUserDataPath,
				os.RemoveAll,
				time.Sleep,
			); cleanupError != nil {
				runError = errors.Join(runError, fmt.Errorf("smoke test用WebView2 profileを削除できません: %w", cleanupError))
			}
		}()
	} else {
		executablePath, executableError := os.Executable()
		if executableError != nil {
			return fmt.Errorf("実行ファイルの場所を取得できません: %w", executableError)
		}
		userConfigDirectory, configError := os.UserConfigDir()
		if configError != nil {
			return fmt.Errorf("利用者設定フォルダーを取得できません: %w", configError)
		}
		webviewUserDataPath, err = resolveWebviewUserDataPath(executablePath, userConfigDirectory)
		if err != nil {
			return fmt.Errorf("WebView2プロファイルのパスを決められません: %w", err)
		}
		if err := os.MkdirAll(webviewUserDataPath, 0o700); err != nil {
			return fmt.Errorf("WebView2プロファイルを準備できません: %w", err)
		}
	}
	application := &Application{generator: generator, smokeMode: smokeMode, smokeAnalysis: smokeAnalysis}
	if smokeAnalysis != nil {
		application.smokeResult = make(chan error, 1)
	}
	runError = wails.Run(&options.App{
		Title:     "Tomiya Code Atlas",
		Width:     1040,
		Height:    760,
		MinWidth:  860,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: webFiles,
		},
		OnStartup:  application.startup,
		OnDomReady: application.domReady,
		Bind:       []interface{}{application},
		Windows:    &windows.Options{WebviewUserDataPath: webviewUserDataPath},
	})
	if runError != nil || application.smokeResult == nil {
		return runError
	}
	return <-application.smokeResult
}

//	{
//	  責務: [resolveWebviewUserDataPath: 配布先ごとに分離したWebView2プロファイルの保存先を作る,]
//	  処理: [1: 実行ファイルのsymlinkを解決して配置フォルダーを決める, 2: OS別に配置先の実体を識別する, 3: 利用者設定内のprofile保存先を返す,]
//	  引数: [executablePath: 実行中のファイルのパス, userConfigDirectory: OSが返した利用者設定フォルダー,]
//	  戻り値: [string: 配置フォルダーを識別するprofileのパス, error: 必須pathまたは実行ファイルの実体を解決できない場合の説明,]
//	}
func resolveWebviewUserDataPath(executablePath, userConfigDirectory string) (string, error) {
	if strings.TrimSpace(executablePath) == "" || strings.TrimSpace(userConfigDirectory) == "" {
		return "", fmt.Errorf("実行ファイルと利用者設定フォルダーを指定してください")
	}
	resolvedExecutablePath, err := filepath.EvalSymlinks(executablePath)
	if err != nil {
		return "", fmt.Errorf("実行ファイルの実体パスを解決できません: %w", err)
	}
	executableDirectory, err := filepath.Abs(filepath.Dir(resolvedExecutablePath))
	if err != nil {
		return "", fmt.Errorf("実行ファイルのフォルダーを解決できません: %w", err)
	}
	configDirectory, err := filepath.Abs(userConfigDirectory)
	if err != nil {
		return "", fmt.Errorf("利用者設定フォルダーを解決できません: %w", err)
	}
	installIdentity := webviewInstallIdentity(executableDirectory)
	profileIdentity := sha256.Sum256([]byte(installIdentity))
	return filepath.Join(configDirectory, "Tomiya Code Atlas", "WebView2", hex.EncodeToString(profileIdentity[:])), nil
}

//	{
//	  責務: [removeSmokeProfileWithRetry: WebView2がprofileを解放するまで一時profile削除を再試行する,],
//	}
func removeSmokeProfileWithRetry(profilePath string, removeAll func(string) error, sleep func(time.Duration)) error {
	var cleanupError error
	for attempt := 0; attempt < smokeProfileCleanupAttempts; attempt++ {
		cleanupError = removeAll(profilePath)
		if cleanupError == nil {
			return nil
		}
		if attempt < smokeProfileCleanupAttempts-1 {
			sleep(smokeProfileCleanupInterval)
		}
	}
	return cleanupError
}

//	{
//	  責務: [
//	    startup: runtime contextを保存してfile dialogとwindow操作へ渡す,
//	  ],
//	}
func (application *Application) startup(runtimeContext context.Context) {
	application.context = runtimeContext
}

//	{
//	  責務: [
//	    domReady: smoke modeで埋込み画面を読み込み、必要ならGUIのAnalyze bindingを実行して終了する,
//	  ],
//	}
func (application *Application) domReady(runtimeContext context.Context) {
	if !application.smokeMode {
		return
	}
	wailsruntime.WindowHide(runtimeContext)
	if application.smokeAnalysis != nil {
		go func() {
			_, smokeError := application.Analyze(
				application.smokeAnalysis.SourcePath,
				application.smokeAnalysis.Operations,
				application.smokeAnalysis.OutputDirectory,
			)
			application.smokeResult <- smokeError
			wailsruntime.Quit(runtimeContext)
		}()
		return
	}
	time.AfterFunc(1500*time.Millisecond, func() {
		wailsruntime.Quit(runtimeContext)
	})
}

//	{
//	  責務: [OpenSourceFile: 対応するソースを選ぶダイアログを開く,],
//	  戻り値: [path: 選択されたファイル、キャンセル時は空文字, error: ダイアログの失敗,],
//	}
func (application *Application) OpenSourceFile() (string, error) {
	return wailsruntime.OpenFileDialog(application.runtimeContext(), wailsruntime.OpenDialogOptions{
		Title: "解析するソースファイルを選択",
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "対応するソース・GitHub Actions workflowファイル",
			Pattern:     sourceFileFilterPattern,
		}},
	})
}

//	{
//	  責務: [OpenProjectFolder: 解析対象のプロジェクトフォルダーを選ぶダイアログを開く,],
//	  戻り値: [path: 選択されたフォルダー、キャンセル時は空文字, error: ダイアログの失敗,],
//	}
func (application *Application) OpenProjectFolder() (string, error) {
	return wailsruntime.OpenDirectoryDialog(application.runtimeContext(), wailsruntime.OpenDialogOptions{
		Title: "解析するプロジェクトフォルダーを選択",
	})
}

//	{
//	  責務: [ChooseOutputFolder: 解析結果の保存先を選ぶダイアログを開く,],
//	  戻り値: [path: 選択されたフォルダー、キャンセル時は空文字, error: ダイアログの失敗,],
//	}
func (application *Application) ChooseOutputFolder() (string, error) {
	return wailsruntime.OpenDirectoryDialog(application.runtimeContext(), wailsruntime.OpenDialogOptions{
		Title: "解析結果の保存フォルダーを選択",
	})
}

//	{
//	  責務: [GetCompatibleOperations: 選択した入力で実行できる操作を画面へ返す]
//	  引数: [sourcePath: fileまたはproject folderのpath]
//	  戻り値: [operations: 入力に対応する操作ID, error: 入力調査が失敗した理由]
//	}
func (application *Application) GetCompatibleOperations(sourcePath string) ([]string, error) {
	provider, supportsInputCompatibility := application.generator.(interface {
		CompatibleOperations(string) ([]string, error)
	})
	if !supportsInputCompatibility {
		return process.SupportedOperationIDs(), nil
	}
	return provider.CompatibleOperations(sourcePath)
}

//	{
//	  責務: [
//	    Analyze: 選択された複数の解析操作をProcess serviceへ渡し、生成fileのpathを画面に返す,
//	  ],
//	  引数: [
//	    sourcePath: 解析するfileまたはproject folder,
//	    operations: 生成する成果物のID一覧,
//	    outputDirectory: 生成fileの保存先,
//	  ],
//	  戻り値: [
//	    AnalysisResult: 生成file pathと完了案内,
//	    error: 入力選択または解析で失敗した理由,
//	  ],
//	}
func (application *Application) Analyze(sourcePath string, operations []string, outputDirectory string) (AnalysisResult, error) {
	if application.generator == nil {
		return AnalysisResult{}, fmt.Errorf("解析serviceを利用できません")
	}
	if strings.TrimSpace(sourcePath) == "" {
		return AnalysisResult{}, fmt.Errorf("解析するファイルまたはプロジェクトフォルダーを選んでください")
	}
	if len(operations) == 0 {
		return AnalysisResult{}, fmt.Errorf("作りたい成果物を一つ以上選んでください")
	}
	seenOperations := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		if !process.IsSupportedOperation(operation) {
			return AnalysisResult{}, fmt.Errorf("未対応の操作です: %s", operation)
		}
		if _, duplicate := seenOperations[operation]; duplicate {
			return AnalysisResult{}, fmt.Errorf("同じ操作が重複して選ばれています: %s", operation)
		}
		seenOperations[operation] = struct{}{}
	}
	if strings.TrimSpace(outputDirectory) == "" {
		return AnalysisResult{}, fmt.Errorf("結果の保存フォルダーを選んでください")
	}

	outputPaths, err := application.generator.Generate(application.runtimeContext(), process.AnalysisRequest{
		SourcePath:      sourcePath,
		Operations:      operations,
		OutputDirectory: outputDirectory,
	})
	if err != nil {
		return AnalysisResult{}, err
	}
	return AnalysisResult{
		OutputPaths: outputPaths,
		Message:     fmt.Sprintf("解析が完了しました。%d件の成果物を保存しました。", len(outputPaths)),
	}, nil
}

//	{
//	  責務: [runtimeContext: 起動済みの画面操作には保存したcontextを渡し、単体テスト時は背景contextを返す,],
//	  戻り値: [context: 画面操作または解析に渡すcontext,],
//	}
func (application *Application) runtimeContext() context.Context {
	if application.context == nil {
		return context.Background()
	}
	return application.context
}
