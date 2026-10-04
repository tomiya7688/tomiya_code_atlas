@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

where go >nul 2>nul
if errorlevel 1 (
    echo [エラー] Go toolchainが見つかりません。Go 1.27.xをインストールしてください。
    echo 入手先: https://go.dev/dl/
    exit /b 1
)

if not exist ".build\dist" mkdir ".build\dist"
if not exist ".build\go-cache" mkdir ".build\go-cache"
if not exist ".build\go-temp" mkdir ".build\go-temp"
set "GOCACHE=%CD%\.build\go-cache"
set "GOTMPDIR=%CD%\.build\go-temp"
if not exist ".venv\Scripts\python.exe" (
    call scripts\build\setup.bat
    if errorlevel 1 exit /b 1
)
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
if not "%BUILD_RESULT%"=="0" (
    popd
    exit /b %BUILD_RESULT%
)
popd
call scripts\build\build_go_parser_backend.bat
if errorlevel 1 exit /b 1
call scripts\build\build_python_backend.bat
if errorlevel 1 exit /b 1
call scripts\build\build_csharp_backend.bat
if errorlevel 1 exit /b 1
call scripts\build\build_cpp_backend.bat
if errorlevel 1 exit /b 1
echo Building Java parser helper and bundled runtime...
call scripts\build\build_java_backend.bat
if errorlevel 1 exit /b 1
if not exist ".build\dist\backends\java\tomiya-java-backend.jar" (
    echo [エラー] JavaParser helper JARが配布先にありません。
    exit /b 1
)
if not exist ".build\dist\backends\java\runtime\bin\java.exe" (
    echo [エラー] Java private runtimeが配布先にありません。
    exit /b 1
)
if not exist ".build\dist\backends\go\tomiya-go-backend.exe" (
    echo [エラー] Go parser helperが配布先にありません。
    exit /b 1
)

if not exist ".build\dist\tomiya-code-atlas.exe" (
    echo [エラー] Go版の配布EXEが生成されませんでした。
    exit /b 1
)
echo.
echo Windows Go distribution build complete.
endlocal
exit /b 0

