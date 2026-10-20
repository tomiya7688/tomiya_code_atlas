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
//	  責務: [ProjectFileScanner: supported source extensionと除外directory規則を所有してproject内の解析対象fileを列挙する]
//	  状態: [extensionLanguages: extensionからlanguage IDへの対応, ignoredDirectoryNames: recursive scanから除くdirectory名]
//	}
type ProjectFileScanner struct {
	extensionLanguages    map[string]string
	ignoredDirectoryNames map[string]struct{}
}

//	{
//	  責務: [ProjectSourceFile: recursive scanで見つけたsourceのpathと言語IDをProcessへ渡す]
//	  状態: [Path: project root基準または指定rootのsource path, Language: extensionから決定したlanguage ID]
//	}
type ProjectSourceFile struct {
	Path     string
	Language string
}

//	{
//	  責務: [NewProjectFileScanner: Python版と同じsource extensionと生成directory除外規則を持つscannerを作る]
//	  処理: [1: 対象extensionと言語IDの対応を作る, 2: 除外directory名を作る, 3: scannerを返す]
//	  引数: []
//	  戻り値: [scanner: project file探索を行うscanner]
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
//	  責務: [DetectLanguage: file extensionから安定したlanguage IDを返す]
//	  処理: [1: extensionを小文字化する, 2: 登録済みlanguage IDを返し、未対応ならunknownを返す]
//	  引数: [filePath: 言語を判定するsource path]
//	  戻り値: [languageID: extensionに対応するlanguage IDまたはunknown]
//	}
func (scanner ProjectFileScanner) DetectLanguage(filePath string) string {
	languageID := scanner.extensionLanguages[strings.ToLower(filepath.Ext(filePath))]
	if languageID == "" {
		return "unknown"
	}
	return languageID
}

//	{
//	  責務: [DiscoverSupportedFiles: fileまたはproject directoryから対応言語のsourceを列挙する]
//	  処理: [1: root配下のfileを安全に列挙する, 2: supported extensionだけをlanguage ID付きで残す, 3: path順を固定して返す]
//	  引数: [rootPath: source fileまたはrecursive scanするproject directory]
//	  戻り値: [files: pathと言語IDを含む解析対象, error: rootのstatまたはdirectory走査に失敗した場合の内容]
//	}
func (scanner ProjectFileScanner) DiscoverSupportedFiles(rootPath string) ([]ProjectSourceFile, error) {
	filePaths, err := scanner.walkProjectFiles(rootPath)
	if err != nil {
		return nil, err
	}

	supportedFiles := make([]ProjectSourceFile, 0, len(filePaths))
	// 未対応extensionはCLIが解析backendを選べないため結果から除外する。
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
//	  責務: [IsDirectory: 指定pathがdirectoryか判定する]
//	  処理: [1: pathのfile情報を取得する, 2: directoryかどうかを返す]
//	  引数: [path: file種別を確認するpath]
//	  戻り値: [bool: directoryならtrue, error: pathを確認できない場合の説明]
//	}
func IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("pathを確認できません (%s): %w", path, err)
	}
	return fileInfo.IsDir(), nil
}

//	{
//	  責務: [DiscoverDeploymentFiles: project directoryからDocker・Compose・Kubernetes設定候補を列挙する]
//	  処理: [1: directory root以外を空結果にする, 2: 共通除外規則で配下fileを列挙する, 3: 設定名またはYAML extensionで候補を選び順序を固定する]
//	  引数: [rootPath: deployment設定を探索するproject directory]
//	  戻り値: [filePaths: deployment解析候補のpath, error: directory走査に失敗した場合の内容]
//	}
func (scanner ProjectFileScanner) DiscoverDeploymentFiles(rootPath string) ([]string, error) {
	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("inspect project root %q: %w", rootPath, err)
	}
	if !rootInfo.IsDir() {
		return []string{}, nil
	}

	filePaths, err := scanner.walkProjectFiles(rootPath)
	if err != nil {
		return nil, err
	}
	deploymentPaths := make([]string, 0)
	// deployment候補は通常source fileと別operationで処理するため、設定fileだけを選ぶ。
	for _, filePath := range filePaths {
		if isDeploymentConfiguration(filePath) {
			deploymentPaths = append(deploymentPaths, filePath)
		}
	}
	return deploymentPaths, nil
}

//	{
//	  責務: [walkProjectFiles: root fileまたはdirectoryからregular fileを再帰列挙し、生成・tooling directoryを除外する]
//	  処理: [1: rootの種類を検証する, 2: ignored directoryへ入らずregular fileを収集する, 3: 大文字小文字に依存しないpath順で返す]
//	  引数: [rootPath: 単一fileまたはproject directory]
//	  戻り値: [filePaths: 列挙したfile path, error: rootやdirectoryの読込に失敗した場合の内容]
//	}
func (scanner ProjectFileScanner) walkProjectFiles(rootPath string) ([]string, error) {
	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("inspect project root %q: %w", rootPath, err)
	}
	if !rootInfo.IsDir() {
		if rootInfo.Mode().IsRegular() {
			return []string{rootPath}, nil
		}
		return nil, fmt.Errorf("project root is not a regular file or directory: %q", rootPath)
	}

	filePaths := make([]string, 0)
	// WalkDirはdirectoryを決定順で巡回し、ignored treeは潜らずfile system symlinkも追跡しない。
	err = filepath.WalkDir(rootPath, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("scan project path %q: %w", filePath, walkErr)
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
			return fmt.Errorf("inspect project file %q: %w", filePath, err)
		}
		if fileInfo.Mode().IsRegular() {
			filePaths = append(filePaths, filePath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 結果順をfilesystem依存から切り離し、CLIの再現可能な出力順を保証する。
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
//	  責務: [isDeploymentConfiguration: file名またはYAML extensionからdeployment設定候補を判定する]
//	  処理: [1: basenameを小文字化する, 2: DockerfileとCompose固定名を確認する, 3: YAML extensionを確認する]
//	  引数: [filePath: 判定対象のproject file]
//	  戻り値: [isConfiguration: deployment解析候補ならtrue]
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
