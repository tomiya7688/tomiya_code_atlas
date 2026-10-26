package data

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
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
	outputPath, err := prepareTextOutputPath(root, relativePath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(outputPath, []byte(contents), 0o644); err != nil {
		return "", fmt.Errorf("出力fileを書き込めません (%s): %w", outputPath, err)
	}
	return outputPath, nil
}

//	{
//	  責務: [WriteTextFileAtRelativePathProtectingSources: sourceを直接変更せず、検証済みのoutputを配置する,]
//	  処理: [1: 相対pathと既存outputを検証し読み取り専用outputを拒否する, 2: 同じdirectoryの一時fileへ内容を書く, 3: 再検証後にrenameで配置する,]
//	  引数: [root: 出力を許可するroot folder, relativePath: root基準のlocal path, contents: 保存するtext, sourceIdentities: 上書きを防ぐ入力source一覧,]
//	  戻り値: [outputPath: 保存したfileのabsolute path, error: path・source衝突・読み取り専用output・書込み・renameに失敗した場合の説明,]
//	  注意: [既存fileのmode bitsは維持しますがowner/group・ACL・xattrは引き継ぎません, POSIXでは内容書込み中の一時fileを0600に制限しますがWindowsでは出力directoryのACLを継承します, 電源断やOS停止後の耐久性を保証せず既存outputが欠落・破損する可能性があります, 別processによるdirectory差替えやsource読込中の同時書込みは保証対象外です, 強制終了で残る.tmpはsource scannerの解析対象外です,]
//	}
func WriteTextFileAtRelativePathProtectingSources(root string, relativePath string, contents string, sourceIdentities SourceFileIdentitySet) (string, error) {
	if err := sourceIdentities.validateHasSources(); err != nil {
		return "", err
	}
	outputPath, err := prepareTextOutputPath(root, relativePath)
	if err != nil {
		return "", err
	}
	if _, _, err := validateProtectedTextOutput(outputPath, sourceIdentities); err != nil {
		return "", err
	}
	temporaryFile, newOutputMode, err := createTemporaryTextFile(filepath.Dir(outputPath))
	if err != nil {
		return "", fmt.Errorf("一時出力fileを作成できません (%s): %w", outputPath, err)
	}
	temporaryPath := temporaryFile.Name()
	temporaryFileExists := true
	defer func() {
		if temporaryFile != nil {
			_ = temporaryFile.Close()
		}
		if temporaryFileExists {
			_ = os.Remove(temporaryPath)
		}
	}()
	//	{
	//	  処理: [contentsを一時fileへ書き込み、短い書込みを検出する,]
	//	}
	written, err := temporaryFile.Write([]byte(contents))
	if err != nil {
		return "", fmt.Errorf("一時出力fileを書き込めません (%s): %w", temporaryPath, err)
	}
	if written != len(contents) {
		return "", fmt.Errorf("一時出力fileを書き込めません (%s): %w", temporaryPath, io.ErrShortWrite)
	}
	//	{
	//	  処理: [rename前に現在のoutput modeを調べ、新規outputは作成時のmodeを使う,]
	//	}
	outputMode, outputExists, err := validateProtectedTextOutput(outputPath, sourceIdentities)
	if err != nil {
		return "", err
	}
	modeToApply := newOutputMode
	if outputExists {
		modeToApply = outputMode
	}
	if err := temporaryFile.Chmod(modeToApply); err != nil {
		return "", fmt.Errorf("一時出力fileのpermissionを設定できません (%s): %w", temporaryPath, err)
	}
	if err := temporaryFile.Close(); err != nil {
		temporaryFile = nil
		return "", fmt.Errorf("一時出力fileを閉じられません (%s): %w", temporaryPath, err)
	}
	temporaryFile = nil
	if _, _, err := validateProtectedTextOutput(outputPath, sourceIdentities); err != nil {
		return "", err
	}
	//	{
	//	  処理: [source fileを開いたまま書き換えないよう、同一directoryの一時fileでoutputを置き換える,]
	//	}
	if err := os.Rename(temporaryPath, outputPath); err != nil {
		return "", fmt.Errorf("一時出力fileを配置できません (%s): %w", outputPath, err)
	}
	temporaryFileExists = false
	return outputPath, nil
}

