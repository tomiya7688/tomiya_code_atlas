package ui

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
)

const windowTitle = "Tomiya Code Atlas"

type pickerKind int

const (
	pickerSourceFile pickerKind = iota
	pickerSourceFolder
	pickerOutputFolder
)

//	{
//	  責務: [operationWidget: 図の表示名と利用者の選択状態をまとめる,],
//	  状態: [definition: Processが公開する処理IDと表示名, checked: 利用者が選んだ状態,],
//	}
type operationWidget struct {
	definition operationDefinition
	checked    widget.Bool
}

//	{
//	  責務: [pickerResult: ファイルやフォルダーの選択結果を画面更新へ渡す,],
//	  状態: [kind: 入力ファイル・入力フォルダー・保存先フォルダーの区別, path: 選択されたパス, err: 選択に失敗した理由,],
//	}
type pickerResult struct {
	kind pickerKind
	path string
	err  error
}

//	{
//	  責務: [analysisResult: 非同期解析の結果を画面更新へ渡す,],
//	  状態: [paths: Processが保存したファイル, err: 解析・構文解析・保存に失敗した理由,],
//	}
type analysisResult struct {
	paths []string
	err   error
}

//	{
//	  責務: [desktopApp: 解析画面の状態と利用者の操作を管理し、図の生成をProcessへ依頼する,],
//	  状態: [window: 描画先, explorer: OS標準の選択画面, generator: Process境界, editors: 入力と保存先, operations: 図の選択状態, analysis: キャンセル可能な非同期処理, results: 非同期処理の結果,],
//	}
type desktopApp struct {
	window             *app.Window
	explorer           *explorer.Explorer
	generator          DiagramGenerator
	analysisCancel     context.CancelFunc
	analysisDone       chan struct{}
	startupError       error
	workingDirectory   string
	sourceEditor       widget.Editor
	outputEditor       widget.Editor
	operations         []operationWidget
	chooseFileButton   widget.Clickable
	chooseInputButton  widget.Clickable
	chooseOutputButton widget.Clickable
	runButton          widget.Clickable
	outputList         widget.List
	shortcutTarget     int
	pickerResults      chan pickerResult
	analysisResults    chan analysisResult
	pickerPending      bool
	analysisPending    bool
	status             string
	diagnostic         string
	outputPaths        []string
}

