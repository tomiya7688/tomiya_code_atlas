@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

if "%~1"=="" (
    echo 使用方法: scripts\build\format_go.bat go\path\to\file.go [追加のGo file...]
    echo 変更したGo fileだけをgofmtで整形します。
    exit /b 2
)

where go >nul 2>nul
if errorlevel 1 (
    echo [エラー] Go 1.27.xが見つかりません。gofmtはGo toolchainに含まれます。
    exit /b 1
)

for /f "tokens=3" %%V in ('go version') do set "GO_VERSION=%%V"
if not "%GO_VERSION:~0,6%"=="go1.27" (
    echo [エラー] Go 1.27.xに含まれるgofmtを使用してください。現在: %GO_VERSION%
    exit /b 1
)

gofmt -w %*
if errorlevel 1 exit /b %errorlevel%

set "UNFORMATTED="
for /f "delims=" %%F in ('gofmt -l %*') do (
    echo [エラー] 整形確認に失敗しました: %%F
    set "UNFORMATTED=1"
)
if defined UNFORMATTED exit /b 1

echo gofmt %GO_VERSION%で指定fileを整形・確認しました。
endlocal
exit /b 0
