package process

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
	"github.com/tomiya7688/tomiya_code_atlas/internal/generators"
	"github.com/tomiya7688/tomiya_code_atlas/internal/languages"
)

//	{
//	  責務: [CommentedSourceWriter: 言語別comment候補をsource copyへ適用して保存する]
//	  状態: [sourceRoot: 元source fileまたはproject folder, outputDirectory: copyの保存先, sourceIsDirectory: 入力がproject folderか]
//	}
type CommentedSourceWriter struct {
	sourceRoot        string
	outputDirectory   string
	sourceIsDirectory bool
}

//	{
//	  責務: [WriteCommentedCopy: 一つのsource fileからcommented copyを生成して保存する]
//	  処理: [1: 言語別comment候補を生成する, 2: source copyへ適用する, 3: 元のfolder構成を保って保存する]
//	  引数: [sourceFile: scannerが検出したfileと言語, source: 読み込み済みの元text, module: file単位のCommon IR]
//	  戻り値: [string: 保存したcopyのabsolute path, error: 候補・構文・path・保存に失敗した場合の説明]
//	}
func (writer CommentedSourceWriter) WriteCommentedCopy(sourceFile data.ProjectSourceFile, source string, module commonir.Module) (string, error) {
	candidates, err := generators.GenerateCommentCandidates(module)
	if err != nil {
		return "", fmt.Errorf("comment候補を生成できません (%s): %w", sourceFile.Path, err)
	}
	syntax, err := languages.CommentSyntaxFor(sourceFile.Language)
	if err != nil {
		return "", err
	}
	commentedSource, err := syntax.Insert(source, candidates)
	if err != nil {
		return "", fmt.Errorf("commentをsource copyへ挿入できません (%s): %w", sourceFile.Path, err)
	}
	relativePath, err := writer.outputRelativePath(sourceFile.Path)
	if err != nil {
		return "", err
	}
	return data.WriteTextFileAtRelativePath(writer.outputDirectory, relativePath, commentedSource)
}

//	{
//	  責務: [outputRelativePath: 単一file名またはproject folderからの相対構成を使ったcomment pathを作る]
//	  処理: [1: 単一fileは別名にする, 2: project fileはroot配下の相対pathを保つ]
//	  引数: [sourcePath: scanで見つけたsource file]
//	  戻り値: [string: output directoryからの相対path, error: root外のpathまたはpath解決に失敗した場合の説明]
//	}
func (writer CommentedSourceWriter) outputRelativePath(sourcePath string) (string, error) {
	if !writer.sourceIsDirectory {
		fileName := filepath.Base(sourcePath)
		extension := filepath.Ext(fileName)
		stem := strings.TrimSuffix(fileName, extension)
		return filepath.Join("comments", stem+"_commented"+extension), nil
	}
	absoluteRoot, err := filepath.Abs(writer.sourceRoot)
	if err != nil {
		return "", fmt.Errorf("source rootを解決できません (%s): %w", writer.sourceRoot, err)
	}
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("source fileを解決できません (%s): %w", sourcePath, err)
	}
	relativePath, err := filepath.Rel(absoluteRoot, absSource)
	if err != nil || !filepath.IsLocal(relativePath) {
		return "", fmt.Errorf("source fileがinput folderの外にあります: %s", sourcePath)
	}
	return filepath.Join("comments", relativePath), nil
}
