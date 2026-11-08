package process

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/tomiya7688/tomiya_code_atlas/internal/analyzers"
	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
	"github.com/tomiya7688/tomiya_code_atlas/internal/evaluators"
	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
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
//	  責務: [Generate: fileまたはproject directory内のsourceを解析し、選択結果をoutput folderへ保存する,]
//	  処理: [1: requestを検証する, 2: Dataで対応sourceを探索する, 3: parser helperでCommon IRへ解析する, 4: 選択されたoutputを保存する,]
//	  引数: [ctx: 呼出元のcancel・deadline, request: source・operation・outputの指定,]
//	  戻り値: [outputPaths: 保存したfile path, error: 入力・parser・Generator・保存に失敗した場合の説明,]
//	}
func (service AnalysisService) Generate(ctx context.Context, request AnalysisRequest) ([]string, error) {
	seenOperations, isDirectory, err := validateAnalysisRequest(request)
	if err != nil {
		return nil, err
	}
	compatibleOperations, err := service.CompatibleOperations(request.SourcePath)
	if err != nil {
		return nil, err
	}
	compatibleSet := make(map[string]struct{}, len(compatibleOperations))
	for _, operation := range compatibleOperations {
		compatibleSet[operation] = struct{}{}
	}
	for _, operation := range request.Operations {
		if _, compatible := compatibleSet[operation]; !compatible {
			return nil, fmt.Errorf(
				"選択した入力では実行できない操作です: %s（この入力で利用できる操作: %s）",
				operation,
				strings.Join(compatibleOperations, ", "),
			)
		}
	}
	scanner := data.NewProjectFileScanner()
	var sourceFiles []data.ProjectSourceFile
	if requiresSourceAnalysis(seenOperations) {
		sourceFiles, err = scanner.DiscoverSupportedFiles(request.SourcePath)
		if err != nil {
			return nil, err
		}
		if len(sourceFiles) == 0 {
			return nil, fmt.Errorf("対応するsource fileが見つかりません: %s", request.SourcePath)
		}
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
	module := commonir.Module{}
	moduleUnits := []analyzers.ModuleDependencyUnit{}
	componentUnits := []analyzers.ComponentDependencyUnit{}
	if len(sourceFiles) > 0 {
		dependencyRoot := request.SourcePath
		if !isDirectory {
			dependencyRoot = filepath.Dir(request.SourcePath)
		}
		module, moduleUnits, componentUnits, err = service.parseProjectModules(ctx, dependencyRoot, sourceFiles, sourceFileIdentities, afterParse)
		if err != nil {
			return nil, err
		}
	}
	deploymentTopology := analyzers.DeploymentTopology{}
	if _, generateDeploymentDiagram := seenOperations["deployment-diagram"]; generateDeploymentDiagram {
		deploymentTopology, err = readDeploymentTopology(request.SourcePath, isDirectory, scanner)
		if err != nil {
			return nil, err
		}
	}
	ciWorkflowFiles := []ciWorkflowFile{}
	if _, generateCIGraph := seenOperations["ci-graph"]; generateCIGraph {
		ciWorkflowFiles, err = readCIWorkflowFiles(request.SourcePath, isDirectory, scanner)
		if err != nil {
			return nil, err
		}
	}
	return writeAnalysisOutputs(request, isDirectory, commentOutputPaths, module, moduleUnits, componentUnits, deploymentTopology, ciWorkflowFiles)
}

//	{
//	  責務: [CompatibleOperations: 入力pathの形式と言語に応じて実行可能な操作IDを返す]
//	  処理: [1: 入力種別と対応sourceを調べる, 2: source言語・workflow・設定fileに応じて操作を選ぶ, 3: registry順で返す]
//	  引数: [sourcePath: source fileまたはproject folder]
//	  戻り値: [operations: 入力に対応する操作ID, error: 入力を調査できない場合の説明]
//	}
func (service AnalysisService) CompatibleOperations(sourcePath string) ([]string, error) {
	scanner := data.NewProjectFileScanner()
	isDirectory, err := data.IsDirectory(sourcePath)
	if err != nil {
		return nil, err
	}
	sourceFiles, err := scanner.DiscoverSupportedFiles(sourcePath)
	if err != nil {
		return nil, err
	}
	ciWorkflowFiles, err := scanner.DiscoverCIWorkflowFiles(sourcePath)
	if err != nil {
		return nil, err
	}
	validCIWorkflows := false
	if len(ciWorkflowFiles) > 0 {
		_, parseErr := readCIWorkflowFiles(sourcePath, isDirectory, scanner)
		validCIWorkflows = parseErr == nil
	}
	if !isDirectory && (strings.EqualFold(filepath.Ext(sourcePath), ".yaml") ||
		strings.EqualFold(filepath.Ext(sourcePath), ".yml")) {
		if !validCIWorkflows {
			return []string{}, nil
		}
		return []string{"ci-graph"}, nil
	}
	if len(sourceFiles) == 0 && !isDirectory {
		return nil, fmt.Errorf("対応するsource fileではありません: %s", sourcePath)
	}

	availableLanguages := make(map[string]struct{}, len(sourceFiles))
	for _, sourceFile := range sourceFiles {
		availableLanguages[sourceFile.Language] = struct{}{}
	}
	available := make(map[string]struct{})
	if len(sourceFiles) > 0 {
		for _, operation := range []string{
			"call-graph", "class-diagram", "sequence-diagram", "communication-diagram",
			"responsibility-table", "comments", "static-spec", "design-quality",
		} {
			available[operation] = struct{}{}
		}
		if _, parsesPython := availableLanguages["python"]; parsesPython {
			for _, operation := range []string{"object-diagram", "state-diagram", "timing-chart", "use-case-diagram"} {
				available[operation] = struct{}{}
			}
		}
		if isDirectory {
			available["package-diagram"] = struct{}{}
			available["component-diagram"] = struct{}{}
		}
	}
	if isDirectory {
		deploymentFiles, err := scanner.DiscoverDeploymentFiles(sourcePath)
		if err != nil {
			return nil, err
		}
		if len(deploymentFiles) > 0 {
			available["deployment-diagram"] = struct{}{}
		}
	}
	if validCIWorkflows {
		available["ci-graph"] = struct{}{}
	}
	operations := make([]string, 0, len(available))
	for _, operation := range SupportedOperationIDs() {
		if _, found := available[operation]; found {
			operations = append(operations, operation)
		}
	}
	return operations, nil
}

//	{
//	  責務: [requiresSourceAnalysis: 選択結果のうち言語sourceのCommon IRを必要とするoperationがあるか調べる]
//	  引数: [operations: 重複を除いたoperation ID集合]
//	  戻り値: [bool: source解析を必要とするoperationがあればtrue]
//	}
func requiresSourceAnalysis(operations map[string]struct{}) bool {
	for operation := range operations {
		if operation != "deployment-diagram" && operation != "ci-graph" {
			return true
		}
	}
	return false
}

//	{
//	  責務: [validateAnalysisRequest: 入力pathとoperationを検証して解析に必要な状態を返す,]
//	  処理: [1: 必須入力とoperationを確認する, 2: source folderを判定する, 3: comments出力先がsource folderと重ならないか確認する,]
//	  引数: [request: source・operation・outputの指定,]
//	  戻り値: [seenOperations: 重複を除いたoperation一覧, isDirectory: sourceがfolderならtrue, error: requestが不正またはコメント出力先がproject内にある場合の説明,]
//	}
func validateAnalysisRequest(request AnalysisRequest) (map[string]struct{}, bool, error) {
	if request.SourcePath == "" || request.OutputDirectory == "" || len(request.Operations) == 0 {
		return nil, false, fmt.Errorf("source fileまたはproject folder、operation、出力folderを指定してください")
	}
	seenOperations := make(map[string]struct{}, len(request.Operations))
	for _, operation := range request.Operations {
		if !IsSupportedOperation(operation) {
			return nil, false, fmt.Errorf("未対応のoperationです: %s", operation)
		}
		if _, duplicate := seenOperations[operation]; duplicate {
			return nil, false, fmt.Errorf("operationを重複して指定しています: %s", operation)
		}
		seenOperations[operation] = struct{}{}
	}
	isDirectory, err := data.IsDirectory(request.SourcePath)
	if err != nil {
		return nil, false, err
	}
	if _, generateComments := seenOperations["comments"]; generateComments && isDirectory {
		commentsDirectory := filepath.Join(request.OutputDirectory, "comments")
		commentsInsideSource, err := data.IsPathWithinOrEqual(request.SourcePath, commentsDirectory)
		if err != nil {
			return nil, false, err
		}
		sourceInsideComments, err := data.IsPathWithinOrEqual(commentsDirectory, request.SourcePath)
		if err != nil {
			return nil, false, err
		}
		if commentsInsideSource || sourceInsideComments {
			return nil, false, fmt.Errorf("コメント付きsourceの保存先は入力project folderと重ならない場所を指定してください")
		}
	}
	return seenOperations, isDirectory, nil
}

//	{
//	  責務: [writeAnalysisOutputs: 選択された解析結果をfolderへ保存する,]
//	  処理: [1: outputの基準名を決める, 2: comments pathを加える, 3: 他operationをMarkdownへrenderして保存する,]
//	  引数: [request: output folderとoperationの指定, isDirectory: sourceがfolderならtrue, commentOutputPaths: 保存済みcomment copy path, module: 解析結果のCommon IR,]
//	  戻り値: [outputPaths: 保存したfile path, error: renderまたは保存に失敗した場合の説明,]
//	}
func writeAnalysisOutputs(request AnalysisRequest, isDirectory bool, commentOutputPaths []string, module commonir.Module, moduleUnits []analyzers.ModuleDependencyUnit, componentUnits []analyzers.ComponentDependencyUnit, deploymentTopology analyzers.DeploymentTopology, ciWorkflowFiles []ciWorkflowFile) ([]string, error) {
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
		if operation == "ci-graph" {
			outputPath, err := data.WriteTextFile(request.OutputDirectory, baseName+"_ci-graph.md", renderCIWorkflowFiles(ciWorkflowFiles))
			if err != nil {
				return nil, err
			}
			outputPaths = append(outputPaths, outputPath)
			continue
		}
		contents, err := renderAnalysisOutput(module, operation, baseName, moduleUnits, componentUnits, deploymentTopology)
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

type ciWorkflowFile struct {
	RelativePath string
	Workflow     models.CIWorkflow
}

//	{
//	  責務: [readCIWorkflowFiles: 選択YAMLまたはproject内GitHub workflow fileを読み取り、CI diagram modelへ変換する]
//	  処理: [1: CI候補を列挙する, 2: YAML本文を読み込む, 3: 既存のCI analyzerで図用modelへ変換する]
//	  引数: [sourceRoot: YAML fileまたはproject folder, sourceIsDirectory: folder入力ならtrue, scanner: 共通path除外規則を持つscanner]
//	  戻り値: [files: path順のworkflow model, error: 読込・YAML構文・workflow契約に失敗した場合の説明]
//	}
func readCIWorkflowFiles(sourceRoot string, sourceIsDirectory bool, scanner data.ProjectFileScanner) ([]ciWorkflowFile, error) {
	paths, err := scanner.DiscoverCIWorkflowFiles(sourceRoot)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("GitHub Actions workflowが見つかりません: %s", sourceRoot)
	}
	files := make([]ciWorkflowFile, 0, len(paths))
	for _, path := range paths {
		source, err := data.ReadSourceFile(path)
		if err != nil {
			return nil, err
		}
		relativePath := filepath.Base(path)
		if sourceIsDirectory {
			relativePath, err = filepath.Rel(sourceRoot, path)
			if err != nil {
				return nil, fmt.Errorf("workflow pathをproject相対に変換できません (%s): %w", path, err)
			}
		}
		workflow, err := analyzers.ParseGitHubActionsWorkflow(source, filepath.ToSlash(relativePath))
		if err != nil {
			return nil, err
		}
		files = append(files, ciWorkflowFile{RelativePath: filepath.ToSlash(relativePath), Workflow: workflow})
	}
	return files, nil
}

//	{
//	  責務: [renderCIWorkflowFiles: 複数workflowをpath見出し付きの一つのMarkdown CI図へまとめる]
//	  処理: [1: 各pathを見出しにする, 2: rendererへlogical diagramを渡す, 3: Mermaid blockへ格納する]
//	  引数: [files: project内のpath順workflow model]
//	  戻り値: [string: 保存可能なGitHub Actions/CI diagram Markdown]
//	}
func renderCIWorkflowFiles(files []ciWorkflowFile) string {
	var output strings.Builder
	output.WriteString("# GitHub Actions / CI図\n")
	for _, file := range files {
		fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", file.RelativePath)
		diagram := generators.BuildCIWorkflowDiagram(file.Workflow)
		output.WriteString(renderers.RenderCIWorkflowMermaid(diagram))
		output.WriteString("```\n")
	}
	return output.String()
}

//	{
//	  責務: [parseProjectModules: 発見した各sourceを適切なparser helperで解析してCommon IRを統合する]
//	  処理: [1: source内容を取得する, 2: language別helperで解析する, 3: 各file callbackを実行してからIRを統合する]
//	  引数: [ctx: parser実行のcancel・deadline, sourceFiles: path順の解析対象, sourceFileIdentities: コメント出力時に捕捉したfile identity。未指定ならnil, afterParse: fileごとの追加処理。未指定ならnil]
//	  戻り値: [commonir.Module: 全sourceの宣言・関係をまとめたIR, error: 読込・path解決・parserに失敗した場合の説明]
//	}
func (service AnalysisService) parseProjectModules(ctx context.Context, sourceRoot string, sourceFiles []data.ProjectSourceFile, sourceFileIdentities *data.SourceFileIdentitySet, afterParse func(data.ProjectSourceFile, string, commonir.Module) error) (commonir.Module, []analyzers.ModuleDependencyUnit, []analyzers.ComponentDependencyUnit, error) {
	combined := commonir.Module{}
	moduleUnits := make([]analyzers.ModuleDependencyUnit, 0, len(sourceFiles))
	componentUnits := make([]analyzers.ComponentDependencyUnit, 0, len(sourceFiles))
	languages := map[string]struct{}{}
	for _, sourceFile := range sourceFiles {
		var source string
		if sourceFileIdentities != nil {
			var err error
			source, err = sourceFileIdentities.ReadSourceFileMatchingIdentity(sourceFile.Path)
			if err != nil {
				return commonir.Module{}, nil, nil, err
			}
		} else {
			var err error
			source, err = data.ReadSourceFile(sourceFile.Path)
			if err != nil {
				return commonir.Module{}, nil, nil, err
			}
		}
		if sourceFile.Language == "unknown" {
			return commonir.Module{}, nil, nil, fmt.Errorf("拡張子から解析言語を判定できません: %s", filepath.Ext(sourceFile.Path))
		}
		sourcePath, err := filepath.Abs(sourceFile.Path)
		if err != nil {
			return commonir.Module{}, nil, nil, fmt.Errorf("source pathを解決できません (%s): %w", sourceFile.Path, err)
		}
		payload, err := service.parserHost.ParseLanguage(ctx, sourceFile.Language, source, &sourcePath)
		if err != nil {
			return commonir.Module{}, nil, nil, fmt.Errorf("sourceを解析できません (%s): %w", sourceFile.Path, err)
		}
		if afterParse != nil {
			if err := afterParse(sourceFile, source, payload.Module); err != nil {
				return commonir.Module{}, nil, nil, err
			}
		}
		appendModule(&combined, payload.Module)
		moduleUnit, componentUnit, err := dependencyUnitsForSource(sourceRoot, sourceFile, payload.Module)
		if err != nil {
			return commonir.Module{}, nil, nil, err
		}
		moduleUnits = append(moduleUnits, moduleUnit)
		componentUnits = append(componentUnits, componentUnit)
		languages[sourceFile.Language] = struct{}{}
	}
	if len(languages) > 1 {
		combined.Language = "mixed"
	}
	return combined, moduleUnits, componentUnits, nil
}

//	{
//	  責務: [dependencyUnitsForSource: project内sourceの相対pathをpackage/moduleとcomponentの解析単位へ変換する]
//	  処理: [1: 入力fileなら親folderをrootにする, 2: 相対pathから拡張子を外してmodule名を作る, 3: packageとcomponentの識別情報を付ける]
//	  引数: [sourceRoot: project folderまたはfile入力時の親folder, sourceFile: 解析済みsourceのpathと言語, module: helperが返したCommon IR]
//	  戻り値: [moduleUnit: package依存解析単位, componentUnit: component依存解析単位, error: pathを相対化できない場合の説明]
//	}
func dependencyUnitsForSource(sourceRoot string, sourceFile data.ProjectSourceFile, module commonir.Module) (analyzers.ModuleDependencyUnit, analyzers.ComponentDependencyUnit, error) {
	relativePath, err := filepath.Rel(sourceRoot, sourceFile.Path)
	if err != nil {
		return analyzers.ModuleDependencyUnit{}, analyzers.ComponentDependencyUnit{}, fmt.Errorf("sourceをproject相対pathへ変換できません (%s): %w", sourceFile.Path, err)
	}
	moduleName := strings.TrimSuffix(relativePath, filepath.Ext(relativePath))
	moduleName = strings.ReplaceAll(moduleName, string(filepath.Separator), ".")
	isPackage := strings.HasSuffix(strings.ToLower(moduleName), ".__init__")
	if isPackage {
		moduleName = strings.TrimSuffix(moduleName, ".__init__")
	}
	if moduleName == "" || moduleName == "." {
		moduleName = filepath.Base(sourceRoot)
	}
	moduleUnit := analyzers.ModuleDependencyUnit{Name: moduleName, CommonIR: module, IsPackage: isPackage}
	componentPath := filepath.Dir(relativePath)
	componentName := "root"
	if componentPath != "." {
		componentName = strings.Split(filepath.ToSlash(componentPath), "/")[0]
	}
	if componentName == "" {
		componentName = "root"
	}
	componentUnit := analyzers.ComponentDependencyUnit{Module: moduleUnit, Component: componentName}
	return moduleUnit, componentUnit, nil
}

//	{
//	  責務: [readDeploymentTopology: project内のDockerfileとCompose設定を読み、確定したdeployment factsを統合する]
//	  処理: [1: folder入力から設定fileを探す, 2: 設定種別ごとに既存analyzerを呼び出す, 3: factsを重複なくmergeする]
//	  引数: [sourceRoot: source fileまたはproject folder, sourceIsDirectory: folder入力ならtrue, scanner: 共通除外規則を持つproject scanner]
//	  戻り値: [DeploymentTopology: project設定から確認できたdeployment facts, error: 設定の読込・解析に失敗した場合の説明]
//	}
func readDeploymentTopology(sourceRoot string, sourceIsDirectory bool, scanner data.ProjectFileScanner) (analyzers.DeploymentTopology, error) {
	if !sourceIsDirectory {
		return analyzers.DeploymentTopology{}, nil
	}
	deploymentPaths, err := scanner.DiscoverDeploymentFiles(sourceRoot)
	if err != nil {
		return analyzers.DeploymentTopology{}, err
	}
	topologies := make([]analyzers.DeploymentTopology, 0, len(deploymentPaths))
	for _, deploymentPath := range deploymentPaths {
		source, err := data.ReadSourceFile(deploymentPath)
		if err != nil {
			return analyzers.DeploymentTopology{}, err
		}
		relativePath, err := filepath.Rel(sourceRoot, deploymentPath)
		if err != nil {
			return analyzers.DeploymentTopology{}, fmt.Errorf("設定fileをproject相対pathへ変換できません (%s): %w", deploymentPath, err)
		}
		if strings.HasPrefix(strings.ToLower(filepath.Base(deploymentPath)), "dockerfile") {
			topologies = append(topologies, analyzers.ParseDockerfile(source, filepath.ToSlash(relativePath)))
			continue
		}
		topology, err := analyzers.ParseCompose(source, filepath.ToSlash(relativePath))
		if err != nil {
			return analyzers.DeploymentTopology{}, fmt.Errorf("配置設定を解析できません (%s): %w", relativePath, err)
		}
		topologies = append(topologies, topology)
	}
	return analyzers.MergeDeploymentTopologies(topologies...), nil
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
func renderAnalysisOutput(module commonir.Module, operation, sourceName string, moduleUnits []analyzers.ModuleDependencyUnit, componentUnits []analyzers.ComponentDependencyUnit, deploymentTopology analyzers.DeploymentTopology) (string, error) {
	switch operation {
	case "call-graph":
		bundle, err := generators.BuildCallGraphBundle(module, 3, nil, nil)
		if err != nil {
			return "", fmt.Errorf("呼び出し図を生成できません: %w", err)
		}
		var output strings.Builder
		output.WriteString("# 呼び出し図\n")
		if len(bundle.Diagrams) == 0 {
			output.WriteString("\n```mermaid\n")
			output.WriteString(renderers.RenderCallGraphMermaid(generators.CallGraph{}, "LR"))
			output.WriteString("```\n")
			return output.String(), nil
		}
		if len(bundle.Diagrams) == 1 {
			return "# 呼び出し図\n\n```mermaid\n" + renderers.RenderCallGraphMermaid(bundle.Diagrams[0].Graph, "LR") + "```\n", nil
		}
		for _, diagram := range bundle.Diagrams {
			fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", diagram.Name)
			output.WriteString(renderers.RenderCallGraphMermaid(diagram.Graph, "LR"))
			output.WriteString("```\n")
		}
		return output.String(), nil
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
	case "sequence-diagram":
		bundle, err := generators.BuildSequenceDiagramBundle(module, 3, 8, nil)
		if err != nil {
			return "", fmt.Errorf("シーケンス図を生成できません: %w", err)
		}
		var output strings.Builder
		output.WriteString("# シーケンス図\n")
		for _, diagram := range bundle.Diagrams {
			fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", diagram.Name)
			output.WriteString(renderers.RenderSequenceDiagramMermaid(diagram))
			output.WriteString("```\n")
		}
		return output.String(), nil
	case "object-diagram":
		bundle, err := generators.BuildObjectDiagramBundle(module, 3)
		if err != nil {
			return "", fmt.Errorf("オブジェクト図を生成できません: %w", err)
		}
		var output strings.Builder
		output.WriteString("# オブジェクト図\n")
		for _, diagram := range bundle.Diagrams {
			fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", diagram.Name)
			output.WriteString(renderers.RenderObjectDiagramMermaid(diagram))
			output.WriteString("```\n")
		}
		return output.String(), nil
	case "communication-diagram":
		bundle, err := generators.BuildCommunicationDiagramBundle(module, 3, 8, false)
		if err != nil {
			return "", fmt.Errorf("コミュニケーション図を生成できません: %w", err)
		}
		var output strings.Builder
		output.WriteString("# コミュニケーション図\n")
		for _, diagram := range bundle.Diagrams {
			fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", diagram.Name)
			output.WriteString(renderers.RenderCommunicationDiagramMermaid(diagram))
			output.WriteString("```\n")
		}
		return output.String(), nil
	case "state-diagram":
		bundle := generators.BuildStateDiagramBundle(module)
		var output strings.Builder
		output.WriteString("# 状態図\n")
		for _, diagram := range bundle.Diagrams {
			fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", diagram.Name)
			output.WriteString(renderers.RenderStateDiagramMermaid(diagram))
			output.WriteString("```\n")
		}
		return output.String(), nil
	case "timing-chart":
		bundle := generators.BuildTimingChartBundle(module)
		var output strings.Builder
		output.WriteString("# タイミングチャート\n")
		for _, chart := range bundle.Charts {
			fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", chart.Name)
			output.WriteString(renderers.RenderTimingChartMermaid(chart))
			output.WriteString("```\n")
		}
		return output.String(), nil
	case "use-case-diagram":
		bundle, err := generators.BuildUseCaseDiagramBundle(module, nil, 8)
		if err != nil {
			return "", fmt.Errorf("ユースケース図を生成できません: %w", err)
		}
		var output strings.Builder
		output.WriteString("# ユースケース図\n")
		for _, diagram := range bundle.Diagrams {
			fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", diagram.Name)
			output.WriteString(renderers.RenderUseCaseDiagramMermaid(diagram))
			output.WriteString("```\n")
		}
		return output.String(), nil
	case "responsibility-table":
		bundle, err := generators.BuildResponsibilityTableBundle(module, 3)
		if err != nil {
			return "", fmt.Errorf("責務表を生成できません: %w", err)
		}
		var output strings.Builder
		output.WriteString("# 責務表\n\n")
		for index, table := range bundle.Tables {
			if index > 0 {
				output.WriteString("\n")
			}
			fmt.Fprintf(&output, "## %s\n\n", table.Name)
			output.WriteString(renderers.RenderResponsibilityMarkdown(table.Rows))
		}
		return output.String(), nil
	case "package-diagram":
		graph := analyzers.BuildPackageDependencyGraph(moduleUnits)
		bundle, err := generators.BuildPackageDiagramBundle(graph, 3)
		if err != nil {
			return "", fmt.Errorf("パッケージ図を生成できません: %w", err)
		}
		var output strings.Builder
		output.WriteString("# パッケージ図\n")
		for _, diagram := range bundle.Diagrams {
			fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", diagram.Name)
			output.WriteString(renderers.RenderPackageDiagramMermaid(diagram))
			output.WriteString("```\n")
		}
		return output.String(), nil
	case "component-diagram":
		graph := analyzers.BuildComponentDependencyGraph(componentUnits)
		bundle, err := generators.BuildComponentDiagramBundle(graph, 3)
		if err != nil {
			return "", fmt.Errorf("コンポーネント図を生成できません: %w", err)
		}
		var output strings.Builder
		output.WriteString("# コンポーネント図\n")
		for _, diagram := range bundle.Diagrams {
			fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", diagram.Name)
			output.WriteString(renderers.RenderComponentDiagramMermaid(diagram))
			output.WriteString("```\n")
		}
		return output.String(), nil
	case "deployment-diagram":
		componentGraph := analyzers.BuildComponentDependencyGraph(componentUnits)
		topology := analyzers.MergeDeploymentTopologies(analyzers.DeploymentTopologyFromComponentGraph(componentGraph), deploymentTopology)
		bundle, err := generators.BuildDeploymentDiagramBundle(topology, 3)
		if err != nil {
			return "", fmt.Errorf("配置図を生成できません: %w", err)
		}
		var output strings.Builder
		output.WriteString("# 配置図\n")
		for _, diagram := range bundle.Diagrams {
			fmt.Fprintf(&output, "\n## %s\n\n```mermaid\n", diagram.Name)
			output.WriteString(renderers.RenderDeploymentDiagramMermaid(diagram))
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
