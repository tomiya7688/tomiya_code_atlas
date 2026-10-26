package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//	{
//	  責務: [FindApplicationRoot: executableまたはworking directoryから配布manifestのrootを見つける]
//	  処理: [1: executable隣接directoryを確認する, 2: working directoryを確認する, 3: 最初のmanifest rootを返す]
//	  引数: [executablePath: 起動したCLIのpath, workingDirectory: 呼出元のcurrent directory]
//	  戻り値: [applicationRoot: backends manifestを含むdirectory, error: manifestが見つからない場合の説明]
//	}
func FindApplicationRoot(executablePath string, workingDirectory string) (string, error) {
	candidates := []string{filepath.Dir(executablePath), workingDirectory}
	for _, candidate := range candidates {
		if hasBackendManifest(candidate) {
			absolutePath, err := filepath.Abs(candidate)
			if err != nil {
				return "", fmt.Errorf("配布directoryのpathを解決できません: %w", err)
			}
			return absolutePath, nil
		}
	}
	return "", fmt.Errorf("backends/manifest.json が見つかりません。CLI実行fileと同じ配布フォルダーにbackendsを配置してください")
}

//	{
//	  責務: [hasBackendManifest: directory内のparser helper manifestがregular fileか判定する]
//	  処理: [1: backends/manifest.jsonをstatする, 2: regular fileならtrueを返す]
//	  引数: [directory: 配布root候補]
//	  戻り値: [bool: parser helper manifestが存在すればtrue]
//	}
func hasBackendManifest(directory string) bool {
	info, err := os.Stat(filepath.Join(directory, "backends", "manifest.json"))
	return err == nil && info.Mode().IsRegular()
}

//	{
//	  責務: [ReadSourceFile: 指定されたsource fileのUTF-8 textを読み込む]
//	  処理: [1: fileを開いて全内容を読む, 2: pathを含む日本語errorまたはsourceを返す]
//	  引数: [sourcePath: 読み込むfileのpath]
//	  戻り値: [source: helperへ渡すtext, error: 読み込みに失敗した場合の説明]
//	}
func ReadSourceFile(sourcePath string) (string, error) {
	contents, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("source fileを読み込めません (%s): %w", sourcePath, err)
	}
	return string(contents), nil
}

//	{
//	  責務: [WriteTextFile: 指定folderへ一つのtext outputを安全なfile名で保存する]
//	  処理: [1: file名がfolder外を指さないことを確認する, 2: 保存folderを作る, 3: fileを書き込み絶対pathを返す]
//	  引数: [directory: output folder, fileName: folder直下に作るfile名, contents: 保存するtext]
//	  戻り値: [outputPath: 保存したfileのabsolute path, error: path検証・directory作成・書込み失敗の説明]
//	}
func WriteTextFile(directory string, fileName string, contents string) (string, error) {
	if fileName == "" || filepath.Base(fileName) != fileName || fileName == "." || fileName == ".." {
		return "", fmt.Errorf("出力file名が不正です: %q", fileName)
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("出力folderを作成できません (%s): %w", directory, err)
	}
	outputPath := filepath.Join(directory, fileName)
	if err := os.WriteFile(outputPath, []byte(contents), 0o644); err != nil {
		return "", fmt.Errorf("出力fileを書き込めません (%s): %w", outputPath, err)
	}
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("出力fileのabsolute pathを解決できません: %w", err)
	}
	return absPath, nil
}

//	{
//	  責務: [WriteTextFileAtRelativePath: root配下の相対pathを検証してtext fileを保存する]
//	  処理: [1: traversalとsymlink directoryを拒否する, 2: 親folderを作りfileを書き込む]
//	  引数: [root: 出力を許可するroot folder, relativePath: root基準のlocal path, contents: 保存するtext]
//	  戻り値: [outputPath: 保存したfileのabsolute path, error: path・directory・書込みに失敗した場合の説明]
//	}
func WriteTextFileAtRelativePath(root string, relativePath string, contents string) (string, error) {
	//	{
	//	  処理: [Clean後もroot外を指すrelative pathを拒否する]
	//	}
	if !filepath.IsLocal(relativePath) || relativePath == "." {
		return "", fmt.Errorf("出力relative pathが不正です: %q", relativePath)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("出力folderを作成できません (%s): %w", root, err)
	}
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("出力rootのabsolute pathを解決できません: %w", err)
	}
	rootPath, err = filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", fmt.Errorf("出力rootを解決できません (%s): %w", root, err)
	}
	cleanPath := filepath.Clean(relativePath)
	pathParts := strings.Split(cleanPath, string(filepath.Separator))
	currentDirectory := rootPath
	//	{
	//	  処理: [親directoryを個別に作り、既存symlinkをたどらない]
	//	}
	for _, directoryName := range pathParts[:len(pathParts)-1] {
		currentDirectory = filepath.Join(currentDirectory, directoryName)
		info, err := os.Lstat(currentDirectory)
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("出力folderを確認できません (%s): %w", currentDirectory, err)
		}
		if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			return "", fmt.Errorf("出力pathにdirectoryではない要素があります: %s", currentDirectory)
		}
		if os.IsNotExist(err) {
			if err := os.Mkdir(currentDirectory, 0o755); err != nil {
				return "", fmt.Errorf("出力folderを作成できません (%s): %w", currentDirectory, err)
			}
		}
	}
	outputPath := filepath.Join(rootPath, cleanPath)
	//	{
	//	  処理: [同名のsymlinkやdirectoryをregular fileとして上書きしない]
	//	}
	if info, err := os.Lstat(outputPath); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("出力fileを確認できません (%s): %w", outputPath, err)
	} else if err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return "", fmt.Errorf("出力先がregular fileではありません: %s", outputPath)
	}
	if err := os.WriteFile(outputPath, []byte(contents), 0o644); err != nil {
		return "", fmt.Errorf("出力fileを書き込めません (%s): %w", outputPath, err)
	}
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("出力fileのabsolute pathを解決できません: %w", err)
	}
	return absPath, nil
}
