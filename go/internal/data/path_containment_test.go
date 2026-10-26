package data

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

//	{
//	  責務: [TestIsPathWithinOrEqualChecksPathBoundaries: 同じfolder・配下・兄弟・親folderの包含判定を検証する,]
//	  処理: [1: project rootとcandidate pathを用意する, 2: candidateごとの包含結果を比較する,]
//	  引数: [t: Go test runner,]
//	  戻り値: []
//	}
func TestIsPathWithinOrEqualChecksPathBoundaries(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		t.Fatalf("create root path: %v", err)
	}
	testCases := []struct {
		name              string
		candidatePath     string
		wantWithinOrEqual bool
	}{
		{name: "same folder", candidatePath: rootPath, wantWithinOrEqual: true},
		{name: "nested existing folder", candidatePath: filepath.Join(rootPath, "src"), wantWithinOrEqual: true},
		{name: "nested missing folder", candidatePath: filepath.Join(rootPath, "new", "output"), wantWithinOrEqual: true},
		{name: "normalized nested path", candidatePath: filepath.Join(rootPath, "src", "..", "output"), wantWithinOrEqual: true},
		{name: "sibling folder with shared prefix", candidatePath: rootPath + "-output", wantWithinOrEqual: false},
		{name: "parent sibling starting with two dots", candidatePath: filepath.Join(filepath.Dir(rootPath), "..foo"), wantWithinOrEqual: false},
		{name: "parent folder", candidatePath: filepath.Dir(rootPath), wantWithinOrEqual: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := IsPathWithinOrEqual(rootPath, testCase.candidatePath)
			if err != nil {
				t.Fatalf("IsPathWithinOrEqual() error = %v", err)
			}
			if got != testCase.wantWithinOrEqual {
				t.Fatalf("IsPathWithinOrEqual() = %v, want %v", got, testCase.wantWithinOrEqual)
			}
		})
	}
}

//	{
//	  責務: [TestIsPathWithinOrEqualResolvesRelativePaths: current working directoryからの相対pathを正規化して比較する,]
//	  処理: [1: working directoryとtemporary projectの相対pathを作る, 2: 相対candidateがproject配下と判定されることを確認する,]
//	  引数: [t: Go test runner,]
//	  戻り値: []
//	}
func TestIsPathWithinOrEqualResolvesRelativePaths(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		t.Fatalf("create root path: %v", err)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if !strings.EqualFold(filepath.VolumeName(workingDirectory), filepath.VolumeName(rootPath)) {
		t.Skip("working directory and temporary folder are on different volumes")
	}
	relativeRoot, err := filepath.Rel(workingDirectory, rootPath)
	if err != nil {
		t.Fatalf("make root path relative: %v", err)
	}
	got, err := IsPathWithinOrEqual(relativeRoot, filepath.Join(relativeRoot, "new-output"))
	if err != nil {
		t.Fatalf("IsPathWithinOrEqual() error = %v", err)
	}
	if !got {
		t.Fatal("IsPathWithinOrEqual() = false, want true for relative paths")
	}
}

//	{
//	  責務: [TestIsPathWithinOrEqualTreatsDifferentVolumesAsOutside: Windowsの別volume pathを包含外として扱う,]
//	  処理: [1: 別volume rootを探す, 2: 異なるvolume間で包含falseとerrorなしを確認する,]
//	  引数: [t: Go test runner,]
//	  戻り値: []
//	}
func TestIsPathWithinOrEqualTreatsDifferentVolumesAsOutside(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows volume names are only available on Windows")
	}
	rootPath := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		t.Fatalf("create root path: %v", err)
	}
	rootVolume := filepath.VolumeName(rootPath)
	for letter := 'A'; letter <= 'Z'; letter++ {
		candidateVolume := string(letter) + ":"
		if strings.EqualFold(rootVolume, candidateVolume) {
			continue
		}
		candidateRoot := candidateVolume + string(filepath.Separator)
		if _, err := os.Stat(candidateRoot); err != nil {
			continue
		}
		got, err := IsPathWithinOrEqual(rootPath, filepath.Join(candidateRoot, "atlas-output"))
		if err != nil {
			t.Fatalf("IsPathWithinOrEqual() error = %v", err)
		}
		if got {
			t.Fatalf("IsPathWithinOrEqual() = true for different volumes %q and %q", rootVolume, candidateVolume)
		}
		return
	}
	t.Skip("no second accessible volume is available")
}

