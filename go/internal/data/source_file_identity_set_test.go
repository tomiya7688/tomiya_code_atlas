package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//	{
//	  責務: [TestSourceFileIdentitySetRejectsEveryInputAlias: 他のinput sourceと同じpathまたはhardlinkをoutputとして拒否する,]
//	}
func TestSourceFileIdentitySetRejectsEveryInputAlias(t *testing.T) {
	root := t.TempDir()
	firstSource := filepath.Join(root, "main.go")
	secondSource := filepath.Join(root, "comments", "main.go")
	if err := os.MkdirAll(filepath.Dir(secondSource), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(firstSource, []byte("package first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secondSourceBytes := []byte("package second\n")
	if err := os.WriteFile(secondSource, secondSourceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	identities, err := NewSourceFileIdentitySet([]string{firstSource, secondSource})
	if err != nil {
		t.Fatalf("NewSourceFileIdentitySet returned error: %v", err)
	}

	if _, err := WriteTextFileAtRelativePathProtectingSources(root, filepath.Join("comments", "main.go"), "overwrite", identities); err == nil {
		t.Fatal("output path equal to a different input source was accepted")
	}
	hardlinkPath := filepath.Join(root, "existing-output.go")
	if err := os.Link(secondSource, hardlinkPath); err != nil {
		t.Skipf("hardlink is not available in this environment: %v", err)
	}
	if _, err := WriteTextFileAtRelativePathProtectingSources(root, filepath.Base(hardlinkPath), "overwrite", identities); err == nil {
		t.Fatal("output hardlink to a different input source was accepted")
	}
	currentSource, err := os.ReadFile(secondSource)
	if err != nil {
		t.Fatalf("second input source could not be read: %v", err)
	}
	if string(currentSource) != string(secondSourceBytes) {
		t.Fatalf("second input source changed: %q", currentSource)
	}
}

//	{
//	  責務: [TestSourceFileIdentitySetRetainsOpenedFileIdentity: source pathの差替え後も解析開始時のfile identityを保つ,]
//	}
func TestSourceFileIdentitySetRetainsOpenedFileIdentity(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.go")
	originalSourcePath := filepath.Join(root, "saved-main.go")
	originalSource := []byte("package original\n")
	if err := os.WriteFile(sourcePath, originalSource, 0o600); err != nil {
		t.Fatal(err)
	}
	identities, err := NewSourceFileIdentitySet([]string{sourcePath})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(sourcePath, originalSourcePath); err != nil {
		t.Fatal(err)
	}
	replacementSource := []byte("package replacement\n")
	if err := os.WriteFile(sourcePath, replacementSource, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := identities.ReadSourceFileMatchingIdentity(sourcePath); err == nil || !strings.Contains(err.Error(), "identity取得後に差し替え") {
		t.Fatalf("source read after path replacement error = %v, want identity-change error", err)
	}
	outputPath := filepath.Join(root, "existing-output.go")
	if err := os.Link(originalSourcePath, outputPath); err != nil {
		t.Skipf("hardlink is not available in this environment: %v", err)
	}
	if _, err := WriteTextFileAtRelativePathProtectingSources(root, filepath.Base(outputPath), "overwrite", identities); err == nil {
		t.Fatal("writer accepted an output hardlink to the file captured before its source path changed")
	}
	for path, want := range map[string][]byte{
		originalSourcePath: originalSource,
		sourcePath:         replacementSource,
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
//	  責務: [TestNormalizeFilePathPreservesCase: path正規化で大小文字を変えない,]
//	}
func TestNormalizeFilePathPreservesCase(t *testing.T) {
	upperCasePath := filepath.Join("project", "Foo.go")
	lowerCasePath := filepath.Join("project", "foo.go")
	if normalizeFilePath(upperCasePath) == normalizeFilePath(lowerCasePath) {
		t.Fatalf("case-distinct paths were treated as equal: %q and %q", upperCasePath, lowerCasePath)
	}
	cleanPath := filepath.Join("project", "nested", "..", "Foo.go")
	if normalizeFilePath(upperCasePath) != normalizeFilePath(cleanPath) {
		t.Fatalf("equivalent cleaned paths were treated as different: %q and %q", upperCasePath, cleanPath)
	}
}

//	{
//	  責務: [TestSourceFileIdentitySetAllowsSafeOutputs: 新規outputと別fileの既存outputを許可する,]
//	}
func TestSourceFileIdentitySetAllowsSafeOutputs(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.go")
	if err := os.WriteFile(sourcePath, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	identities, err := NewSourceFileIdentitySet([]string{sourcePath})
	if err != nil {
		t.Fatalf("NewSourceFileIdentitySet returned error: %v", err)
	}
	newOutputPath := filepath.Join(root, "new-output.go")
	if _, err := WriteTextFileAtRelativePathProtectingSources(root, filepath.Base(newOutputPath), "new contents", identities); err != nil {
		t.Fatalf("new output was rejected: %v", err)
	}
	separateOutputPath := filepath.Join(root, "separate-output.go")
	if err := os.WriteFile(separateOutputPath, []byte("old output\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	beforeUpdate, err := os.Stat(separateOutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteTextFileAtRelativePathProtectingSources(root, filepath.Base(separateOutputPath), "updated output", identities); err != nil {
		t.Fatalf("separate existing output was rejected: %v", err)
	}
	updatedOutput, err := os.ReadFile(separateOutputPath)
	if err != nil || string(updatedOutput) != "updated output" {
		t.Fatalf("separate output contents = %q, error = %v", updatedOutput, err)
	}
	afterUpdate, err := os.Stat(separateOutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if afterUpdate.Mode().Perm() != beforeUpdate.Mode().Perm() {
		t.Fatalf("updated output mode = %v, want retained mode %v", afterUpdate.Mode().Perm(), beforeUpdate.Mode().Perm())
	}
}

//	{
//	  責務: [TestProtectedTextWriterRejectsMissingSourceIdentities: 入力source identityがない場合にoutput作成を始めない,]
//	}
func TestProtectedTextWriterRejectsMissingSourceIdentities(t *testing.T) {
	root := t.TempDir()
	outputPath := filepath.Join(root, "output.go")
	if _, err := WriteTextFileAtRelativePathProtectingSources(root, filepath.Base(outputPath), "contents", SourceFileIdentitySet{}); err == nil {
		t.Fatal("protected text writer accepted an empty source identity set")
	}
	if _, err := os.Lstat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("output path state = %v, want no file", err)
	}
}

//	{
//	  責務: [TestProtectedTextWriterRejectsReadOnlyOutput: readonly既存outputを変えず一時fileも残さない,]
//	}
func TestProtectedTextWriterRejectsReadOnlyOutput(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.go")
	if err := os.WriteFile(sourcePath, []byte("package source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	identities, err := NewSourceFileIdentitySet([]string{sourcePath})
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(root, "output.go")
	originalOutput := []byte("old output\n")
	if err := os.WriteFile(outputPath, originalOutput, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(outputPath, 0o444); err != nil {
		t.Skipf("read-only permissions are not available in this environment: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(outputPath, 0o600) })
	if _, err := WriteTextFileAtRelativePathProtectingSources(root, filepath.Base(outputPath), "new output", identities); err == nil || !strings.Contains(err.Error(), "書込み不可") {
		t.Fatalf("read-only output error = %v, want write-protection error", err)
	}
	currentOutput, err := os.ReadFile(outputPath)
	if err != nil || string(currentOutput) != string(originalOutput) {
		t.Fatalf("read-only output contents = %q, error = %v", currentOutput, err)
	}
	temporaryFiles, err := filepath.Glob(filepath.Join(root, ".tca-output-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary output files remain after rejected write: %v", temporaryFiles)
	}
}

//	{
//	  責務: [TestSourceFileIdentitySetRejectsDanglingSymlink: dangling symlinkを新規outputと誤認せず拒否する,]
//	}
func TestSourceFileIdentitySetRejectsDanglingSymlink(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.go")
	if err := os.WriteFile(sourcePath, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	identities, err := NewSourceFileIdentitySet([]string{sourcePath})
	if err != nil {
		t.Fatalf("NewSourceFileIdentitySet returned error: %v", err)
	}
	symlinkPath := filepath.Join(root, "dangling-output.go")
	if err := os.Symlink(filepath.Join(root, "missing.go"), symlinkPath); err != nil {
		t.Skipf("symlink is not available in this environment: %v", err)
	}
	if _, err := WriteTextFileAtRelativePathProtectingSources(root, filepath.Base(symlinkPath), "overwrite", identities); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("dangling symlink validation error = %v, want regular-file error", err)
	}
}
