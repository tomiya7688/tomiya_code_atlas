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
)

var version = "dev"

//	{
//	  責務: [終了状態の契約: CLIの成功・実行失敗・引数誤りを異なる終了コードで表す,],
//	}
const (
	exitSuccess       = 0
	exitAnalysisError = 1
	exitUsageError    = 2
)

//	{
//	  責務: [main: 標準CLI引数と入出力先を使ってコマンドを振り分ける,],
//	}
func main() {
	if desktopGUIEnabled && (len(os.Args) == 1 || (len(os.Args) == 2 && os.Args[1] == "gui")) {
		runDesktopGUI()
		return
	}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

//	{
//	  責務: [run: 最上位のコマンドを検証し、ヘルプ・バージョン表示・解析へ振り分ける,],
//	  処理: [1: コマンド名を判定する, 2: 該当する処理を呼び出す, 3: 結果に応じた終了コードを返す,],
//	  引数: [args: CLIで指定されたコマンドライン引数, stdout: 正常時の出力先, stderr: エラー時の出力先,],
//	  戻り値: [int: 成功時0・解析失敗時1・引数誤り時2のプロセス終了コード,],
//	}
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeHelp(stdout)
		return exitSuccess
	}

	switch args[0] {
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
			fmt.Fprintln(stderr, "使い方は `tomiya-code-atlas-cli.exe analyze --help` で確認してください。")
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
//	  責務: [writeHelp: 利用できるコマンドと詳しい案内の表示方法を説明する,],
//	  引数: [w: 案内を書き出す出力先,],
//	}
func writeHelp(w io.Writer) {
	fmt.Fprintln(w, "Tomiya Code Atlas (Go)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "使用方法: tomiya-code-atlas-cli.exe [コマンド]")
	if desktopGUIEnabled {
		fmt.Fprintln(w, "Windows版GUI: tomiya-code-atlas.exe（引数なしで起動）")
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "コマンド:")
	fmt.Fprintln(w, "  --help, -h     この案内を表示")
	fmt.Fprintln(w, "  --version, -V  バージョンを表示")
	fmt.Fprintln(w, "  analyze        ソースを解析して図を保存")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "詳しい使い方は、tomiya-code-atlas-cli.exe analyze --help で確認してください。")
}

//	{
//	  責務: [analyzeOptions: analyzeコマンドの入力・生成する図・保存先を保持する,],
//	  状態: [sourcePath: 解析するソースファイルまたはフォルダー, operations: 生成する図の処理ID, outputDirectory: 生成ファイルの保存先,],
//	}
type analyzeOptions struct {
	sourcePath      string
	operations      []string
	outputDirectory string
}

//	{
//	  責務: [supportedOperations: Go CLIで生成できる図の処理IDを列挙する,],
//	}
var supportedOperations = loadSupportedOperations()

//	{
//	  責務: [loadSupportedOperations: Processの対応一覧をCLI引数の検証に使う集合へ変換する,],
//	  処理: [1: Processから対応する処理IDを取得する, 2: IDをキーとする集合を作る,],
//	  戻り値: [map[string]struct{}: 対応する処理IDをキーに持つ集合,],
//	}
func loadSupportedOperations() map[string]struct{} {
	operations := make(map[string]struct{})
	for _, operation := range process.SupportedDiagramOperations() {
		operations[operation] = struct{}{}
	}
	return operations
}

//	{
//	  責務: [parseAnalyzeArguments: analyzeコマンドの引数を検証し、実行用の設定へ変換する,],
//	  処理: [1: ソースと図の種類・保存先を読み取る, 2: 必須値・重複・未対応の値を検証する, 3: 検証済みの設定を返す,],
//	  引数: [arguments: analyzeの後に指定された引数,],
//	  戻り値: [options: 検証済みの実行設定, error: 不明な指定・不足・重複・未対応の値の説明,],
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
				return analyzeOptions{}, fmt.Errorf("--operation に図の種類を指定してください")
			}
			index++
			operation := arguments[index]
			if _, supported := supportedOperations[operation]; !supported {
				availableOperations := strings.Join(process.SupportedDiagramOperations(), ", ")
				return analyzeOptions{}, fmt.Errorf("未対応の図です: %s（選択可能: %s）", operation, availableOperations)
			}
			if _, duplicate := seenOperations[operation]; duplicate {
				return analyzeOptions{}, fmt.Errorf("同じ図を重複して指定しています: %s", operation)
			}
			seenOperations[operation] = struct{}{}
			options.operations = append(options.operations, operation)
		case "--output-dir":
			if outputDirectorySet {
				return analyzeOptions{}, fmt.Errorf("--output-dir は一度だけ指定してください")
			}
			if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "--") {
				return analyzeOptions{}, fmt.Errorf("--output-dir に保存先フォルダーを指定してください")
			}
			index++
			options.outputDirectory = arguments[index]
			outputDirectorySet = true
		default:
			if strings.HasPrefix(argument, "-") {
				return analyzeOptions{}, fmt.Errorf("認識できないオプションです: %s", argument)
			}
			if options.sourcePath != "" {
				return analyzeOptions{}, fmt.Errorf("解析するソースファイルまたはプロジェクトフォルダーは一つだけ指定してください")
			}
			options.sourcePath = argument
		}
	}
	if options.sourcePath == "" {
		return analyzeOptions{}, fmt.Errorf("解析するソースファイルまたはプロジェクトフォルダーを指定してください")
	}
	if len(options.operations) == 0 {
		return analyzeOptions{}, fmt.Errorf("--operation で図を1つ以上指定してください")
	}
	if options.outputDirectory == "" {
		return analyzeOptions{}, fmt.Errorf("--output-dir で保存先フォルダーを指定してください")
	}
	return options, nil
}

