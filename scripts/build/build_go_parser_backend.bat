@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

where go >nul 2>nul
if errorlevel 1 (
    echo [エラー] Go parser helperのbuildにはGo 1.27.xが必要です。
    exit /b 1
)
for /f "tokens=3" %%V in ('go version') do set "GO_VERSION=%%V"
if not "%GO_VERSION:~0,6%"=="go1.27" (
    echo [エラー] Go parser helperはGo 1.27.xでbuildしてください。検出したversion: %GO_VERSION%
    exit /b 1
)

if not exist ".build\go-cache" mkdir ".build\go-cache"
if not exist ".build\go-temp" mkdir ".build\go-temp"
if not exist ".build\dist\backends\go" mkdir ".build\dist\backends\go"
set "GOCACHE=%CD%\.build\go-cache"
set "GOTMPDIR=%CD%\.build\go-temp"

pushd backends\go
if errorlevel 1 exit /b 1
go test ./...
if errorlevel 1 (
    popd
    exit /b 1
)
go build -trimpath -ldflags "-s -w" -o "..\..\.build\dist\backends\go\tomiya-go-backend.exe" .
set "BUILD_RESULT=%errorlevel%"
popd
if not "%BUILD_RESULT%"=="0" exit /b %BUILD_RESULT%

if not exist ".build\dist\backends\go\tomiya-go-backend.exe" (
    echo [エラー] Go parser helperが生成されませんでした。
    exit /b 1
)
copy /y "backends\go\NOTICE.txt" ".build\dist\backends\go\NOTICE.txt" >nul
if errorlevel 1 (
    echo [エラー] Go parserのlicense noticeを配置できませんでした。
    exit /b 1
)
if not exist ".build\dist\backends\go\NOTICE.txt" (
    echo [エラー] Go parserのlicense noticeが見つかりません。
    exit /b 1
)
echo Go parser helper build complete.
endlocal
exit /b 0

