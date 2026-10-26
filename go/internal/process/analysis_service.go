package process

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
	"github.com/tomiya7688/tomiya_code_atlas/internal/evaluators"
	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
	"github.com/tomiya7688/tomiya_code_atlas/internal/renderers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/staticdoc"
)

//	{
//	  責務: [AnalysisRequest: 一回のsource解析で生成するoperationと入出力pathを保持する]
//	  状態: [SourcePath: 解析するsource fileまたはproject directory, Operations: 実行するanalysis operation, OutputDirectory: 生成fileの保存先]
//	}
type AnalysisRequest struct {
	SourcePath      string
	Operations      []string
	OutputDirectory string
}

//	{
//	  責務: [AnalysisService: parser helperと共通Generator/Evaluator/Rendererを調整して選択結果を保存する]
//	  状態: [parserHost: manifestからparser helperを選択するProcess service]
//	}
type AnalysisService struct {
	parserHost ParserHost
}

//	{
//	  責務: [NewAnalysisService: 配布manifestからparser helper hostを検証してserviceを作る]
//	  処理: [1: ParserHostを初期化する, 2: analysis serviceを返す]
//	  引数: [applicationRoot: backends manifestとhelper assetのroot, timeout: helper一回の実行時間上限]
//	  戻り値: [service: source解析と製品出力のProcess service, error: manifest・契約違反の説明]
//	}
func NewAnalysisService(applicationRoot string, timeout time.Duration) (AnalysisService, error) {
	parserHost, err := NewParserHost(applicationRoot, timeout)
	if err != nil {
		return AnalysisService{}, err
	}
	return AnalysisService{parserHost: parserHost}, nil
}

