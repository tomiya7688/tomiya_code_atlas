//go:build !windows

package ui

import "path/filepath"

//	{
//	  責務: [webviewInstallIdentity: 実体pathへ解決済みの配置先フォルダーをprofile識別値として返す,]
//	  引数: [installDirectory: 実行ファイルが配置された実体フォルダー,]
//	  戻り値: [string: 同一配置先に安定した識別値,]
//	}
func webviewInstallIdentity(installDirectory string) string {
	return filepath.Clean(installDirectory)
}
