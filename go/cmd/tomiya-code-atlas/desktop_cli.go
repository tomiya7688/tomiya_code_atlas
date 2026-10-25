//go:build !windows || console

package main

const desktopGUIEnabled = false

//	{
//	  責務: [runDesktopGUI: GUIを含まないCLI版で誤ってGUIの起動が要求された場合は何もしない,],
//	}
func runDesktopGUI() {}
