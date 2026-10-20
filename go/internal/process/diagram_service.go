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

//	{
//	  責務: [DiagramRequest: 一回のsource解析で生成するoperationと入出力pathを保持する]
//	  状態: [SourcePath: 解析する一つのsource file, Operations: 実行するdiagram operation, OutputDirectory: 生成fileの保存先]
//	}
type DiagramRequest struct {
	SourcePath      string
	Operations      []string
	OutputDirectory string
}

//	{
//	  責務: [DiagramService: parser helperと共通Generator/Rendererを調整して図を保存する]
//	  状態: [parserHost: manifestからparser helperを選択するProcess service]
//	}
type DiagramService struct {
	parserHost ParserHost
}

//	{
//	  責務: [NewDiagramService: 配布manifestからparser helper hostを検証してserviceを作る]
//	  処理: [1: ParserHostを初期化する, 2: diagram serviceを返す]
//	  引数: [applicationRoot: backends manifestとhelper assetのroot, timeout: helper一回の実行時間上限]
//	  戻り値: [service: source解析と図出力のProcess service, error: manifest・契約違反の説明]
//	}
func NewDiagramService(applicationRoot string, timeout time.Duration) (DiagramService, error) {
	parserHost, err := NewParserHost(applicationRoot, timeout)
	if err != nil {
		return DiagramService{}, err
	}
	return DiagramService{parserHost: parserHost}, nil
}

//	{
//	  責務: [Generate: sourceを一度parseし、選択した複数diagramをMarkdown fileへ保存する]
//	  処理: [1: requestの必須pathとoperationを検証する, 2: Dataからsourceを読みlanguageを判定する, 3: manifest helperでCommon IRへparseする, 4: 各diagramを生成してDataへ保存する]
//	  引数: [ctx: 呼出元のcancel・deadline, request: source・operation・outputの指定]
//	  戻り値: [outputPaths: 保存したfile path, error: 入力・parser・Generator・保存に失敗した場合の説明]
//	}
func (service DiagramService) Generate(ctx context.Context, request DiagramRequest) ([]string, error) {
	if request.SourcePath == "" || request.OutputDirectory == "" || len(request.Operations) == 0 {
		return nil, fmt.Errorf("source file、operation、出力folderを指定してください")
	}
	seenOperations := make(map[string]struct{}, len(request.Operations))
	for _, operation := range request.Operations {
		if operation != "call-graph" && operation != "class-diagram" {
			return nil, fmt.Errorf("未対応のoperationです: %s", operation)
		}
		if _, duplicate := seenOperations[operation]; duplicate {
			return nil, fmt.Errorf("operationを重複して指定しています: %s", operation)
		}
		seenOperations[operation] = struct{}{}
	}
	source, err := data.ReadSourceFile(request.SourcePath)
	if err != nil {
		return nil, err
	}
	scanner := data.NewProjectFileScanner()
	language := scanner.DetectLanguage(request.SourcePath)
	if language == "unknown" {
		return nil, fmt.Errorf("拡張子から解析言語を判定できません: %s", filepath.Ext(request.SourcePath))
	}
	sourcePath, err := filepath.Abs(request.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("source pathを解決できません: %w", err)
	}
	payload, err := service.parserHost.ParseLanguage(ctx, language, source, &sourcePath)
	if err != nil {
		return nil, err
	}
	baseName := strings.TrimSuffix(filepath.Base(request.SourcePath), filepath.Ext(request.SourcePath))
	outputPaths := make([]string, 0, len(request.Operations))
	for _, operation := range request.Operations {
		contents, err := renderDiagram(payload.Module, operation)
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
//	  責務: [renderDiagram: Common IRから選択diagramをMermaid Markdownへ変換する]
//	  処理: [1: operationに対応するlogical modelを生成する, 2: Rendererを呼びMarkdown fenceへ包む]
//	  引数: [module: helperが返したlanguage-neutral Common IR, operation: diagram ID]
//	  戻り値: [string: 保存可能なMermaid Markdown, error: Generatorが失敗した場合の説明]
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
		return "", fmt.Errorf("未対応のoperationです: %s", operation)
	}
}
