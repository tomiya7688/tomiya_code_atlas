package process

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
	"github.com/tomiya7688/tomiya_code_atlas/internal/renderers"
)

const (
	OperationCallGraph    = "call-graph"
	OperationClassDiagram = "class-diagram"
)

var diagramOperationIDs = []string{OperationCallGraph, OperationClassDiagram}

//	{
//	  責務: [SupportedSourceFileExtensions: Dataの探索器が対応する拡張子をファイル選択画面へ公開する,],
//	  戻り値: [extensions: 選択画面に渡す小文字の拡張子一覧,],
//	}
func SupportedSourceFileExtensions() []string {
	return data.NewProjectFileScanner().SupportedFileExtensions()
}

//	{
//	  責務: [SupportedDiagramOperations: Go版Processが実行できる図の処理IDを定めた順序で返す,],
//	  戻り値: [operations: 利用可能な処理IDの複製,],
//	}
func SupportedDiagramOperations() []string {
	return append([]string(nil), diagramOperationIDs...)
}

//	{
//	  責務: [IsDiagramOperationSupported: 指定された処理IDがGo版Processに実装されているか判定する,],
//	  引数: [operation: 実装の有無を確認する処理ID,],
//	  戻り値: [bool: 対応済みならtrue,],
//	}
func IsDiagramOperationSupported(operation string) bool {
	for _, supportedOperation := range diagramOperationIDs {
		if operation == supportedOperation {
			return true
		}
	}
	return false
}

//	{
//	  責務: [DiagramRequest: 一回の解析で作成する図と入出力パスを保持する,],
//	  状態: [SourcePath: 解析するファイルまたはプロジェクトフォルダー, Operations: 生成する図の処理ID, OutputDirectory: 生成ファイルの保存先,],
//	}
type DiagramRequest struct {
	SourcePath      string
	Operations      []string
	OutputDirectory string
}

//	{
//	  責務: [DiagramService: 言語別の解析補助プログラムと共通の生成・描画処理を調整して図を保存する,],
//	  状態: [parserHost: manifestを使って解析補助プログラムを選ぶProcessサービス,],
//	}
type DiagramService struct {
	parserHost ParserHost
}

//	{
//	  責務: [NewDiagramService: 配布manifestと解析補助プログラムを検証して図生成サービスを作る,],
//	  処理: [1: ParserHostを初期化する, 2: 図生成サービスを返す,],
//	  引数: [applicationRoot: parser manifestと補助プログラムの起点, timeout: 補助プログラム一回の実行時間上限,],
//	  戻り値: [service: ソース解析と図の出力を行うProcessサービス, error: manifestまたは契約の検証に失敗した理由,],
//	}
func NewDiagramService(applicationRoot string, timeout time.Duration) (DiagramService, error) {
	parserHost, err := NewParserHost(applicationRoot, timeout)
	if err != nil {
		return DiagramService{}, err
	}
	return DiagramService{parserHost: parserHost}, nil
}

//	{
//	  責務: [Generate: 指定されたファイルまたはプロジェクトフォルダーを解析し、選んだ図をMarkdownへ保存する,],
//	  処理: [1: 入力・処理ID・保存先を検証する, 2: Dataから対応ソースを探す, 3: 言語別の解析補助プログラムで共通IRを作り統合する, 4: 選んだ図を生成して保存する,],
//	  引数: [ctx: 呼出元が指定する取消・期限, request: ソース・図の種類・保存先,],
//	  戻り値: [outputPaths: 保存したファイルのパス, error: 入力・解析・図の生成・保存に失敗した理由,],
//	}
func (service DiagramService) Generate(ctx context.Context, request DiagramRequest) ([]string, error) {
	if request.SourcePath == "" || request.OutputDirectory == "" || len(request.Operations) == 0 {
		return nil, fmt.Errorf("解析するソースファイルまたはプロジェクトフォルダー、図の種類、保存先フォルダーを指定してください")
	}
	seenOperations := make(map[string]struct{}, len(request.Operations))
	for _, operation := range request.Operations {
		if !IsDiagramOperationSupported(operation) {
			return nil, fmt.Errorf("未対応の図です: %s", operation)
		}
		if _, duplicate := seenOperations[operation]; duplicate {
			return nil, fmt.Errorf("同じ図を重複して指定しています: %s", operation)
		}
		seenOperations[operation] = struct{}{}
	}
	scanner := data.NewProjectFileScanner()
	sourceFiles, err := scanner.DiscoverSupportedFiles(request.SourcePath)
	if err != nil {
		return nil, err
	}
	if len(sourceFiles) == 0 {
		return nil, fmt.Errorf("対応するソースファイルが見つかりません: %s", request.SourcePath)
	}
	module, err := service.parseProjectModules(ctx, scanner, sourceFiles)
	if err != nil {
		return nil, err
	}
	isDirectory, err := data.IsDirectory(request.SourcePath)
	if err != nil {
		return nil, err
	}
	baseName := filepath.Base(request.SourcePath)
	if !isDirectory {
		baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	}
	outputPaths := make([]string, 0, len(request.Operations))
	for _, operation := range request.Operations {
		contents, err := renderDiagram(module, operation)
		if err != nil {
			return nil, err
		}
		outputPath, err := data.WriteTextFile(request.OutputDirectory, baseName+"_"+operation+".md", contents)
		if err != nil {
			return nil, err
		}
		outputPaths = append(outputPaths, outputPath)
	}
	return outputPaths, nil
}