//	{
//	  責務: [validateProtectedTextOutput: output pathが入力sourceやunsafe fileを指さないことを確認する,]
//	  処理: [1: dangling symlinkとregular file以外を拒否する, 2: source pathおよびfile identityを照合する,]
//	  引数: [outputPath: 検証するabsolute path, sourceIdentities: 上書きを防ぐ入力source一覧,]
//	  戻り値: [outputMode: 既存outputのpermission bits, outputExists: 既存outputがあればtrue, error: path確認またはsource衝突の場合の説明,]
//	}
func validateProtectedTextOutput(outputPath string, sourceIdentities SourceFileIdentitySet) (os.FileMode, bool, error) {
	outputInfo, err := os.Lstat(outputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, sourceIdentities.validateOutputFileDoesNotOverwriteSource(outputPath, nil)
		}
		return 0, false, fmt.Errorf("出力fileを確認できません (%s): %w", outputPath, err)
	}
	if outputInfo.Mode()&os.ModeSymlink != 0 || !outputInfo.Mode().IsRegular() {
		return 0, false, fmt.Errorf("出力先がregular fileではありません: %s", outputPath)
	}
	outputMode := outputInfo.Mode().Perm()
	if outputMode&0o222 == 0 {
		return 0, false, fmt.Errorf("既存の出力fileが書込み不可です: %s", outputPath)
	}
	if err := sourceIdentities.validateOutputFileDoesNotOverwriteSource(outputPath, outputInfo); err != nil {
		return 0, false, err
	}
	return outputMode, true, nil
}

//	{
//	  責務: [createTemporaryTextFile: umaskを反映した既定modeを記録しPOSIXで書込中のpermissionを絞る一時fileを作る,]
//	  処理: [1: 0644とumaskで排他的にfileを作る, 2: 既定modeを記録してPOSIXで書込み前に0600へ制限する,]
//	  引数: [directory: 一時fileを配置するdirectory,]
//	  戻り値: [temporaryFile: 作成した一時file, defaultMode: umask反映後の新規output mode, error: 一時fileを作れない場合の説明,]
//	}
func createTemporaryTextFile(directory string) (*os.File, os.FileMode, error) {
	for attempt := 0; attempt < 10; attempt++ {
		var randomBytes [16]byte
		if _, err := rand.Read(randomBytes[:]); err != nil {
			return nil, 0, fmt.Errorf("一時file名を作成できません: %w", err)
		}
		fileName := ".tca-output-" + hex.EncodeToString(randomBytes[:]) + ".tmp"
		filePath := filepath.Join(directory, fileName)
		temporaryFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			fileInfo, err := temporaryFile.Stat()
			if err != nil {
				_ = temporaryFile.Close()
				_ = os.Remove(filePath)
				return nil, 0, fmt.Errorf("一時出力fileのmodeを確認できません (%s): %w", filePath, err)
			}
			defaultMode := fileInfo.Mode().Perm()
			if err := temporaryFile.Chmod(0o600); err != nil {
				_ = temporaryFile.Close()
				_ = os.Remove(filePath)
				return nil, 0, fmt.Errorf("一時出力fileを非公開にできません (%s): %w", filePath, err)
			}
			return temporaryFile, defaultMode, nil
		}
		if !os.IsExist(err) {
			return nil, 0, err
		}
	}
	return nil, 0, fmt.Errorf("一時file名の重複が続いたため作成を中止しました")
}

//	{
//	  責務: [prepareTextOutputPath: 相対pathを検証し、安全なroot配下の出力pathを用意する,]
//	  処理: [1: rootと相対pathを検証する, 2: parent folderをsymlinkなしで作る, 3: output候補の種類を確認する,]
//	  引数: [root: 出力を許可するroot folder, relativePath: root基準のlocal path,]
//	  戻り値: [outputPath: root配下のabsolute path, error: path検証・directory作成に失敗した場合の説明,]
//	}
func prepareTextOutputPath(root string, relativePath string) (string, error) {
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
	return outputPath, nil
}
