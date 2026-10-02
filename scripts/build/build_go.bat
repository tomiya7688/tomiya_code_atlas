@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

where go >nul 2>nul
if errorlevel 1 (
    echo [エラー] Go toolchainが見つかりません。Go 1.22以降をインストールしてください。
    echo 入手先: https://go.dev/dl/
    exit /b 1
)

if not exist ".build\dist" mkdir ".build\dist"
if not exist ".build\go-cache" mkdir ".build\go-cache"
if not exist ".build\go-temp" mkdir ".build\go-temp"
set "GOCACHE=%CD%\.build\go-cache"
set "GOTMPDIR=%CD%\.build\go-temp"
pushd go
if errorlevel 1 (
    echo [エラー] Goソースのディレクトリが見つかりません: go
    exit /b 1
)
go test ./...
if errorlevel 1 (
    popd
    exit /b 1
)

go build -trimpath -ldflags "-s -w -X main.version=0.1.0-dev" -o "..\.build\dist\tomiya-code-atlas.exe" "./cmd/tomiya-code-atlas"
set "BUILD_RESULT=%errorlevel%"
popd
if not "%BUILD_RESULT%"=="0" exit /b %BUILD_RESULT%

if not exist ".build\dist\tomiya-code-atlas.exe" (
    echo [エラー] Go版の配布EXEが生成されませんでした。
    exit /b 1
)

echo.
echo Go版のWindowsアプリをbuildしました: .build\dist\tomiya-code-atlas.exe
endlocal
exit /b 0
