package data

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//	{
//	  責務: [ProjectFileScanner: 対応するソース拡張子と除外ルールを管理し、解析対象のファイルを探す,],
//	  状態: [extensionLanguages: 拡張子と解析言語の対応表, ignoredDirectoryNames: 探索対象から除くフォルダー名,],
//	}
type ProjectFileScanner struct {
	extensionLanguages    map[string]string
	ignoredDirectoryNames map[string]struct{}
}

//	{
//	  責務: [ProjectSourceFile: 探索で見つけたソースのパスと言語IDをProcessへ渡す,],
//	  状態: [Path: 指定された起点から見つかったソースのパス, Language: 拡張子から決定した言語ID,],
//	}
type ProjectSourceFile struct {
	Path     string
	Language string
}

//	{
//	  責務: [NewProjectFileScanner: 対応言語の拡張子と除外ルールを備えた探索器を作る,],
//	  処理: [1: 拡張子と言語IDの対応表を作る, 2: 探索から除くフォルダー名を設定する, 3: 設定済みの探索器を返す,],
//	  戻り値: [scanner: ソースファイルを探す探索器,],
//	}
func NewProjectFileScanner() ProjectFileScanner {
	return ProjectFileScanner{
		extensionLanguages: map[string]string{
			".py": "python", ".gd": "gdscript", ".cs": "csharp",
			".cpp": "cpp", ".cc": "cpp", ".cxx": "cpp", ".hpp": "cpp", ".hxx": "cpp",
			".java": "java", ".go": "go", ".yml": "yaml", ".yaml": "yaml",
		},
		ignoredDirectoryNames: map[string]struct{}{
			".git": {}, ".dev": {}, ".codex": {}, ".venv": {}, "venv": {},
			"build": {}, "dist": {}, "__pycache__": {},
		},
	}
}

//	{
//	  責務: [SupportedFileExtensions: ファイル選択画面で使う拡張子を昇順で返す,],
//	  戻り値: [extensions: 探索器が対応する拡張子の複製,],
//	}
func (scanner ProjectFileScanner) SupportedFileExtensions() []string {
	extensions := make([]string, 0, len(scanner.extensionLanguages))
	for extension := range scanner.extensionLanguages {
		extensions = append(extensions, extension)
	}
	sort.Strings(extensions)
	return extensions
}

//	{
//	  責務: [DetectLanguage: ファイルの拡張子から解析言語を決める,],
//	  処理: [1: 拡張子を小文字にそろえる, 2: 対応する言語IDを返し、未対応ならunknownを返す,],
//	  引数: [filePath: 言語を判定するソースのパス,],
//	  戻り値: [languageID: 拡張子に対応する言語IDまたはunknown,],
//	}
func (scanner ProjectFileScanner) DetectLanguage(filePath string) string {
	languageID := scanner.extensionLanguages[strings.ToLower(filepath.Ext(filePath))]
	if languageID == "" {
		return "unknown"
	}
	return languageID
}

//	{
//	  責務: [DiscoverSupportedFiles: ファイルまたはプロジェクトフォルダーから対応言語のソースを探す,],
//	  処理: [1: 起点のファイルまたは配下のファイルを列挙する, 2: 対応拡張子のファイルに言語IDを付ける, 3: パス順に並べて返す,],
//	  引数: [rootPath: ソースファイルまたは再帰探索するプロジェクトフォルダー,],
//	  戻り値: [files: パスと言語IDを含む解析対象, error: 起点の確認またはフォルダー走査に失敗した理由,],
//	}
func (scanner ProjectFileScanner) DiscoverSupportedFiles(rootPath string) ([]ProjectSourceFile, error) {
	filePaths, err := scanner.walkProjectFiles(rootPath)
	if err != nil {
		return nil, err
	}

	supportedFiles := make([]ProjectSourceFile, 0, len(filePaths))
	// 未対応の拡張子はCLIで解析器を選べないため、結果から除外する。
	for _, filePath := range filePaths {
		languageID := scanner.DetectLanguage(filePath)
		if languageID == "unknown" {
			continue
		}
		supportedFiles = append(supportedFiles, ProjectSourceFile{Path: filePath, Language: languageID})
	}
	return supportedFiles, nil
}

//	{
//	  責務: [IsDirectory: 指定されたパスがフォルダーか判定する,],
//	  処理: [1: パスの情報を取得する, 2: フォルダーかどうかを返す,],
//	  引数: [path: 種類を確認するパス,],
//	  戻り値: [bool: フォルダーならtrue, error: パスの情報を取得できない場合の説明,],
//	}
func IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("パスを確認できません (%s): %w", path, err)
	}
	return fileInfo.IsDir(), nil
}