//	{
//	  責務: [Generate: fileまたはproject directory内のsourceを解析し、選択結果をoutput folderへ保存する]
//	  処理: [1: requestの必須pathとoperationを検証する, 2: Dataで対応sourceを探索する, 3: 各fileをmanifest helperでCommon IRへparseして統合する, 4: diagram/spec/reportまたはcommented source copyを保存する]
//	  引数: [ctx: 呼出元のcancel・deadline, request: source・operation・outputの指定]
//	  戻り値: [outputPaths: 保存したfile path, error: 入力・parser・Generator・保存に失敗した場合の説明]
//	}
func (service AnalysisService) Generate(ctx context.Context, request AnalysisRequest) ([]string, error) {
	if request.SourcePath == "" || request.OutputDirectory == "" || len(request.Operations) == 0 {
		return nil, fmt.Errorf("source fileまたはproject folder、operation、出力folderを指定してください")
	}
	seenOperations := make(map[string]struct{}, len(request.Operations))
	for _, operation := range request.Operations {
		if !IsSupportedOperation(operation) {
			return nil, fmt.Errorf("未対応のoperationです: %s", operation)
		}
		if _, duplicate := seenOperations[operation]; duplicate {
			return nil, fmt.Errorf("operationを重複して指定しています: %s", operation)
		}
		seenOperations[operation] = struct{}{}
	}
	scanner := data.NewProjectFileScanner()
	sourceFiles, err := scanner.DiscoverSupportedFiles(request.SourcePath)
	if err != nil {
		return nil, err
	}
	if len(sourceFiles) == 0 {
		return nil, fmt.Errorf("対応するsource fileが見つかりません: %s", request.SourcePath)
	}
	isDirectory, err := data.IsDirectory(request.SourcePath)
	if err != nil {
		return nil, err
	}
	commentOutputPaths := make([]string, 0, len(sourceFiles))
	var sourceFileIdentities *data.SourceFileIdentitySet
	var afterParse func(data.ProjectSourceFile, string, commonir.Module) error
	if _, generateComments := seenOperations["comments"]; generateComments {
		//	{
		//	  処理: [全source pathのfile identityをparser実行前に捕捉する,]
		//	}
		sourcePaths := make([]string, len(sourceFiles))
		for index, sourceFile := range sourceFiles {
			sourcePaths[index] = sourceFile.Path
		}
		identities, err := data.NewSourceFileIdentitySet(sourcePaths)
		if err != nil {
			return nil, err
		}
		sourceFileIdentities = &identities
		commentWriter := CommentedSourceWriter{
			sourceRoot:           request.SourcePath,
			outputDirectory:      request.OutputDirectory,
			sourceIsDirectory:    isDirectory,
			sourceFileIdentities: identities,
		}
		afterParse = func(sourceFile data.ProjectSourceFile, source string, module commonir.Module) error {
			outputPath, err := commentWriter.WriteCommentedCopy(sourceFile, source, module)
			if err != nil {
				return err
			}
			commentOutputPaths = append(commentOutputPaths, outputPath)
			return nil
		}
	}
	module, err := service.parseProjectModules(ctx, sourceFiles, sourceFileIdentities, afterParse)
	if err != nil {
		return nil, err
	}
	baseName := filepath.Base(request.SourcePath)
	if !isDirectory {
		baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	}
	outputPaths := make([]string, 0, len(request.Operations))
	for _, operation := range request.Operations {
		if operation == "comments" {
			outputPaths = append(outputPaths, commentOutputPaths...)
			continue
		}
		contents, err := renderAnalysisOutput(module, operation, baseName)
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
//	  責務: [parseProjectModules: 発見した各sourceを適切なparser helperで解析してCommon IRを統合する]
//	  処理: [1: source内容を取得する, 2: language別helperで解析する, 3: 各file callbackを実行してからIRを統合する]
//	  引数: [ctx: parser実行のcancel・deadline, sourceFiles: path順の解析対象, sourceFileIdentities: コメント出力時に捕捉したfile identity。未指定ならnil, afterParse: fileごとの追加処理。未指定ならnil]
//	  戻り値: [commonir.Module: 全sourceの宣言・関係をまとめたIR, error: 読込・path解決・parserに失敗した場合の説明]
//	}
func (service AnalysisService) parseProjectModules(ctx context.Context, sourceFiles []data.ProjectSourceFile, sourceFileIdentities *data.SourceFileIdentitySet, afterParse func(data.ProjectSourceFile, string, commonir.Module) error) (commonir.Module, error) {
	combined := commonir.Module{}
	languages := map[string]struct{}{}
	for _, sourceFile := range sourceFiles {
		var source string
		if sourceFileIdentities != nil {
			var err error
			source, err = sourceFileIdentities.ReadSourceFileMatchingIdentity(sourceFile.Path)
			if err != nil {
				return commonir.Module{}, err
			}
		} else {
			var err error
			source, err = data.ReadSourceFile(sourceFile.Path)
			if err != nil {
				return commonir.Module{}, err
			}
		}
		if sourceFile.Language == "unknown" {
			return commonir.Module{}, fmt.Errorf("拡張子から解析言語を判定できません: %s", filepath.Ext(sourceFile.Path))
		}
		sourcePath, err := filepath.Abs(sourceFile.Path)
		if err != nil {
			return commonir.Module{}, fmt.Errorf("source pathを解決できません (%s): %w", sourceFile.Path, err)
		}
		payload, err := service.parserHost.ParseLanguage(ctx, sourceFile.Language, source, &sourcePath)
		if err != nil {
			return commonir.Module{}, fmt.Errorf("sourceを解析できません (%s): %w", sourceFile.Path, err)
		}
		if afterParse != nil {
			if err := afterParse(sourceFile, source, payload.Module); err != nil {
				return commonir.Module{}, err
			}
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
//	  責務: [appendModule: parser helperが返したCommon IRの配列要素を集約先へ追加する]
//	  処理: [1: 初回moduleのlanguageを記録する, 2: 各言語非依存fact collectionを追加する]
//	  引数: [target: 解析結果を集約するmodule, source: 一つのfileから得たmodule]
//	  戻り値: []
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
//	  責務: [renderAnalysisOutput: Common IRから選択operationの日本語Markdownを生成する]
//	  処理: [1: operationに対応するGenerator/Evaluatorを呼び出す, 2: logical resultをRendererへ渡す]
//	  引数: [module: helperが返したlanguage-neutral Common IR, operation: 配布CLI operation ID, sourceName: 入力fileまたはfolder名]
//	  戻り値: [1: string: 保存可能なMarkdown, 2: error: GeneratorまたはEvaluatorが失敗した場合の説明]
//	}
func renderAnalysisOutput(module commonir.Module, operation, sourceName string) (string, error) {
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
	case "static-spec":
		return staticdoc.Render(module, sourceName), nil
	case "design-quality":
		report, err := evaluators.EvaluateDesignQuality(module, nil)
		if err != nil {
			return "", fmt.Errorf("設計品質を評価できません: %w", err)
		}
		return renderers.RenderDesignQualityMarkdown(report), nil
	default:
		return "", fmt.Errorf("未対応のoperationです: %s", operation)
	}
}
