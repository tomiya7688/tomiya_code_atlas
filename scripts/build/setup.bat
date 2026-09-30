@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0\..\.."

if not exist ".build\temp" mkdir ".build\temp"
set "TEMP=%CD%\.build\temp"
set "TMP=%TEMP%"

where py >nul 2>nul
if not errorlevel 1 (
    py -3.12 -c "import sys; raise SystemExit(0 if sys.version_info[:2] == (3, 12) else 1)" >nul 2>nul
    if not errorlevel 1 set "PYTHON=py -3.12"
)

if not defined PYTHON (
    where python >nul 2>nul
    if errorlevel 1 (
        echo [エラー] Python 3.11以降が見つかりません。PythonをインストールしてPATHに追加してください。
        echo 入手先: https://www.python.org/downloads/windows/
        exit /b 1
    )
    python -c "import sys; raise SystemExit(0 if sys.version_info >= (3, 11) else 1)" >nul 2>nul
    if errorlevel 1 (
        echo [エラー] Python 3.11以降が必要です。Python 3.12または3.11以降をインストールしてください。
        echo 入手先: https://www.python.org/downloads/windows/
        exit /b 1
    )
    set "PYTHON=python"
)

%PYTHON% -c "import sys; raise SystemExit(0 if sys.version_info >= (3, 11) else 1)" >nul 2>nul
if errorlevel 1 (
    echo [エラー] Python 3.11以降を起動できません。Pythonのインストールを確認してください。
    exit /b 1
)

if not exist ".venv\Scripts\python.exe" (
    %PYTHON% -m venv .venv
    if errorlevel 1 (
        echo [エラー] Python仮想環境を作成できませんでした。Pythonのインストールと書き込み権限を確認してください。
        exit /b 1
    )
)

".venv\Scripts\python.exe" -c "import sys; raise SystemExit(0 if sys.version_info >= (3, 11) else 1)" >nul 2>nul
if errorlevel 1 (
    echo [エラー] 既存の .venv はPython 3.11以降ではありません。.venv を削除して再実行してください。
    exit /b 1
)

".venv\Scripts\python.exe" -m pip --version >nul 2>nul
if errorlevel 1 (
    echo pipが見つからないため、Python標準のensurepipで修復しています。
    ".venv\Scripts\python.exe" -m ensurepip --upgrade --default-pip
    if errorlevel 1 (
        echo [エラー] 仮想環境を修復できませんでした。.venvを削除して再実行してください。
        exit /b 1
    )
)

if not exist ".build\metadata" mkdir ".build\metadata"
if errorlevel 1 exit /b 1

".venv\Scripts\python.exe" -m pip install --upgrade pip
if errorlevel 1 (
    echo [エラー] pipを更新できませんでした。ネットワーク接続と一時フォルダーへの書き込み権限を確認してください。
    exit /b 1
)
".venv\Scripts\python.exe" -m pip install -e ".[test,exe]" build
if errorlevel 1 (
    echo [エラー] Python依存パッケージを準備できませんでした。ネットワーク接続と一時フォルダーへの書き込み権限を確認してください。
    exit /b 1
)

echo.
echo Python環境の準備が完了しました。ソース起動は scripts\build\run_source.bat、Windowsアプリのbuildは build_exe.bat を実行してください。
endlocal & exit /b 0