//	{
//	  責務: [parseProjectModules: 見つかった各ソースを対応する解析補助プログラムで解析し、共通IRへまとめる,],
//	  処理: [1: ソースをパス順に読み込む, 2: 言語別の補助プログラムで解析する, 3: 各結果の情報を一つのモジュールへ加える,],
//	  引数: [ctx: 解析処理に指定する取消・期限, scanner: 拡張子から言語を判定する探索器, sourceFiles: パス順に並んだ解析対象,],
//	  戻り値: [commonir.Module: 全ソースの宣言と関係をまとめたIR, error: 読込・パス解決・構文解析に失敗した理由,],
//	}
func (service DiagramService) parseProjectModules(ctx context.Context, scanner data.ProjectFileScanner, sourceFiles []data.ProjectSourceFile) (commonir.Module, error) {
	combined := commonir.Module{}
	languages := map[string]struct{}{}
	for _, sourceFile := range sourceFiles {
		source, err := data.ReadSourceFile(sourceFile.Path)
		if err != nil {
			return commonir.Module{}, err
		}
		if sourceFile.Language == "unknown" {
			return commonir.Module{}, fmt.Errorf("拡張子から解析言語を判定できません: %s", filepath.Ext(sourceFile.Path))
		}
		sourcePath, err := filepath.Abs(sourceFile.Path)
		if err != nil {
			return commonir.Module{}, fmt.Errorf("ソースファイルのパスを解決できません (%s): %w", sourceFile.Path, err)
		}
		payload, err := service.parserHost.ParseLanguage(ctx, sourceFile.Language, source, &sourcePath)
		if err != nil {
			return commonir.Module{}, fmt.Errorf("ソースファイルを解析できません (%s): %w", sourceFile.Path, err)
		}
		appendModule(&combined, payload.Module)
		languages[sourceFile.Language] = struct{}{}
	}
	if len(languages) > 1 {
		combined.Language = "mixed"
	}
	return combined, nil
}

//	{
//	  責務: [appendModule: 一つのソースから得た共通IRを集約先のモジュールへ加える,],
//	  処理: [1: 集約先に言語を記録する, 2: 言語に依存しない情報をすべて追加する,],
//	  引数: [target: 解析結果を集約するモジュール, source: 一つのファイルから得たモジュール,],
//	}
func appendModule(target *commonir.Module, source commonir.Module) {
	if target.Language == "" {
		target.Language = source.Language
	} else if target.Language != source.Language {
		target.Language = "mixed"
	}
	target.Entities = append(target.Entities, source.Entities...)
	target.Objects = append(target.Objects, source.Objects...)
	target.StateMachines = append(target.StateMachines, source.StateMachines...)
	target.TimingFlows = append(target.TimingFlows, source.TimingFlows...)
	target.InputEvents = append(target.InputEvents, source.InputEvents...)
	target.Signals = append(target.Signals, source.Signals...)
	target.Diagnostics = append(target.Diagnostics, source.Diagnostics...)
	target.Imports = append(target.Imports, source.Imports...)
}

//	{
//	  責務: [renderDiagram: 共通IRから選択された図をMermaid形式のMarkdownへ変換する,],
//	  処理: [1: 図に対応する論理モデルを作る, 2: RendererでMarkdown形式に整える,],
//	  引数: [module: 解析補助プログラムから得た言語非依存のIR, operation: 生成する図の処理ID,],
//	  戻り値: [string: 保存可能なMermaid形式のMarkdown, error: 図の生成に失敗した理由,],
//	}
func renderDiagram(module commonir.Module, operation string) (string, error) {
	switch operation {
	case "call-graph":
		graph, err := generators.BuildCallGraph(module, nil, nil)
		if err != nil {
			return "", fmt.Errorf("呼び出し図を生成できません: %w", err)
		}
		return "# 呼び出し図\n\n```mermaid\n" + renderers.RenderCallGraphMermaid(graph, "LR") + "```\n", nil
	case "class-diagram":
		bundle, err := generators.BuildClassDiagramBundle(module, 3, nil)
		if err != nil {
			return "", fmt.Errorf("クラス図を生成できません: %w", err)
		}
		var output strings.Builder
		output.WriteString("# クラス図\n")
		for _, diagram := range bundle.Diagrams {
			fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", diagram.Name)
			output.WriteString(renderers.RenderClassDiagramMermaid(diagram))
			output.WriteString("```\n")
		}
		return output.String(), nil
	default:
		return "", fmt.Errorf("未対応の図です: %s", operation)
	}
}
