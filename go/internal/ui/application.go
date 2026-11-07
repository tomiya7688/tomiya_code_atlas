package ui

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//	{
//	  責務: [sourceFileFilterPattern: GUIで選択できる解析対象の拡張子を表す,],
//	}
const sourceFileFilterPattern = "*.py;*.gd;*.cs;*.cpp;*.cc;*.cxx;*.hpp;*.hxx;*.java;*.go"

//go:embed web/*
var embeddedWebFiles embed.FS

//	{
//	  責務: [
//	    DiagramGenerator: UIとProcess間で図生成の要求と結果を受け渡す契約,
//	  ],
//	}
type DiagramGenerator interface {
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
//	  状態: [
//	    generator: Processの図生成契約,
//	    context: file dialogとwindow制御に使うruntime情報,
//	    smokeMode: CIの起動確認後に画面を閉じる設定,
//	  ],
//	}
type Application struct {
	generator DiagramGenerator
	context   context.Context
	smokeMode bool
}

//	{
//	  責務: [
//	    Run: 埋込み画面を解析serviceへ接続し、desktop windowを起動する,
//	  ],
//	}
func Run(generator DiagramGenerator, smokeMode bool) error {
	webFiles, err := fs.Sub(embeddedWebFiles, "web")
	if err != nil {
		return fmt.Errorf("GUIの埋込みassetを開けません: %w", err)
	}
	application := &Application{generator: generator, smokeMode: smokeMode}
	return wails.Run(&options.App{
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
	})
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
//	    domReady: CI smoke modeではruntime初期化後にGUIを隠して終了する,
//	  ],
//	}
func (application *Application) domReady(runtimeContext context.Context) {
	if !application.smokeMode {
		return
	}
	wailsruntime.WindowHide(runtimeContext)
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
			DisplayName: "対応するソースファイル",
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
//	  責務: [
//	    Analyze: 選択された複数図をProcess serviceへ渡し、生成pathを画面に返す,
//	  ],
//	  引数: [
//	    sourcePath: 解析するfileまたはproject folder,
//	    operations: 生成する図のID一覧,
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
		return AnalysisResult{}, fmt.Errorf("作りたい図を一つ以上選んでください")
	}
	seenOperations := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		if operation != "call-graph" && operation != "class-diagram" {
			return AnalysisResult{}, fmt.Errorf("未対応の図です: %s", operation)
		}
		if _, duplicate := seenOperations[operation]; duplicate {
			return AnalysisResult{}, fmt.Errorf("同じ図が重複して選ばれています: %s", operation)
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
		Message:     fmt.Sprintf("解析が完了しました。%d件の図を保存しました。", len(outputPaths)),
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
