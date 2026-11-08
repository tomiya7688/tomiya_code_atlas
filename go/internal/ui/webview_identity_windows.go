//go:build windows

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsFileIDInfo struct {
	volumeSerialNumber uint64
	fileID             [16]byte
}

//	{
//	  責務: [webviewInstallIdentity: Windowsのvolume IDとfolder file IDから配置先の実体を識別する,]
//	  処理: [1: directory handleを開く, 2: volume IDと128-bit file IDを取得する, 3: 取得に失敗した場合はpathを使う,]
//	  引数: [installDirectory: 実行ファイルが配置された実体フォルダー,]
//	  戻り値: [string: 同一folderのcase aliasとsymlinkを共有し、別folderを区別する識別値,]
//	}
func webviewInstallIdentity(installDirectory string) string {
	pathPointer, pathError := windows.UTF16PtrFromString(installDirectory)
	if pathError != nil {
		return webviewPathFallbackIdentity(installDirectory)
	}
	fileHandle, handleError := windows.CreateFile(
		pathPointer,
		windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if handleError != nil {
		return webviewPathFallbackIdentity(installDirectory)
	}
	defer windows.CloseHandle(fileHandle)

	var fileInformation windowsFileIDInfo
	informationError := windows.GetFileInformationByHandleEx(
		fileHandle,
		windows.FileIdInfo,
		(*byte)(unsafe.Pointer(&fileInformation)),
		uint32(unsafe.Sizeof(fileInformation)),
	)
	if informationError != nil {
		if finalPathIdentity, finalPathError := webviewFinalPathIdentity(fileHandle); finalPathError == nil {
			return finalPathIdentity
		}
		return webviewPathFallbackIdentity(installDirectory)
	}
	return fmt.Sprintf("file:%016x:%x", fileInformation.volumeSerialNumber, fileInformation.fileID)
}

//	{
//	  責務: [webviewPathFallbackIdentity: directory handleを使えない場合に実体比較で安全なpath識別値を作る,]
//	  処理: [1: 元pathと大文字pathの実体情報を取得する, 2: 同じ実体ならcaseを正規化する, 3: 別実体または取得失敗時は元表記を保つ,]
//	  引数: [installDirectory: 実行ファイルが配置されたfolder path,]
//	  戻り値: [string: 同一実体を共有し、case-sensitiveな別folderを区別する識別値,]
//	}
func webviewPathFallbackIdentity(installDirectory string) string {
	cleanDirectory := filepath.Clean(installDirectory)
	originalInformation, originalError := os.Stat(cleanDirectory)
	if originalError != nil {
		return "path:" + cleanDirectory
	}
	caseNormalizedDirectory := strings.ToUpper(cleanDirectory)
	caseNormalizedInformation, caseNormalizedError := os.Stat(caseNormalizedDirectory)
	if caseNormalizedError == nil && os.SameFile(originalInformation, caseNormalizedInformation) {
		return "path:" + caseNormalizedDirectory
	}
	return "path:" + cleanDirectory
}

//	{
//	  責務: [webviewFinalPathIdentity: directory handleからWindowsが正規化した最終pathを取得する,]
//	  処理: [1: path bufferを用意する, 2: handleの最終pathを取得し、buffer不足なら拡張する, 3: profile識別用prefixを付けて返す,]
//	  引数: [fileHandle: 実体directoryを開いたWindows handle,]
//	  戻り値: [string: case aliasとsymlinkを解決したpath識別値, error: Windowsが最終pathを返せない場合の説明,]
//	}
func webviewFinalPathIdentity(fileHandle windows.Handle) (string, error) {
	pathBuffer := make([]uint16, 256)
	for {
		pathLength, err := windows.GetFinalPathNameByHandle(fileHandle, &pathBuffer[0], uint32(len(pathBuffer)), 0)
		if pathLength == 0 {
			return "", fmt.Errorf("実体directoryの最終pathを取得できません: %w", err)
		}
		if pathLength >= uint32(len(pathBuffer)) {
			pathBuffer = make([]uint16, pathLength+1)
			continue
		}
		return "path:" + windows.UTF16ToString(pathBuffer[:pathLength]), nil
	}
}