//	{
//	  責務: [runAnalyze: 指定されたソースを解析し、選んだ図を保存先へ出力する,],
//	  処理: [1: 解析補助プログラムを含むサービスを準備する, 2: Processへ解析とIRの統合を依頼する, 3: 選んだ図をMermaid形式で生成して保存する,],
//	  引数: [options: 検証済みの入力・図の種類・保存先,],
//	  戻り値: [error: ソース読込・構文解析・図の生成・保存に失敗した場合の説明,],
//	}
func runAnalyze(options analyzeOptions) error {
	service, err := newDiagramService()
	if err != nil {
		return err
	}
	_, err = service.Generate(context.Background(), process.DiagramRequest{
		SourcePath: options.sourcePath, Operations: options.operations, OutputDirectory: options.outputDirectory,
	})
	return err
}

//	{
//	  責務: [newDiagramService: 配布ファイルの場所を特定し、図生成を担うProcessサービスを作る,],
//	  処理: [1: 実行ファイルと起動フォルダーを調べる, 2: 配布ルートを特定する, 3: 解析補助プログラムを含むサービスを返す,],
//	  戻り値: [service: 図生成を調整するProcessサービス, error: 実行場所や解析補助プログラムを解決できない場合の説明,],
//	}
func newDiagramService() (process.DiagramService, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return process.DiagramService{}, fmt.Errorf("実行ファイルの場所を確認できません: %w", err)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return process.DiagramService{}, fmt.Errorf("実行時のフォルダーを確認できません: %w", err)
	}
	applicationRoot, err := data.FindApplicationRoot(executablePath, workingDirectory)
	if err != nil {
		return process.DiagramService{}, err
	}
	service, err := process.NewDiagramService(applicationRoot, 30*time.Second)
	if err != nil {
		return process.DiagramService{}, fmt.Errorf("解析補助プログラムを準備できません: %w", err)
	}
	return service, nil
}

//	{
//	  責務: [writeAnalyzeHelp: analyzeの入力・図の複数指定・保存先を日本語で案内する,],
//	}
func writeAnalyzeHelp(w io.Writer) {
	fmt.Fprintln(w, "使用方法: tomiya-code-atlas-cli.exe analyze <ソースファイルまたはプロジェクトフォルダー> --operation <図> [--operation <図> ...] --output-dir <保存先フォルダー>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "--operationは作成する図ごとに繰り返し指定できます。")
	fmt.Fprintln(w, "入力にフォルダーを指定すると、対応するソースファイルを配下から再帰的に探します。")
	fmt.Fprintln(w, "各ファイルを拡張子に対応する解析補助プログラムで調べ、結果をまとめて図を作成します。")
	fmt.Fprintln(w, "  call-graph    呼び出し図を生成")
	fmt.Fprintln(w, "  class-diagram クラス図を生成")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "例: tomiya-code-atlas-cli.exe analyze src --operation call-graph --operation class-diagram --output-dir atlas-output")
}
