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
//	  責務: [TestCommentedSourceWriterPreservesInputAndWritesCommentCopy: 元sourceを保持しGo comment付きcopyを保存する,]
//	}
func TestCommentedSourceWriterPreservesInputAndWritesCommentCopy(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.go")
	source := "package sample\n\nfunc Unannotated() {}\n"
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	identities, err := data.NewSourceFileIdentitySet([]string{sourcePath})
	if err != nil {
		t.Fatal(err)
	}
	writer := CommentedSourceWriter{
		sourceRoot:           sourcePath,
		outputDirectory:      filepath.Join(root, "output"),
		sourceFileIdentities: identities,
	}
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
//	  責務: [TestCommentedSourceWriterRejectsSourceOverwriteAndPreservesInput: outputがsource fileと同じ場合に書込みを拒否して元のbyteを保つ,]
//	}
func TestCommentedSourceWriterRejectsSourceOverwriteAndPreservesInput(t *testing.T) {
	parent := t.TempDir()
	sourceRoot := filepath.Join(parent, "comments")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(sourceRoot, "main.go")
	originalSource := []byte("package sample\n\nfunc Unannotated() {}\n")
	if err := os.WriteFile(sourcePath, originalSource, 0o600); err != nil {
		t.Fatal(err)
	}

	identities, err := data.NewSourceFileIdentitySet([]string{sourcePath})
	if err != nil {
		t.Fatal(err)
	}
	writer := CommentedSourceWriter{
		sourceRoot:           sourceRoot,
		outputDirectory:      parent,
		sourceIsDirectory:    true,
		sourceFileIdentities: identities,
	}
	_, err = writer.WriteCommentedCopy(
		data.ProjectSourceFile{Path: sourcePath, Language: "go"},
		string(originalSource),
		commonir.Module{Entities: []commonir.Entity{{
			Kind: "function", Name: "Unannotated", SourceLocation: commonir.SourceLocation{Line: 3},
		}}},
	)
	if err == nil {
		t.Fatal("CommentedSourceWriter.WriteCommentedCopy overwrote its source file")
	}
	if !strings.Contains(err.Error(), "出力fileが解析対象source fileと同じです") {
		t.Fatalf("error = %q, want source/output collision error", err)
	}

	currentSource, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("original source could not be read after rejected write: %v", err)
	}
	if string(currentSource) != string(originalSource) {
		t.Fatalf("original source changed after rejected write: %q", currentSource)
	}
}

//	{
//	  責務: [TestCommentedSourceWriterRejectsOutputThatAliasesAnotherInput: directoryのcomment outputが別のinput sourceと同じ場合に両sourceを保護する,]
//	}
func TestCommentedSourceWriterRejectsOutputThatAliasesAnotherInput(t *testing.T) {
	root := t.TempDir()
	firstSourcePath := filepath.Join(root, "main.go")
	secondSourcePath := filepath.Join(root, "comments", "main.go")
	if err := os.MkdirAll(filepath.Dir(secondSourcePath), 0o755); err != nil {
		t.Fatal(err)
	}
	firstSource := []byte("package first\n\nfunc First() {}\n")
	secondSource := []byte("package second\n\nfunc Second() {}\n")
	if err := os.WriteFile(firstSourcePath, firstSource, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondSourcePath, secondSource, 0o600); err != nil {
		t.Fatal(err)
	}
	identities, err := data.NewSourceFileIdentitySet([]string{firstSourcePath, secondSourcePath})
	if err != nil {
		t.Fatal(err)
	}
	writer := CommentedSourceWriter{
		sourceRoot:           root,
		outputDirectory:      root,
		sourceIsDirectory:    true,
		sourceFileIdentities: identities,
	}
	_, err = writer.WriteCommentedCopy(
		data.ProjectSourceFile{Path: firstSourcePath, Language: "go"},
		string(firstSource),
		commonir.Module{Entities: []commonir.Entity{{
			Kind: "function", Name: "First", SourceLocation: commonir.SourceLocation{Line: 3},
		}}},
	)
	if err == nil {
		t.Fatal("CommentedSourceWriter.WriteCommentedCopy overwrote a different input source")
	}
	if !strings.Contains(err.Error(), "出力fileが解析対象source fileと同じです") {
		t.Fatalf("error = %q, want input/output collision error", err)
	}
	for path, want := range map[string][]byte{
		firstSourcePath:  firstSource,
		secondSourcePath: secondSource,
	} {
		actual, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("source could not be read after rejected write (%s): %v", path, err)
		}
		if string(actual) != string(want) {
			t.Errorf("source changed after rejected write (%s): %q", path, actual)
		}
	}
}

//	{
//	  責務: [TestCommentedSourceWriterRejectsSingleFileHardlinkToSource: 単一fileのcomment outputが元sourceのhardlinkなら書込みを拒否する,]
//	}
func TestCommentedSourceWriterRejectsSingleFileHardlinkToSource(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.go")
	originalSource := []byte("package sample\n\nfunc Unannotated() {}\n")
	if err := os.WriteFile(sourcePath, originalSource, 0o600); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(root, "comments", "main_commented.go")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(sourcePath, outputPath); err != nil {
		t.Skipf("hardlink is not available in this environment: %v", err)
	}

	identities, err := data.NewSourceFileIdentitySet([]string{sourcePath})
	if err != nil {
		t.Fatal(err)
	}
	writer := CommentedSourceWriter{
		sourceRoot:           sourcePath,
		outputDirectory:      root,
		sourceFileIdentities: identities,
	}
	_, err = writer.WriteCommentedCopy(
		data.ProjectSourceFile{Path: sourcePath, Language: "go"},
		string(originalSource),
		commonir.Module{Entities: []commonir.Entity{{
			Kind: "function", Name: "Unannotated", SourceLocation: commonir.SourceLocation{Line: 3},
		}}},
	)
	if err == nil {
		t.Fatal("CommentedSourceWriter.WriteCommentedCopy wrote through a hardlink to its source")
	}
	if !strings.Contains(err.Error(), "出力fileが解析対象source fileと同じです") {
		t.Fatalf("error = %q, want source/output collision error", err)
	}

	currentSource, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("original source could not be read after rejected hardlink write: %v", err)
	}
	if string(currentSource) != string(originalSource) {
		t.Fatalf("original source changed after rejected hardlink write: %q", currentSource)
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
