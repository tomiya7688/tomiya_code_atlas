package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//	{
//	  責務: [TestFindApplicationRootPrefersExecutableDirectory: 配布manifestのある実行directoryをrootとして解決する]
//	}
func TestFindApplicationRootPrefersExecutableDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "backends"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backends", "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	workingRoot := t.TempDir()
	actual, err := FindApplicationRoot(filepath.Join(root, "tomiya-code-atlas.exe"), workingRoot)
	if err != nil {
		t.Fatalf("FindApplicationRoot returned error: %v", err)
	}
	if actual != root {
		t.Fatalf("application root = %q, want %q", actual, root)
	}
}

//	{
//	  責務: [TestFindApplicationRootFallsBackToWorkingDirectory: go run利用時にcurrent directoryのmanifestを利用する]
//	}
func TestFindApplicationRootFallsBackToWorkingDirectory(t *testing.T) {
	workingRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workingRoot, "backends"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workingRoot, "backends", "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	actual, err := FindApplicationRoot(filepath.Join(t.TempDir(), "go-build.exe"), workingRoot)
	if err != nil {
		t.Fatalf("FindApplicationRoot returned error: %v", err)
	}
	if actual != workingRoot {
		t.Fatalf("application root = %q, want %q", actual, workingRoot)
	}
}

//	{
//	  責務: [TestWriteTextFileCreatesDirectoryAndRejectsTraversal: 出力folder作成とdirectory traversal拒否を検証する]
//	}
func TestWriteTextFileCreatesDirectoryAndRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	outputPath, err := WriteTextFile(filepath.Join(root, "nested", "output"), "result.md", "# result\n")
	if err != nil {
		t.Fatalf("WriteTextFile returned error: %v", err)
	}
	contents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("output file was not created: %v", err)
	}
	if string(contents) != "# result\n" {
		t.Fatalf("output contents = %q", contents)
	}
	if _, err := WriteTextFile(root, filepath.Join("..", "escape.md"), "unsafe"); err == nil || !strings.Contains(err.Error(), "不正") {
		t.Fatalf("directory traversal error = %v", err)
	}
}
