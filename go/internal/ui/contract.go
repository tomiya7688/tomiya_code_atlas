package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
)

//	{
//	  責務: [DiagramGenerator: Goの画面からProcessへ図生成を依頼する契約を定義する,],
//	}
type DiagramGenerator interface {
	Generate(ctx context.Context, request process.DiagramRequest) ([]string, error)
}

//	{
//	  責務: [operationDefinition: 画面で選べる図のIDと表示名を保持する,],
//	  状態: [id: Processへ渡す処理ID, label: 利用者に表示する日本語名,],
//	}
type operationDefinition struct {
	id    string
	label string
}

//	{
//	  責務: [supportedOperationDefinitions: Processが対応する図に画面用の日本語名を付ける,],
//	  処理: [1: Processの対応一覧を取得する, 2: 各IDに表示名を付けて返す,],
//	  戻り値: [definitions: Go GUIで選べる図と表示名の一覧,],
//	}
func supportedOperationDefinitions() []operationDefinition {
	labels := map[string]string{
		process.OperationCallGraph:    "呼び出し図（Mermaid）",
		process.OperationClassDiagram: "クラス図（Mermaid）",
	}
	operations := process.SupportedDiagramOperations()
	definitions := make([]operationDefinition, 0, len(operations))
	for _, operation := range operations {
		definitions = append(definitions, operationDefinition{id: operation, label: labels[operation]})
	}
	return definitions
}

//	{
//	  責務: [buildDiagramRequest: 画面の入力からProcessへ渡す依頼を作り、不足値・重複・未対応の図を検証する,],
//	  処理: [1: 入力元と保存先を確認する, 2: 図が一つ以上選ばれ、すべて対応済みか確認する, 3: 検証済みの依頼を返す,],
//	  引数: [sourcePath: 解析するファイルまたはプロジェクトフォルダー, outputDirectory: 結果を保存するフォルダー, operations: 利用者が選んだ処理ID,],
//	  戻り値: [request: Processへ渡す依頼データ, error: 入力不足または無効な図を選んだ場合の説明,],
//	}
func buildDiagramRequest(sourcePath, outputDirectory string, operations []string) (process.DiagramRequest, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	outputDirectory = strings.TrimSpace(outputDirectory)
	if sourcePath == "" {
		return process.DiagramRequest{}, fmt.Errorf("先に解析するファイルまたはプロジェクトフォルダーを選択してください")
	}
	if outputDirectory == "" {
		return process.DiagramRequest{}, fmt.Errorf("先に結果の保存先フォルダーを選択してください")
	}
	if len(operations) == 0 {
		return process.DiagramRequest{}, fmt.Errorf("実行する解析を1つ以上選択してください")
	}
	seenOperations := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		if !process.IsDiagramOperationSupported(operation) {
			return process.DiagramRequest{}, fmt.Errorf("現在のGo版では利用できない図です: %s", operation)
		}
		if _, duplicate := seenOperations[operation]; duplicate {
			return process.DiagramRequest{}, fmt.Errorf("同じ解析が重複して選択されています: %s", operation)
		}
		seenOperations[operation] = struct{}{}
	}
	return process.DiagramRequest{
		SourcePath: sourcePath, OutputDirectory: outputDirectory,
		Operations: append([]string(nil), operations...),
	}, nil
}
