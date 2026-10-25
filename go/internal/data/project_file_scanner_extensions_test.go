package data

import (
	"sort"
	"testing"
)

//	{
//	  責務: [TestSupportedFileExtensionsAreSortedAndRecognized: ファイル ダイアログ向け拡張子一覧が探索器契約と一致することを検証する,],
//	  処理: [1: 対応拡張子一覧を取得する, 2: 昇順と探索器の言語判定を確認する,],
//	}
func TestSupportedFileExtensionsAreSortedAndRecognized(t *testing.T) {
	scanner := NewProjectFileScanner()
	extensions := scanner.SupportedFileExtensions()
	if len(extensions) == 0 {
		t.Fatal("SupportedFileExtensions returned an empty list")
	}
	if !sort.StringsAreSorted(extensions) {
		t.Fatalf("supported extensions are not sorted: %#v", extensions)
	}
	for _, extension := range extensions {
		if scanner.DetectLanguage("source"+extension) == "unknown" {
			t.Errorf("supported extension %q is not recognized by DetectLanguage", extension)
		}
	}
}
