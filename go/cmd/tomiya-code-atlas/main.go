package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
	"github.com/tomiya7688/tomiya_code_atlas/internal/ui"
)

var version = "dev"

//	{
//	  責務: [exit status contract: CLI成功・実行失敗・引数誤りのprocess exit codeを固定する]
//	}
const (
	exitSuccess       = 0
	exitAnalysisError = 1
	exitUsageError    = 2
)

//	{
//	  責務: [main: 標準CLI引数と標準streamでcommand dispatcherを起動する]
//	}
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

//	{
//	  責務: [
//	    run: 標準引数と出力をGUI / CLI dispatcherへ渡す,
//	  ],
//	}
func run(args []string, stdout, stderr io.Writer) int {
	return runWithGUI(args, stdout, stderr, launchGUI)
}

//	{
//	  責務: [
//	    runWithGUI: 引数なしはGUIを開き、command指定時は既存CLIへ振り分ける,
//	  ],
//	}
func runWithGUI(args []string, stdout, stderr io.Writer, launch func(bool) error) int {
	if len(args) == 0 {
		if err := launch(false); err != nil {
			fmt.Fprintf(stderr, "[エラー] GUIを起動できません: %s\n", err)
			return exitAnalysisError
		}
		return exitSuccess
	}

	switch args[0] {
	case "--gui-smoke":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "[エラー] --gui-smoke に追加の引数は指定できません")
			return exitUsageError
		}
		if err := launch(true); err != nil {
			fmt.Fprintf(stderr, "[エラー] GUI起動smoke testに失敗しました: %s\n", err)
			return exitAnalysisError
		}
		return exitSuccess
	case "--help", "-h", "help":
		writeHelp(stdout)
		return exitSuccess
	case "--version", "-V", "version":
		fmt.Fprintf(stdout, "Tomiya Code Atlas (Go) %s\n", version)
		return exitSuccess
	case "analyze":
		if len(args) > 1 && (args[1] == "--help" || args[1] == "-h") {
			writeAnalyzeHelp(stdout)
			return exitSuccess
		}
		options, err := parseAnalyzeArguments(args[1:])
		if err != nil {
			fmt.Fprintf(stderr, "[エラー] %s\n", err)
			fmt.Fprintln(stderr, "使い方は `tomiya-code-atlas analyze --help` で確認してください。")
			return exitUsageError
		}
		if err := runAnalyze(options); err != nil {
			fmt.Fprintf(stderr, "[エラー] 解析に失敗しました: %s\n", err)
			return exitAnalysisError
		}
		fmt.Fprintf(stdout, "解析結果を保存しました: %s\n", options.outputDirectory)
		return exitSuccess
	default:
		fmt.Fprintf(stderr, "[エラー] まだ利用できないコマンドです: %s\n", args[0])
		fmt.Fprintln(stderr, "利用可能なコマンドは --help で確認してください。")
		return exitUsageError
	}
}

//	{
//	  責務: [
//	    launchGUI: CLIと共有する解析serviceをGUIへ渡してdesktop windowを起動する,
//	  ],
//	}
func launchGUI(smokeMode bool) error {
	service, err := newAnalysisService()
	if err != nil {
		return err
	}
	return ui.Run(service, smokeMode)
}

//	{
//	  責務: [
//	    newAnalysisService: 配布物と実行directoryからCLI / GUIが共有する解析serviceを準備する,
//	  ],
//	}
func newAnalysisService() (process.AnalysisService, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return process.AnalysisService{}, fmt.Errorf("実行fileのpathを確認できません: %w", err)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return process.AnalysisService{}, fmt.Errorf("実行directoryを確認できません: %w", err)
	}
	applicationRoot, err := data.FindApplicationRoot(executablePath, workingDirectory)
	if err != nil {
		return process.AnalysisService{}, err
	}
	service, err := process.NewAnalysisService(applicationRoot, 30*time.Second)
	if err != nil {
		return process.AnalysisService{}, fmt.Errorf("parser helperを準備できません: %w", err)
	}
	return service, nil
}

//	{
//	  責務: [writeHelp: 利用可能commandと詳細helpの開き方を日本語で表示する]
//	  引数: [w: help本文を書き出すstream]
//	}
func writeHelp(w io.Writer) {
	fmt.Fprintln(w, "Tomiya Code Atlas (Go)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "使用方法: tomiya-code-atlas [コマンド]")
	fmt.Fprintln(w, "コマンドを指定せずに起動するとGUIが開きます。")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "コマンド:")
	fmt.Fprintln(w, "  --help, -h     この案内を表示")
	fmt.Fprintln(w, "  --version, -V  バージョンを表示")
	fmt.Fprintln(w, "  analyze        ソースを解析して選択した結果を保存")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "詳しい使い方: tomiya-code-atlas analyze --help")
}

//	{
//	  責務: [analyzeOptions: analyze commandの入力source、operation、出力先を保持する]
//	  状態: [sourcePath: 解析するsource file, operations: 実行するanalysis operation, outputDirectory: 生成fileの保存先]
//	}
type analyzeOptions struct {
	sourcePath      string
	operations      []string
	outputDirectory string
}

