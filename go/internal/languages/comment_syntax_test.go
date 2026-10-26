package languages

import (
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/models"
)

//	{
//	  責務: [TestCommentSyntaxForSupportedLanguages: 対象言語ごとのcomment prefixを検証する]
//	}
func TestCommentSyntaxForSupportedLanguages(t *testing.T) {
	testCases := []struct {
		languageID string
		wantPrefix string
	}{
		{languageID: "python", wantPrefix: "#"},
		{languageID: "gdscript", wantPrefix: "#"},
		{languageID: "csharp", wantPrefix: "//"},
		{languageID: "cpp", wantPrefix: "//"},
		{languageID: "go", wantPrefix: "//"},
		{languageID: "java", wantPrefix: "//"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.languageID, func(t *testing.T) {
			syntax, err := CommentSyntaxFor(testCase.languageID)
			if err != nil {
				t.Fatalf("CommentSyntaxFor returned error: %v", err)
			}
			output, err := syntax.Insert("func run() {}\n", []models.CommentCandidate{{
				Name: "run", Line: 1, Text: "処理を実行します。",
			}})
			if err != nil {
				t.Fatalf("Insert returned error: %v", err)
			}
			if !strings.HasPrefix(output, testCase.wantPrefix+" 処理を実行します。\n") {
				t.Fatalf("output = %q, want prefix %q", output, testCase.wantPrefix)
			}
		})
	}
}

//	{
//	  責務: [TestCommentSyntaxInsertPreservesIndentAndLineEndings: sourceを変更せずindentとCRLFを保って候補を挿入する]
//	}
func TestCommentSyntaxInsertPreservesIndentAndLineEndings(t *testing.T) {
	source := "package sample\r\n\r\nfunc run() {}\r\n"
	syntax, err := CommentSyntaxFor("go")
	if err != nil {
		t.Fatal(err)
	}
	output, err := syntax.Insert(source, []models.CommentCandidate{{
		Name: "run", Line: 3, Indent: "\t", Text: "処理を実行します。",
	}})
	if err != nil {
		t.Fatalf("Insert returned error: %v", err)
	}
	want := "package sample\r\n\r\n\t// 処理を実行します。\r\nfunc run() {}\r\n"
	if output != want {
		t.Fatalf("output = %q, want %q", output, want)
	}
	if source != "package sample\r\n\r\nfunc run() {}\r\n" {
		t.Fatal("Insert changed the input source")
	}
}

//	{
//	  責務: [TestCommentSyntaxRejectsInvalidCandidate: 範囲外行と複数行commentを拒否する]
//	}
func TestCommentSyntaxRejectsInvalidCandidate(t *testing.T) {
	syntax, err := CommentSyntaxFor("python")
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []models.CommentCandidate{
		{Name: "outside", Line: 2, Text: "説明します。"},
		{Name: "multiline", Line: 1, Text: "一行目\n二行目"},
	} {
		if _, err := syntax.Insert("def run():\n", []models.CommentCandidate{candidate}); err == nil {
			t.Errorf("Insert accepted invalid candidate %#v", candidate)
		}
	}
	if _, err := CommentSyntaxFor("yaml"); err == nil {
		t.Fatal("CommentSyntaxFor accepted unsupported yaml comments")
	}
}