//	{
//	  責務: [Run: Gioのイベントループを起動し、Windowsのデスクトップ画面を表示する,],
//	  処理: [1: OS専用スレッドでウィンドウを作る, 2: 利用者の操作と再描画を処理する, 3: 閉じる時に解析をキャンセルし、アプリケーションを終了する,],
//	  引数: [generator: 図の生成を依頼するProcessサービス, startupError: 起動時に解析機能を準備できなかった理由,],
//	}
func Run(generator DiagramGenerator, startupError error) {
	go func() {
		window := new(app.Window)
		window.Option(app.Title(windowTitle), app.Size(unit.Dp(980), unit.Dp(720)))
		application := newDesktopApp(window, generator, startupError)
		application.explorer = explorer.NewExplorer(window)
		theme := material.NewTheme()
		eventLoopError := runWindow(window, application, theme)
		application.cancelPendingAnalysis()
		if eventLoopError != nil {
			log.Printf("GUIを終了しました: %v", eventLoopError)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

//	{
//	  責務: [newDesktopApp: 入力欄・図の選択肢・結果表示を備えた画面状態を作る,],
//	  処理: [1: 入力欄を一行表示にする, 2: 利用者のホームフォルダーに保存先の初期値を設定する, 3: Processが対応する図を既定で選ぶ,],
//	  引数: [window: Gioのウィンドウ, generator: Processへ図の生成を依頼するサービス, startupError: 起動時の診断,],
//	  戻り値: [desktopApp: 初期化した画面状態,],
//	}
func newDesktopApp(window *app.Window, generator DiagramGenerator, startupError error) *desktopApp {
	workingDirectory, err := os.Getwd()
	if err != nil {
		workingDirectory = "."
	}
	userHomeDirectory, err := os.UserHomeDir()
	if err != nil {
		userHomeDirectory = ""
	}
	application := &desktopApp{
		window: window, generator: generator, startupError: startupError,
		workingDirectory: workingDirectory,
		pickerResults:    make(chan pickerResult, 1),
		analysisResults:  make(chan analysisResult, 1),
		status:           "入力と保存先を選び、作りたい結果にチェックを入れてください。",
	}
	application.sourceEditor.SingleLine = true
	application.outputEditor.SingleLine = true
	application.outputEditor.SetText(defaultOutputDirectory(userHomeDirectory, workingDirectory))
	for _, definition := range supportedOperationDefinitions() {
		operation := operationWidget{definition: definition}
		operation.checked.Value = definition.id == process.OperationCallGraph
		application.operations = append(application.operations, operation)
	}
	if startupError != nil {
		application.status = "Go版の解析機能を準備できませんでした。診断を確認してください。"
		application.diagnostic = startupError.Error()
	}
	return application
}

//	{
//	  責務: [defaultOutputDirectory: 利用者が書き込めるホームフォルダーを優先して出力先を決める,],
//	  処理: [1: ホームフォルダーが得られたらそこを使う, 2: 得られなければ実行時フォルダーを使う, 3: 製品専用の出力フォルダーを返す,],
//	  引数: [userHomeDirectory: 利用者のホームフォルダー, workingDirectory: アプリケーションの実行時フォルダー,],
//	  戻り値: [string: 図の保存先として画面に表示するフォルダー,],
//	}
func defaultOutputDirectory(userHomeDirectory, workingDirectory string) string {
	baseDirectory := strings.TrimSpace(userHomeDirectory)
	if baseDirectory == "" {
		baseDirectory = workingDirectory
	}
	return filepath.Join(baseDirectory, "Tomiya Code Atlas", "atlas-output")
}

//	{
//	  責務: [runWindow: Gioのウィンドウイベントを受け取り、選択・解析・描画を処理する,],
//	  処理: [1: Explorerへウィンドウイベントを渡す, 2: 非同期処理の結果を画面へ反映する, 3: 現在の画面を描画する,],
//	  引数: [window: Gioのウィンドウ, application: 画面の状態, theme: システムフォントに対応するテーマ,],
//	  戻り値: [error: ウィンドウ終了時にOSから返されたエラー,],
//	}
func runWindow(window *app.Window, application *desktopApp, theme *material.Theme) error {
	var operations op.Ops
	for {
		windowEvent := window.Event()
		// ExplorerはOS標準ダイアログを表示するため、Frame以外も含む全イベントを受け取る必要がある。
		application.explorer.ListenEvents(windowEvent)
		switch event := windowEvent.(type) {
		case app.DestroyEvent:
			return event.Err
		case app.FrameEvent:
			layoutContext := app.NewContext(&operations, event)
			application.consumeAsyncResults()
			application.layout(layoutContext, theme)
			event.Frame(layoutContext.Ops)
		}
	}
}

//	{
//	  責務: [selectedOperations: 利用者が選んだ処理IDをProcessの定めた順序で返す,],
//	  戻り値: [operations: 選択された処理IDの一覧,],
//	}
func (application *desktopApp) selectedOperations() []string {
	operations := make([]string, 0, len(application.operations))
	for index := range application.operations {
		operation := &application.operations[index]
		if operation.checked.Value {
			operations = append(operations, operation.definition.id)
		}
	}
	return operations
}

//	{
//	  責務: [startAnalysis: 入力を検証し、画面を応答可能なままProcessの解析を始める,],
//	  処理: [1: 多重実行・起動失敗・入力不足を判定する, 2: 解析中であることを画面に示す, 3: 完了結果を画面へ通知する,],
//	}
func (application *desktopApp) startAnalysis() {
	if application.analysisPending {
		return
	}
	if application.pickerPending {
		application.status = "ファイルやフォルダーの選択が終わってから解析を実行してください。"
		application.diagnostic = ""
		return
	}
	application.outputPaths = nil
	if application.startupError != nil || application.generator == nil {
		application.status = "解析処理を準備できません。"
		application.diagnostic = ""
		return
	}
	sourcePath, err := resolvePathFromDirectory(application.sourceEditor.Text(), application.workingDirectory)
	if err != nil {
		application.status = "解析するファイルまたはフォルダーのパスを確認してください。"
		application.diagnostic = err.Error()
		return
	}
	outputDirectory, err := resolvePathFromDirectory(application.outputEditor.Text(), application.workingDirectory)
	if err != nil {
		application.status = "結果の保存先フォルダーのパスを確認してください。"
		application.diagnostic = err.Error()
		return
	}
	request, err := buildDiagramRequest(
		sourcePath,
		outputDirectory,
		application.selectedOperations(),
	)
	if err != nil {
		application.status = "入力を確認してください。"
		application.diagnostic = err.Error()
		return
	}
	application.analysisPending = true
	application.diagnostic = ""
	application.status = "解析中です。完了すると生成したファイルの場所をここに表示します。"
	analysisContext, cancelAnalysis := context.WithCancel(context.Background())
	analysisDone := make(chan struct{})
	application.analysisCancel = cancelAnalysis
	application.analysisDone = analysisDone
	go func() {
		defer close(analysisDone)
		paths, generationError := generateDiagramsSafely(analysisContext, application.generator, request)
		application.analysisResults <- analysisResult{paths: paths, err: generationError}
		application.window.Invalidate()
	}()
}

//	{
//	  責務: [generateDiagramsSafely: 解析サービスのpanicを画面で表示できるエラーへ変換する,],
//	  処理: [1: Processへ図の生成を依頼する, 2: panicを記録して診断用のエラーを返す,],
//	  引数: [ctx: 解析を取り消すためのcontext, generator: 図の生成を依頼するProcessサービス, request: 検証済みの生成依頼,],
//	  戻り値: [paths: 保存された図のパス, error: 生成に失敗した理由または予期しないpanic,],
//	}
func generateDiagramsSafely(ctx context.Context, generator DiagramGenerator, request process.DiagramRequest) (paths []string, generationError error) {
	defer func() {
		if panicValue := recover(); panicValue != nil {
			log.Printf("図の生成処理で予期しないエラーが発生しました: %v", panicValue)
			paths = nil
			generationError = fmt.Errorf("解析処理が予期せず停止しました: %v", panicValue)
		}
	}()
	return generator.Generate(ctx, request)
}

//	{
//	  責務: [consumeAsyncResults: ファイル選択とProcess解析の結果を画面へ反映する,],
//	  処理: [1: 選択結果を入力欄へ反映する, 2: 解析結果と失敗理由を表示する,],
//	}
func (application *desktopApp) consumeAsyncResults() {
	select {
	case result := <-application.pickerResults:
		application.pickerPending = false
		application.applyPickerResult(result)
	default:
	}
	select {
	case result := <-application.analysisResults:
		application.analysisPending = false
		if application.analysisCancel != nil {
			application.analysisCancel()
			application.analysisCancel = nil
		}
		application.analysisDone = nil
		if result.err != nil {
			application.status = "解析に失敗しました。入力・parser・保存先を確認してください。"
			application.diagnostic = result.err.Error()
			application.outputPaths = result.paths
			return
		}
		application.outputPaths = result.paths
		application.status = fmt.Sprintf("解析が完了し、%d件を保存しました。", len(result.paths))
		application.diagnostic = ""
	default:
	}
}

//	{
//	  責務: [cancelPendingAnalysis: ウィンドウ終了時に解析を止め、外部parserの終了を待つ,],
//	  処理: [1: 解析contextをキャンセルする, 2: 最大5秒間だけ解析goroutineの終了を待つ,],
//	}
func (application *desktopApp) cancelPendingAnalysis() {
	if application.analysisCancel == nil {
		return
	}
	application.analysisCancel()
	application.analysisCancel = nil
	if application.analysisDone == nil {
		return
	}
	shutdownTimer := time.NewTimer(5 * time.Second)
	defer shutdownTimer.Stop()
	select {
	case <-application.analysisDone:
	case <-shutdownTimer.C:
		log.Printf("解析処理が終了しないため、GUIを終了します")
	}
	application.analysisDone = nil
}

//	{
//	  責務: [resolvePathFromDirectory: 入力パスを検証し、相対パスをGUI起動時の基準フォルダーから解決する,],
//	  処理: [1: 前後の空白と引用符を整理する, 2: ドライブ相対パスを拒否する, 3: 絶対パスを保ち、相対パスを基準フォルダーへ結合する,],
//	  引数: [path: ファイルまたはフォルダーのパス, workingDirectory: GUI起動時に記録した基準フォルダー,],
//	  戻り値: [string: 空欄、絶対パス、または基準フォルダーから解決したパス, error: 曖昧なドライブ相対パスの説明,],
//	}
func resolvePathFromDirectory(path, workingDirectory string) (string, error) {
	path = strings.TrimSpace(path)
	if len(path) >= 2 && strings.HasPrefix(path, "\"") && strings.HasSuffix(path, "\"") {
		path = strings.TrimSpace(path[1 : len(path)-1])
	}
	if path == "" {
		return "", nil
	}
	if !filepath.IsAbs(path) && filepath.VolumeName(path) != "" {
		return "", fmt.Errorf(`ドライブ相対パスは使えません。「C:\project\main.py」のような絶対パスを入力してください`)
	}
	if !filepath.IsAbs(path) && filepath.VolumeName(path) == "" && (strings.HasPrefix(path, `\`) || strings.HasPrefix(path, `/`)) {
		return "", fmt.Errorf(`ドライブ指定のないルート相対パスは使えません。「C:\project\main.py」のような絶対パスを入力してください`)
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	return filepath.Join(workingDirectory, path), nil
}