//	{
//	  責務: [parseAnalyzeArguments: analyze commandの引数を検証してCLI契約へ変換する]
//	  処理: [1: positional sourceとoperation/output optionを読む, 2: 必須値と重複を検証する, 3: 検証済みoptionを返す]
//	  引数: [arguments: analyze command名の後に指定された引数]
//	  戻り値: [options: 実行option, error: 未知option・不足値・重複・不正operationの説明]
//	}
func parseAnalyzeArguments(arguments []string) (analyzeOptions, error) {
	options := analyzeOptions{}
	seenOperations := map[string]struct{}{}
	outputDirectorySet := false
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		switch argument {
		case "--operation":
			if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "--") {
				return analyzeOptions{}, fmt.Errorf("--operation の値を指定してください")
			}
			index++
			operation := arguments[index]
			if !process.IsSupportedOperation(operation) {
				return analyzeOptions{}, fmt.Errorf("未対応のoperationです: %s（選択可能: %s）", operation, strings.Join(process.SupportedOperationIDs(), ", "))
			}
			if _, duplicate := seenOperations[operation]; duplicate {
				return analyzeOptions{}, fmt.Errorf("operationを重複して指定しています: %s", operation)
			}
			seenOperations[operation] = struct{}{}
			options.operations = append(options.operations, operation)
		case "--output-dir":
			if outputDirectorySet {
				return analyzeOptions{}, fmt.Errorf("--output-dir は一度だけ指定してください")
			}
			if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "--") {
				return analyzeOptions{}, fmt.Errorf("--output-dir の保存先を指定してください")
			}
			index++
			options.outputDirectory = arguments[index]
			outputDirectorySet = true
		default:
			if strings.HasPrefix(argument, "-") {
				return analyzeOptions{}, fmt.Errorf("認識できないオプションです: %s", argument)
			}
			if options.sourcePath != "" {
				return analyzeOptions{}, fmt.Errorf("解析するsource fileまたはproject folderは一つだけ指定してください")
			}
			options.sourcePath = argument
		}
	}
	if options.sourcePath == "" {
		return analyzeOptions{}, fmt.Errorf("解析するsource fileまたはproject folderを指定してください")
	}
	if len(options.operations) == 0 {
		return analyzeOptions{}, fmt.Errorf("--operation を1つ以上指定してください")
	}
	if options.outputDirectory == "" {
		return analyzeOptions{}, fmt.Errorf("--output-dir で保存先フォルダーを指定してください")
	}
	return options, nil
}

//	{
//	  責務: [runAnalyze: 指定source fileまたはproject folderをparserで解析し、選択した製品出力を保存する]
//	  処理: [1: parser hostを解決する, 2: Process serviceへ解析とCommon IR統合を依頼する, 3: 選択operationの成果物をoutput directoryへ保存する]
//	  引数: [options: 検証済みsource・operation・output directory]
//	  戻り値: [error: source読込・parser・generator・renderer・file保存に失敗した場合の内容]
//	}
func runAnalyze(options analyzeOptions) error {
	service, err := newAnalysisService()
	if err != nil {
		return err
	}
	_, err = service.Generate(context.Background(), process.AnalysisRequest{
		SourcePath: options.sourcePath, Operations: options.operations, OutputDirectory: options.outputDirectory,
	})
	return err
}

//	{
//	  責務: [writeAnalyzeHelp: analyzeの入力・operation複数指定・出力先を日本語で案内する]
//	}
func writeAnalyzeHelp(w io.Writer) {
	fmt.Fprintln(w, "使用方法: tomiya-code-atlas analyze <ソースファイルまたはプロジェクトフォルダー> --operation <解析結果> [--operation <解析結果> ...] --output-dir <保存先フォルダー>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "--operationは作成する解析結果ごとに繰り返し指定できます。")
	fmt.Fprintln(w, "入力にフォルダーを指定すると、対応するソースファイルを配下から再帰的に探します。")
	fmt.Fprintln(w, "各ファイルを拡張子に対応する解析補助プログラムで調べ、選択した結果を保存します。")
	fmt.Fprintln(w, "commentsは元ソースを変更せず、commentsフォルダーへコメントを追加したコピーを保存します。")
	for _, operation := range process.SupportedOperationIDs() {
		fmt.Fprintf(w, "  %-14s %s\n", operation, analysisOperationDescription(operation))
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "例: tomiya-code-atlas analyze src --operation call-graph --operation class-diagram --output-dir atlas-output")
}

//	{
//	  責務: [analysisOperationDescription: CLI operation IDを利用者向けの日本語説明へ変換する]
//	  引数: [operation: SupportedOperationIDsに登録されたoperation]
//	  戻り値: [string: operationが生成する結果の日本語説明]
//	}
func analysisOperationDescription(operation string) string {
	switch operation {
	case "call-graph":
		return "呼び出し図を生成"
	case "class-diagram":
		return "クラス図を生成"
	case "comments":
		return "コメントを追加したsource copyを保存"
	case "static-spec":
		return "静的仕様書を生成"
	case "design-quality":
		return "設計品質レポートを生成"
	default:
		return "未対応の解析結果"
	}
}
