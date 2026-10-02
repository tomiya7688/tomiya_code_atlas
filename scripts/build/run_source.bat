@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0\..\.."

if not exist ".venv\Scripts\python.exe" (
    echo [エラー] Python環境が見つかりません。先に scripts\build\setup.bat を実行してください。
    exit /b 1
)

".venv\Scripts\python.exe" app.py %*
set "RESULT=%errorlevel%"
endlocal & exit /b %RESULT%
