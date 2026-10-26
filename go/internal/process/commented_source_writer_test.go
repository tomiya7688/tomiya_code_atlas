package process

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/data"
)

//	{
//	  責務: [TestCommentedSourceWriterPreservesInputAndWritesCommentCopy: 元sourceを保持しGo comment付きcopyを保存する]
//	}
func TestCommentedSourceWriterPreservesInputAndWritesCommentCopy(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.go")
	source := "package sample\n\nfunc Unannotated() {}\n"
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := CommentedSourceWriter{sourceRoot: sourcePath, outputDirectory: filepath.Join(root, "output")}
	outputPath, err := writer.WriteCommentedCopy(
		data.ProjectSourceFile{Path: sourcePath, Language: "go"},
		source,
		commonir.Module{Entities: []commonir.Entity{{
			Kind: "function", Name: "Unannotated", SourceLocation: commonir.SourceLocation{Line: 3},
		}}},
	)
	if err != nil {
		t.Fatalf("CommentedSourceWriter.WriteCommentedCopy returned error: %v", err)
	}
	if filepath.Base(outputPath) != "main_commented.go" {
		t.Fatalf("output path = %q, want main_commented.go", outputPath)
	}
	commentedSource, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("commented source copy was not written: %v", err)
	}
	if !strings.Contains(string(commentedSource), "// unannotatedの処理を行います。") {
		t.Fatalf("commented source = %q", commentedSource)
	}
	originalSource, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("original source could not be read: %v", err)
	}
	if string(originalSource) != source {
		t.Fatalf("original source changed: %q", originalSource)
	}
}

//	{
//	  責務: [TestCommentedSourceWriterPreservesDirectoryStructure: folder入力で元fileの相対pathを維持する]
//	}
func TestCommentedSourceWriterPreservesDirectoryStructure(t *testing.T) {
	root := t.TempDir()
	writer := CommentedSourceWriter{sourceRoot: root, outputDirectory: filepath.Join(root, "output"), sourceIsDirectory: true}
	actual, err := writer.outputRelativePath(filepath.Join(root, "pkg", "main.go"))
	if err != nil {
		t.Fatalf("outputRelativePath returned error: %v", err)
	}
	if want := filepath.Join("comments", "pkg", "main.go"); actual != want {
		t.Fatalf("output path = %q, want %q", actual, want)
	}
	if _, err := writer.outputRelativePath(filepath.Join(t.TempDir(), "outside.go")); err == nil {
		t.Fatal("outputRelativePath accepted a file outside the input project")
	}
}
