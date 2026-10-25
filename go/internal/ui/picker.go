package ui

import (
	"errors"
	"fmt"
	"io"

	"gioui.org/x/explorer"
	"github.com/tomiya7688/tomiya_code_atlas/internal/process"
)

//	{
//	  責務: [startPicker: OS標準の選択画面を別スレッドで開き、ウィンドウの応答を保つ,],
//	  処理: [1: 選択画面の多重起動を防ぐ, 2: 種類に応じた選択画面を開く, 3: 選択結果を画面へ通知する,],
//	  引数: [kind: 解析対象ファイル・解析対象フォルダー・保存先フォルダーの区別,],
//	}
func (application *desktopApp) startPicker(kind pickerKind) {
	if application.analysisPending {
		application.status = "解析中は入力や保存先を変更できません。完了後に選択してください。"
		application.diagnostic = ""
		return
	}
	if application.pickerPending {
		application.status = "現在の選択画面を閉じてから、別の項目を選択してください。"
		application.diagnostic = ""
		return
	}
	application.pickerPending = true
	application.diagnostic = ""
	application.status = pickerStatus(kind)
	go func() {
		result := pickerResult{kind: kind}
		switch kind {
		case pickerSourceFile:
			selectedFile, err := application.explorer.ChooseFile(process.SupportedSourceFileExtensions()...)
			result.err = err
			if err == nil {
				result.path, result.err = selectedFilePath(selectedFile)
			}
		case pickerSourceFolder, pickerOutputFolder:
			result.path, result.err = application.explorer.ChooseFolder()
		default:
			result.err = fmt.Errorf("種類を認識できないフォルダー選択画面です: %d", kind)
		}
		application.pickerResults <- result
		application.window.Invalidate()
	}()
}

//	{
//	  責務: [pickerStatus: 開いている選択画面の種類を利用者へ伝える,],
//	  引数: [kind: OS標準画面で選んでいる対象の種類,],
//	  戻り値: [string: 選択中の操作を説明する状態表示,],
//	}
func pickerStatus(kind pickerKind) string {
	switch kind {
	case pickerSourceFile:
		return "解析するファイルを選択してください。"
	case pickerSourceFolder:
		return "解析するプロジェクトフォルダーを選択してください。"
	case pickerOutputFolder:
		return "結果の保存先フォルダーを選択してください。"
	default:
		return "ファイル選択画面を開いています。"
	}
}

//	{
//	  責務: [selectedFilePath: Gioが返した選択ファイルからパスを取得し、読み取り用ハンドルを閉じる,],
//	  引数: [selectedFile: Gioのファイル選択画面から受け取った読み取り可能な値,],
//	  戻り値: [path: 選択されたソースのパス, error: パス取得またはハンドルの終了に失敗した理由,],
//	}
func selectedFilePath(selectedFile io.ReadCloser) (selectedPath string, selectionError error) {
	if selectedFile == nil {
		return "", fmt.Errorf("ファイル選択画面から結果を受け取れませんでした")
	}
	defer func() {
		if closeError := selectedFile.Close(); selectionError == nil && closeError != nil {
			selectedPath = ""
			selectionError = fmt.Errorf("選択したファイルを閉じられません: %w", closeError)
		}
	}()
	namedFile, ok := selectedFile.(interface{ Name() string })
	if !ok {
		return "", fmt.Errorf("選択したファイルのパスを取得できませんでした")
	}
	if namedFile.Name() == "" {
		return "", fmt.Errorf("選択したファイルのパスが空です")
	}
	return namedFile.Name(), nil
}

//	{
//	  責務: [applyPickerResult: ファイル選択の結果を入力欄へ反映し、取消と失敗を分けて表示する,],
//	  引数: [result: ファイル選択処理から返された対象・パス・エラー,],
//	}
func (application *desktopApp) applyPickerResult(result pickerResult) {
	if errors.Is(result.err, explorer.ErrUserDecline) {
		application.status = "選択を取り消しました。必要な入力または保存先を選択してください。"
		application.diagnostic = ""
		return
	}
	if result.err != nil {
		application.status = "選択画面を完了できませんでした。"
		application.diagnostic = result.err.Error()
		return
	}
	switch result.kind {
	case pickerSourceFile, pickerSourceFolder:
		application.sourceEditor.SetText(result.path)
	case pickerOutputFolder:
		application.outputEditor.SetText(result.path)
	default:
		application.status = "選択結果の種類を認識できません。"
		application.diagnostic = fmt.Sprintf("選択画面の種類を判別できません: %d", result.kind)
		return
	}
	application.status = fmt.Sprintf("選択しました: %s", result.path)
	application.diagnostic = ""
}
