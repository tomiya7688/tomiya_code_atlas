package data

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

//	{
//	  責務: [TestProjectFileScannerDetectsLanguages: Python版と同じextensionが安定したlanguage IDへ対応することを検証する]
//	  処理: [1: source extensionの代表例を列挙する, 2: language IDを比較する, 3: 未対応extensionがunknownになることを確認する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestProjectFileScannerDetectsLanguages(t *testing.T) {
	scanner := NewProjectFileScanner()
	testCases := []struct {
		path         string
		wantLanguage string
	}{
		{path: "main.PY", wantLanguage: "python"},
		{path: "player.gd", wantLanguage: "gdscript"},
		{path: "Game.cs", wantLanguage: "csharp"},
		{path: "engine.cpp", wantLanguage: "cpp"},
		{path: "Main.java", wantLanguage: "java"},
		{path: "main.go", wantLanguage: "go"},
		{path: "workflow.yml", wantLanguage: "yaml"},
		{path: "README.md", wantLanguage: "unknown"},
	}
	for _, testCase := range testCases {
		if got := scanner.DetectLanguage(testCase.path); got != testCase.wantLanguage {
			t.Errorf("DetectLanguage(%q) = %q, want %q", testCase.path, got, testCase.wantLanguage)
		}
	}
}

//	{
//	  責務: [TestProjectFileScannerFindsSupportedFilesAndSkipsGeneratedTrees: recursive scanが対応sourceだけを決定順で返しtooling・生成directoryを除外することを検証する]
//	  処理: [1: 対応sourceと無関係fileをrootに作る, 2: 除外directoryにもsourceを作る, 3: scan結果pathと言語IDを比較する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestProjectFileScannerFindsSupportedFilesAndSkipsGeneratedTrees(t *testing.T) {
	rootPath := t.TempDir()
	writeProjectTestFile(t, rootPath, "src/main.py")
	writeProjectTestFile(t, rootPath, "src/Game.cs")
	writeProjectTestFile(t, rootPath, "src/readme.md")
	writeProjectTestFile(t, rootPath, ".git/ignored.py")
	writeProjectTestFile(t, rootPath, ".dev/ignored.go")
	writeProjectTestFile(t, rootPath, "dist/generated.java")
	writeProjectTestFile(t, rootPath, "src/build/generated.gd")

	scanner := NewProjectFileScanner()
	gotFiles, err := scanner.DiscoverSupportedFiles(rootPath)
	if err != nil {
		t.Fatalf("DiscoverSupportedFiles() error = %v", err)
	}
	wantFiles := []ProjectSourceFile{
		{Path: filepath.Join(rootPath, "src", "Game.cs"), Language: "csharp"},
		{Path: filepath.Join(rootPath, "src", "main.py"), Language: "python"},
	}
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("DiscoverSupportedFiles() = %#v, want %#v", gotFiles, wantFiles)
	}
}

//	{
//	  責務: [TestProjectFileScannerSupportsSingleFileRoot: 単一source fileの選択でもpathと言語IDを保持することを検証する]
//	  処理: [1: source fileを作る, 2: file rootをscanする, 3: 一件だけ返ることを検証する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestProjectFileScannerSupportsSingleFileRoot(t *testing.T) {
	rootPath := t.TempDir()
	filePath := writeProjectTestFile(t, rootPath, "single.go")
	gotFiles, err := NewProjectFileScanner().DiscoverSupportedFiles(filePath)
	if err != nil {
		t.Fatalf("DiscoverSupportedFiles() error = %v", err)
	}
	wantFiles := []ProjectSourceFile{{Path: filePath, Language: "go"}}
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("DiscoverSupportedFiles() = %#v, want %#v", gotFiles, wantFiles)
	}
}

//	{
//	  責務: [TestIsDirectoryDistinguishesFilesAndDirectories: fileとdirectoryの判定および存在しないpathのerrorを検証する]
//	  処理: [1: fileとdirectoryを用意する, 2: IsDirectoryの結果を比較する, 3: 未存在pathがerrorになることを確認する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestIsDirectoryDistinguishesFilesAndDirectories(t *testing.T) {
	rootPath := t.TempDir()
	filePath := writeProjectTestFile(t, rootPath, "source.go")
	if isDirectory, err := IsDirectory(rootPath); err != nil || !isDirectory {
		t.Fatalf("IsDirectory(directory) = (%v, %v), want (true, nil)", isDirectory, err)
	}
	if isDirectory, err := IsDirectory(filePath); err != nil || isDirectory {
		t.Fatalf("IsDirectory(file) = (%v, %v), want (false, nil)", isDirectory, err)
	}
	if _, err := IsDirectory(filepath.Join(rootPath, "missing")); err == nil {
		t.Fatal("IsDirectory(missing) returned no error")
	}
}

//	{
//	  責務: [TestProjectFileScannerFindsDeploymentCandidates: deployment scanがDockerfile・Compose・YAML候補を見つけてignored treeを除くことを検証する]
//	  処理: [1: DockerfileとYAML設定を作る, 2: 除外directoryにも候補名を作る, 3: scan結果をpath順で比較する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestProjectFileScannerFindsDeploymentCandidates(t *testing.T) {
	rootPath := t.TempDir()
	writeProjectTestFile(t, rootPath, "Dockerfile")
	writeProjectTestFile(t, rootPath, "compose.yaml")
	writeProjectTestFile(t, rootPath, "k8s/deployment.yml")
	writeProjectTestFile(t, rootPath, "dist/docker-compose.yml")
	writeProjectTestFile(t, rootPath, "README.md")

	gotPaths, err := NewProjectFileScanner().DiscoverDeploymentFiles(rootPath)
	if err != nil {
		t.Fatalf("DiscoverDeploymentFiles() error = %v", err)
	}
	wantPaths := []string{
		filepath.Join(rootPath, "compose.yaml"),
		filepath.Join(rootPath, "Dockerfile"),
		filepath.Join(rootPath, "k8s", "deployment.yml"),
	}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("DiscoverDeploymentFiles() = %#v, want %#v", gotPaths, wantPaths)
	}
}

//	{
//	  責務: [TestProjectFileScannerReportsMissingRoot: 存在しないrootをscanした場合に成功扱いせずerrorにすることを検証する]
//	  処理: [1: 存在しないroot pathを作る, 2: source scanを実行する, 3: root pathを含むerrorを確認する]
//	  引数: [t: Go test runner]
//	  戻り値: []
//	}
func TestProjectFileScannerReportsMissingRoot(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "missing-project")
	if _, err := NewProjectFileScanner().DiscoverSupportedFiles(rootPath); err == nil {
		t.Fatal("DiscoverSupportedFiles() returned no error for a missing project root")
	}
}

//	{
//	  責務: [writeProjectTestFile: project scan fixtureを親directory込みで作成する]
//	  処理: [1: 親directoryを作る, 2: fixture fileを書き込む, 3: 作成したpathを返す]
//	  引数: [t: Go test runner, rootPath: fixture root, relativePath: rootからのfixture path]
//	  戻り値: [filePath: 作成したfixture path]
//	}
func writeProjectTestFile(t *testing.T, rootPath string, relativePath string) string {
	t.Helper()
	filePath := filepath.Join(rootPath, relativePath)
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create project test directory: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write project test file: %v", err)
	}
	return filePath
}
