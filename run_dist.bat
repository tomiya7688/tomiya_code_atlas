@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0"

if "%~1"=="" goto launch_gui
if /i "%~1"=="gui" goto launch_gui
set "APP=.build\dist\tomiya-code-atlas-cli.exe"
goto launch_app

:launch_gui
set "APP=.build\dist\tomiya-code-atlas.exe"

:launch_app
if not exist "%APP%" (
    echo [エラー] ビルド済みアプリが見つかりません: %APP%
    echo 先に build_exe.bat を実行するか、GitHub ActionsからWindows配布artifactを取得してください。
    exit /b 1
)

"%APP%" %*
set "RESULT=%errorlevel%"
endlocal & exit /b %RESULT%
