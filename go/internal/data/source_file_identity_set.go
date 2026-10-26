package data

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

//	{
//	  責務: [SourceFileIdentitySet: 一回の解析対象source fileのpathとfile identityを保持する,]
//	  状態: [files: 絶対pathとos.FileInfoを対応させた入力source一覧,]
//	}
type SourceFileIdentitySet struct {
	files  []sourceFileIdentity
	byPath map[string]os.FileInfo
}

//	{
//	  責務: [sourceFileIdentity: source pathとfile identityを組にする,]
//	  状態: [path: sourceのabsolute path, info: sourceのfile情報,]
//	}
type sourceFileIdentity struct {
	path string
	info os.FileInfo
}

//	{
//	  責務: [NewSourceFileIdentitySet: 解析対象source fileのidentityを一度だけ取得する,]
//	  処理: [1: 各pathをabsolute pathへ解決する, 2: regular file情報を集める,]
//	  引数: [sourcePaths: 解析対象source file一覧,]
//	  戻り値: [SourceFileIdentitySet: 出力衝突検査に使う入力file一覧, error: pathまたはfile情報を取得できない場合の説明,]
//	}
func NewSourceFileIdentitySet(sourcePaths []string) (SourceFileIdentitySet, error) {
	if len(sourcePaths) == 0 {
		return SourceFileIdentitySet{}, fmt.Errorf("source file一覧が空です")
	}
	files := make([]sourceFileIdentity, 0, len(sourcePaths))
	fileInfoByPath := make(map[string]os.FileInfo, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		//	{
		//	  処理: [source pathをabsolute pathへ解決する,]
		//	}
		absolutePath, err := filepath.Abs(sourcePath)
		if err != nil {
			return SourceFileIdentitySet{}, fmt.Errorf("source fileのpathを解決できません (%s): %w", sourcePath, err)
		}
		//	{
		//	  処理: [source pathを開き、handleからregular fileのidentityを取得する,]
		//	}
		sourceFile, err := os.Open(absolutePath)
		if err != nil {
			return SourceFileIdentitySet{}, fmt.Errorf("source fileを開けません (%s): %w", absolutePath, err)
		}
		fileInfo, err := sourceFile.Stat()
		if err != nil {
			_ = sourceFile.Close()
			return SourceFileIdentitySet{}, fmt.Errorf("source fileを確認できません (%s): %w", absolutePath, err)
		}
		if err := sourceFile.Close(); err != nil {
			return SourceFileIdentitySet{}, fmt.Errorf("source fileを閉じられません (%s): %w", absolutePath, err)
		}
		if !fileInfo.Mode().IsRegular() {
			return SourceFileIdentitySet{}, fmt.Errorf("source fileがregular fileではありません: %s", absolutePath)
		}
		files = append(files, sourceFileIdentity{path: absolutePath, info: fileInfo})
		fileInfoByPath[normalizeFilePath(absolutePath)] = fileInfo
	}
	return SourceFileIdentitySet{files: files, byPath: fileInfoByPath}, nil
}

//	{
//	  責務: [ReadSourceFileMatchingIdentity: 解析対象として捕捉したfile identityと一致するhandleからsourceを読む,]
//	  処理: [1: source pathに対応する捕捉identityを探す, 2: fileを開いてidentityを照合する, 3: 同じhandleからsourceを読む,]
//	  引数: [sourcePath: scannerが検出した解析対象source path,]
//	  戻り値: [source: parserへ渡すsource内容, error: path差替え・読込・file情報取得に失敗した場合の説明,]
//	}
func (identities SourceFileIdentitySet) ReadSourceFileMatchingIdentity(sourcePath string) (string, error) {
	if err := identities.validateHasSources(); err != nil {
		return "", err
	}
	absolutePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("source fileのpathを解決できません (%s): %w", sourcePath, err)
	}
	capturedInfo, found := identities.byPath[normalizeFilePath(absolutePath)]
	if !found {
		return "", fmt.Errorf("source file identityが解析対象一覧にありません: %s", absolutePath)
	}
	openFile, err := os.Open(absolutePath)
	if err != nil {
		return "", fmt.Errorf("source fileを開けません (%s): %w", absolutePath, err)
	}
	fileInfo, err := openFile.Stat()
	if err != nil {
		_ = openFile.Close()
		return "", fmt.Errorf("source fileを確認できません (%s): %w", absolutePath, err)
	}
	if !fileInfo.Mode().IsRegular() {
		_ = openFile.Close()
		return "", fmt.Errorf("source fileがregular fileではありません: %s", absolutePath)
	}
	if !os.SameFile(capturedInfo, fileInfo) {
		_ = openFile.Close()
		return "", fmt.Errorf("source fileがidentity取得後に差し替えられました: %s", absolutePath)
	}
	contents, err := io.ReadAll(openFile)
	if err != nil {
		_ = openFile.Close()
		return "", fmt.Errorf("source fileを読み込めません (%s): %w", absolutePath, err)
	}
	if err := openFile.Close(); err != nil {
		return "", fmt.Errorf("source fileを閉じられません (%s): %w", absolutePath, err)
	}
	return string(contents), nil
}

//	{
//	  責務: [validateOutputFileDoesNotOverwriteSource: output候補が入力sourceのどれかと同じfileでないことを確認する,]
//	  処理: [1: output pathをabsolute pathへ解決する, 2: 全sourceとpathおよびfile identityを比較する,]
//	  引数: [outputPath: 保存先path, outputInfo: 保存先が存在する場合のfile情報,]
//	  戻り値: [error: path確認またはsource衝突の場合の説明,]
//	}
func (identities SourceFileIdentitySet) validateOutputFileDoesNotOverwriteSource(outputPath string, outputInfo os.FileInfo) error {
	if err := identities.validateHasSources(); err != nil {
		return err
	}
	if outputInfo != nil && !outputInfo.Mode().IsRegular() {
		return fmt.Errorf("output pathがregular fileではありません: %s", outputPath)
	}
	absoluteOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("output pathを解決できません (%s): %w", outputPath, err)
	}
	if _, isInputPath := identities.byPath[normalizeFilePath(absoluteOutput)]; isInputPath {
		return fmt.Errorf("出力fileが解析対象source fileと同じです: %s", absoluteOutput)
	}
	if outputInfo == nil {
		return nil
	}
	//	{
	//	  処理: [既存outputのhardlink identityを全入力sourceと比較する,]
	//	}
	for _, source := range identities.files {
		if os.SameFile(source.info, outputInfo) {
			return fmt.Errorf("出力fileが解析対象source fileと同じです: %s", source.path)
		}
	}
	return nil
}

//	{
//	  責務: [validateHasSources: 出力保護に使うsource identityが設定されていることを確認する,]
//	  戻り値: [error: source一覧が空の場合の説明,]
//	}
func (identities SourceFileIdentitySet) validateHasSources() error {
	if len(identities.files) == 0 {
		return fmt.Errorf("source file identityが設定されていません")
	}
	return nil
}

//	{
//	  責務: [normalizeFilePath: pathを大文字小文字を保ってcleanし比較用にそろえる,]
//	  引数: [path: cleanするfile path,]
//	  戻り値: [string: cleanしたpath,]
//	}
func normalizeFilePath(path string) string {
	return filepath.Clean(path)
}
