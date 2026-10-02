@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0"

set "APP=.build\dist\tomiya-code-atlas.exe"
if not exist "%APP%" (
    echo [エラー] ビルド済みアプリが見つかりません: %APP%
    echo 先に build_exe.bat を実行するか、ダウンロードした tomiya-code-atlas.exe を直接起動してください。
    exit /b 1
)

"%APP%" %*
set "RESULT=%errorlevel%"
endlocal & exit /b %RESULT%
