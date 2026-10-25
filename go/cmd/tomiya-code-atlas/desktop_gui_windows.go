//go:build windows && !console

package main

import "github.com/tomiya7688/tomiya_code_atlas/internal/ui"

const desktopGUIEnabled = true

//	{
//	  責務: [runDesktopGUI: GoのProcessを初期化し、Windows向けGio GUIを起動する,],
//	  処理: [1: 解析サービスを作る, 2: 初期化診断とともにGUIを表示する,],
//	}
func runDesktopGUI() {
	service, err := newDiagramService()
	ui.Run(service, err)
}
