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
//	  責務: [run: top-level commandを検証しhelp・version・analyzeの処理先へ振り分ける]
//	  処理: [1: command名を判定する, 2: 各commandを呼び出す, 3: versioned CLI exit codeを返す]
//	  引数: [args: command line引数, stdout: 利用者向け正常出力, stderr: 利用者向けerror出力]
//	  戻り値: [int: 成功0、解析失敗1、引数誤り2のprocess exit status]
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
//	  責務: [writeHelp: 利用可能commandと詳細helpの開き方を日本語で表示する]
//	  引数: [w: help本文を書き出すstream]
//	}
func writeHelp(w io.Writer) {
	fmt.Fprintln(w, "Tomiya Code Atlas (Go)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "使用方法: tomiya-code-atlas [コマンド]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "コマンド:")
	fmt.Fprintln(w, "  --help, -h     この案内を表示")
	fmt.Fprintln(w, "  --version, -V  バージョンを表示")
	fmt.Fprintln(w, "  analyze        ソースを解析して図を保存")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "詳しい使い方: tomiya-code-atlas analyze --help")
}

//	{
//	  責務: [analyzeOptions: analyze commandの入力source、operation、出力先を保持する]
//	  状態: [sourcePath: 解析するsource file, operations: 実行する図生成operation, outputDirectory: 生成fileの保存先]
//	}
type analyzeOptions struct {
	sourcePath      string
	operations      []string
	outputDirectory string
}

//	{
//	  責務: [supportedOperations: Go CLIで生成できるdiagram operationを列挙する]
//	}
var supportedOperations = map[string]struct{}{"call-graph": {}, "class-diagram": {}}

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
			if _, supported := supportedOperations[operation]; !supported {
				return analyzeOptions{}, fmt.Errorf("未対応のoperationです: %s（選択可能: call-graph, class-diagram）", operation)
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
//	  責務: [runAnalyze: 指定source fileまたはproject folderをparserで解析し、複数の図を出力先へ保存する]
//	  処理: [1: parser hostを解決する, 2: Process serviceへ解析とCommon IR統合を依頼する, 3: operationごとにMermaidを生成しMarkdown outputを保存する]
//	  引数: [options: 検証済みsource・operation・output directory]
//	  戻り値: [error: source読込・parser・generator・renderer・file保存に失敗した場合の内容]
//	}
func runAnalyze(options analyzeOptions) error {
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("実行fileのpathを確認できません: %w", err)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("実行directoryを確認できません: %w", err)
	}
	applicationRoot, err := data.FindApplicationRoot(executablePath, workingDirectory)
	if err != nil {
		return err
	}
	service, err := process.NewDiagramService(applicationRoot, 30*time.Second)
	if err != nil {
		return fmt.Errorf("parser helperを準備できません: %w", err)
	}
	_, err = service.Generate(context.Background(), process.DiagramRequest{
		SourcePath: options.sourcePath, Operations: options.operations, OutputDirectory: options.outputDirectory,
	})
	return err
}

//	{
//	  責務: [writeAnalyzeHelp: analyzeの入力・operation複数指定・出力先を日本語で案内する]
//	}
func writeAnalyzeHelp(w io.Writer) {
	fmt.Fprintln(w, "使用方法: tomiya-code-atlas analyze <source file or project folder> --operation <operation> [--operation <operation> ...] --output-dir <folder>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "operationは複数回指定できます。")
	fmt.Fprintln(w, "入力がfolderの場合、対応するsource fileを下位folderも含めて解析します。")
	fmt.Fprintln(w, "各source fileは拡張子に対応するparserで個別に解析し、結果をまとめて図を生成します。")
	fmt.Fprintln(w, "  call-graph    呼び出し図を生成")
	fmt.Fprintln(w, "  class-diagram クラス図を生成")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "例: tomiya-code-atlas analyze src --operation call-graph --operation class-diagram --output-dir atlas-output")
}
