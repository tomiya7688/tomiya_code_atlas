//go:build windows

package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

//	{
//	  責務: [TestWebviewFinalPathIdentityResolvesWindowsCaseAliases: file IDを取得できない場合のfallbackでも同一directoryのcase aliasを同じidentityにする,]
//	}
func TestWebviewFinalPathIdentityResolvesWindowsCaseAliases(t *testing.T) {
	installDirectory := filepath.Join(t.TempDir(), "Application")
	if err := os.MkdirAll(installDirectory, 0o700); err != nil {
		t.Fatalf("create installation directory: %v", err)
	}
	openDirectory := func(directory string) windows.Handle {
		t.Helper()
		pathPointer, err := windows.UTF16PtrFromString(directory)
		if err != nil {
			t.Fatalf("convert directory path: %v", err)
		}
		handle, err := windows.CreateFile(
			pathPointer,
			windows.FILE_READ_ATTRIBUTES,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
			nil,
			windows.OPEN_EXISTING,
			windows.FILE_FLAG_BACKUP_SEMANTICS,
			0,
		)
		if err != nil {
			t.Fatalf("open installation directory: %v", err)
		}
		return handle
	}
	firstHandle := openDirectory(installDirectory)
	defer windows.CloseHandle(firstHandle)
	firstIdentity, err := webviewFinalPathIdentity(firstHandle)
	if err != nil {
		t.Fatalf("resolve first final path identity: %v", err)
	}
	caseAliasHandle := openDirectory(strings.ToUpper(installDirectory))
	defer windows.CloseHandle(caseAliasHandle)
	caseAliasIdentity, err := webviewFinalPathIdentity(caseAliasHandle)
	if err != nil {
		t.Fatalf("resolve case alias final path identity: %v", err)
	}
	if firstIdentity != caseAliasIdentity {
		t.Fatalf("final path identity changed with case: first %q, case alias %q", firstIdentity, caseAliasIdentity)
	}
}

//	{
//	  責務: [TestWebviewPathFallbackIdentityComparesDirectoryIdentity: fallbackは同じ実体だけcaseを統合し、別pathの文字列を保持する,]
//	}
func TestWebviewPathFallbackIdentityComparesDirectoryIdentity(t *testing.T) {
	installDirectory := filepath.Join(t.TempDir(), "Application")
	if err := os.MkdirAll(installDirectory, 0o700); err != nil {
		t.Fatalf("create installation directory: %v", err)
	}
	caseAliasDirectory := strings.ToUpper(installDirectory)
	firstIdentity := webviewPathFallbackIdentity(installDirectory)
	caseAliasIdentity := webviewPathFallbackIdentity(caseAliasDirectory)
	firstInformation, firstError := os.Stat(installDirectory)
	caseAliasInformation, caseAliasError := os.Stat(caseAliasDirectory)
	sameDirectory := firstError == nil && caseAliasError == nil && os.SameFile(firstInformation, caseAliasInformation)
	if sameDirectory && firstIdentity != caseAliasIdentity {
		t.Fatalf("fallback identity changed for the same directory: first %q, alias %q", firstIdentity, caseAliasIdentity)
	}
	if !sameDirectory && firstIdentity == caseAliasIdentity {
		t.Fatalf("fallback identity conflated different or unavailable paths: first %q, alias %q", firstIdentity, caseAliasIdentity)
	}
}
