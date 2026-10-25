@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

if "%~1"=="" (
    echo 使用方法: scripts\build\format_python.bat path\to\changed_file.py [追加のPython file...]
    echo 指定したPython fileだけをRuffで整形します。
    exit /b 2
)

set "RUFF=%CD%\.venv\Scripts\ruff.exe"
if not exist "%RUFF%" (
    echo [エラー] Ruffが見つかりません。先に scripts\build\setup.bat を実行してください。
    exit /b 1
)

for /f "tokens=2" %%V in ('"%RUFF%" --version') do set "RUFF_VERSION=%%V"
if not "%RUFF_VERSION%"=="0.15.22" (
    echo [エラー] Ruff 0.15.22を使用してください。現在: %RUFF_VERSION%
    exit /b 1
)

"%RUFF%" format --no-cache %*
if errorlevel 1 exit /b %errorlevel%
"%RUFF%" format --check --no-cache %*
if errorlevel 1 exit /b %errorlevel%

echo Ruff %RUFF_VERSION%で指定fileを整形・確認しました。
endlocal
exit /b 0