//	{
//	  責務: [DiscoverDeploymentFiles: プロジェクトフォルダーからDocker・Compose・Kubernetes設定の候補を探す,],
//	  処理: [1: フォルダー以外の起点には空結果を返す, 2: 共通の除外ルールを守って配下を列挙する, 3: 設定名またはYAML拡張子で候補を選び、順序を固定する,],
//	  引数: [rootPath: deployment設定を探すプロジェクトフォルダー,],
//	  戻り値: [filePaths: deployment解析候補のパス, error: フォルダー走査に失敗した理由,],
//	}
func (scanner ProjectFileScanner) DiscoverDeploymentFiles(rootPath string) ([]string, error) {
	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("プロジェクトの起点を確認できません (%q): %w", rootPath, err)
	}
	if !rootInfo.IsDir() {
		return []string{}, nil
	}

	filePaths, err := scanner.walkProjectFiles(rootPath)
	if err != nil {
		return nil, err
	}
	deploymentPaths := make([]string, 0)
	// デプロイ設定は通常のソース解析と別に扱うため、設定ファイルだけを選ぶ。
	for _, filePath := range filePaths {
		if isDeploymentConfiguration(filePath) {
			deploymentPaths = append(deploymentPaths, filePath)
		}
	}
	return deploymentPaths, nil
}

//	{
//	  責務: [walkProjectFiles: 指定された起点からファイルを再帰的に探し、生成物やツール用フォルダーを除外する,],
//	  処理: [1: 起点の種類を確認する, 2: 除外対象へ入らずファイルを収集する, 3: 大文字小文字に依存しない順序で返す,],
//	  引数: [rootPath: 単一ファイルまたはプロジェクトフォルダー,],
//	  戻り値: [filePaths: 見つかったファイルのパス, error: 起点またはフォルダーの読込に失敗した理由,],
//	}
func (scanner ProjectFileScanner) walkProjectFiles(rootPath string) ([]string, error) {
	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("プロジェクトの起点を確認できません (%q): %w", rootPath, err)
	}
	if !rootInfo.IsDir() {
		if rootInfo.Mode().IsRegular() {
			return []string{rootPath}, nil
		}
		return nil, fmt.Errorf("プロジェクトの起点は通常ファイルまたはフォルダーではありません: %q", rootPath)
	}

	filePaths := make([]string, 0)
	// WalkDirはフォルダーを決定順で巡回し、除外対象へ入らず、ファイルシステムのシンボリックリンクも追跡しない。
	err = filepath.WalkDir(rootPath, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("プロジェクト内のパスを走査できません (%q): %w", filePath, walkErr)
		}
		if filePath == rootPath {
			return nil
		}
		if entry.IsDir() {
			if _, ignored := scanner.ignoredDirectoryNames[entry.Name()]; ignored {
				return filepath.SkipDir
			}
			return nil
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("プロジェクト内のファイルを確認できません (%q): %w", filePath, err)
		}
		if fileInfo.Mode().IsRegular() {
			filePaths = append(filePaths, filePath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// ファイルシステムに依存しない順序にそろえ、CLIの再現可能な出力を保証する。
	sort.Slice(filePaths, func(left int, right int) bool {
		leftPath := strings.ToLower(filepath.ToSlash(filePaths[left]))
		rightPath := strings.ToLower(filepath.ToSlash(filePaths[right]))
		if leftPath == rightPath {
			return filePaths[left] < filePaths[right]
		}
		return leftPath < rightPath
	})
	return filePaths, nil
}

//	{
//	  責務: [isDeploymentConfiguration: ファイル名やYAML拡張子からdeployment設定の候補か判定する,],
//	  処理: [1: ファイル名を小文字にそろえる, 2: DockerfileとComposeの固定名を確認する, 3: YAML拡張子を確認する,],
//	  引数: [filePath: 判定するプロジェクトファイル,],
//	  戻り値: [isConfiguration: deployment解析の候補ならtrue,],
//	}
func isDeploymentConfiguration(filePath string) bool {
	baseName := strings.ToLower(filepath.Base(filePath))
	if strings.HasPrefix(baseName, "dockerfile") {
		return true
	}
	switch baseName {
	case "compose.yml", "compose.yaml", "docker-compose.yml", "docker-compose.yaml":
		return true
	}
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".yml", ".yaml":
		return true
	default:
		return false
	}
}
