@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

where go >nul 2>nul
if errorlevel 1 (
    echo [エラー] Go parser helperのbuildにGo toolchainが必要です。
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
echo Go parser helper build complete.
endlocal
exit /b 0