//	{
//	  責務: [TestIsPathWithinOrEqualResolvesWindowsCaseAndSeparatorVariants: Windowsのcase違いとslash区切りを正規化して包含を判定する,]
//	  処理: [1: project rootを作る, 2: 大文字小文字とseparatorが異なるcandidateを作る, 3: 同じproject配下と判定されることを確認する,]
//	  引数: [t: Go test runner,]
//	  戻り値: []
//	}
func TestIsPathWithinOrEqualResolvesWindowsCaseAndSeparatorVariants(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows path casing and slash rules are only available on Windows")
	}
	rootPath := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		t.Fatalf("create root path: %v", err)
	}
	candidatePath := strings.ToLower(filepath.Join(rootPath, "new-output"))
	candidatePath = strings.ReplaceAll(candidatePath, "\\", "/")
	got, err := IsPathWithinOrEqual(rootPath, candidatePath)
	if err != nil {
		t.Fatalf("IsPathWithinOrEqual() error = %v", err)
	}
	if !got {
		t.Fatal("IsPathWithinOrEqual() = false, want true for Windows case and separator variants")
	}
}

//	{
//	  責務: [TestIsPathWithinOrEqualResolvesDirectorySymlink: symlink aliasでも実体folderの包含関係を検出する,]
//	  処理: [1: project rootへのsymlinkを作る, 2: alias配下のcandidateがroot内と判定されることを確認する,]
//	  引数: [t: Go test runner,]
//	  戻り値: []
//	}
func TestIsPathWithinOrEqualResolvesDirectorySymlink(t *testing.T) {
	parentPath := t.TempDir()
	rootPath := filepath.Join(parentPath, "project")
	symlinkPath := filepath.Join(parentPath, "project-alias")
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		t.Fatalf("create root path: %v", err)
	}
	if err := os.Symlink(rootPath, symlinkPath); err != nil {
		t.Skipf("directory symlink is unavailable: %v", err)
	}
	got, err := IsPathWithinOrEqual(rootPath, filepath.Join(symlinkPath, "new-output"))
	if err != nil {
		t.Fatalf("IsPathWithinOrEqual() error = %v", err)
	}
	if !got {
		t.Fatal("IsPathWithinOrEqual() = false, want true for a symlink alias")
	}
	got, err = IsPathWithinOrEqual(symlinkPath, filepath.Join(rootPath, "new-output"))
	if err != nil {
		t.Fatalf("IsPathWithinOrEqual() with symlink root error = %v", err)
	}
	if !got {
		t.Fatal("IsPathWithinOrEqual() = false, want true when the root is a symlink")
	}
}

//	{
//	  責務: [TestIsPathWithinOrEqualRejectsDanglingSymlink: 解決できないdangling symlinkを安全側で拒否する,]
//	  処理: [1: targetが存在しないsymlinkを作る, 2: path解決がerrorになることを確認する,]
//	  引数: [t: Go test runner,]
//	  戻り値: []
//	}
func TestIsPathWithinOrEqualRejectsDanglingSymlink(t *testing.T) {
	parentPath := t.TempDir()
	rootPath := filepath.Join(parentPath, "project")
	danglingPath := filepath.Join(parentPath, "output-alias")
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		t.Fatalf("create root path: %v", err)
	}
	if err := os.Symlink(filepath.Join(parentPath, "missing-output"), danglingPath); err != nil {
		t.Skipf("dangling symlink is unavailable: %v", err)
	}
	_, err := IsPathWithinOrEqual(rootPath, danglingPath)
	if err == nil || !strings.Contains(err.Error(), "dangling symlink") {
		t.Fatalf("IsPathWithinOrEqual() error = %v, want dangling symlink error", err)
	}
}
