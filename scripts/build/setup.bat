@echo off
goto :main

:try_python_version
py -%1 tools\python_tls_floor.py >nul 2>nul
if errorlevel 1 exit /b 0
set "PYTHON=py -%1"
exit /b 0

:main
chcp 65001 >nul
setlocal
set "PYTHON="
cd /d "%~dp0\..\.."

if not exist ".build\temp" mkdir ".build\temp"
set "TEMP=%CD%\.build\temp"
set "TMP=%TEMP%"

where py >nul 2>nul
if not errorlevel 1 (
    call :try_python_version 3.15
    if not defined PYTHON call :try_python_version 3.14
    if not defined PYTHON call :try_python_version 3.13
    if not defined PYTHON call :try_python_version 3.12
    if not defined PYTHON call :try_python_version 3.11
)

if not defined PYTHON (
    where python >nul 2>nul
    if errorlevel 1 (
        echo [エラー] 対応する安定版CPythonが見つかりません。3.11.18、3.12.16、3.13.18、3.14.10、3.15.2以降をインストールしてください。
        echo 入手先: https://www.python.org/downloads/windows/
        exit /b 1
    )
    python -c "import sys; raise SystemExit(0 if sys.version_info >= (3, 11) else 1)" >nul 2>nul
    if errorlevel 1 (
        echo [エラー] 対応する安定版CPythonが必要です。3.11.18、3.12.16、3.13.18、3.14.10、3.15.2以降をインストールしてください。
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

%PYTHON% tools\python_tls_floor.py
if errorlevel 1 (
    echo [エラー] 安全なpip更新の前に、必要なCPython安定版patch levelを満たすPythonへ切り替えてください。
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

".venv\Scripts\python.exe" tools\python_tls_floor.py
if errorlevel 1 (
    echo [エラー] 既存の .venv は安全なpip更新に必要なCPython安定版patch levelを満たしません。.venv を対応Pythonで作り直してください。
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

".venv\Scripts\python.exe" -m pip install --upgrade --require-hashes -r tools\requirements-pip-bootstrap.txt
if errorlevel 1 (
    echo [エラー] SHA-256固定のpipを導入できませんでした。ネットワーク接続と一時フォルダーへの書き込み権限を確認してください。
    exit /b 1
)
".venv\Scripts\python.exe" -m pip install --upgrade --require-hashes -r tools\requirements-python-tls.txt
if errorlevel 1 (
    echo [エラー] 修正版certifiを導入できませんでした。ネットワーク接続と一時フォルダーへの書き込み権限を確認してください。
    exit /b 1
)
".venv\Scripts\python.exe" tools\verify_python_tls.py
if errorlevel 1 (
    echo [エラー] Python、package version、またはCA trust storeがTLS security floorを満たしていません。
    echo [確認] Python 3.11.18、3.12.16、3.13.18、3.14.10、3.15.2以降の安定版を使用してください。
    echo [確認] Windows Updateでシステムのroot certificateを更新してから再実行してください。
    exit /b 1
)
".venv\Scripts\python.exe" -m pip install -e ".[test,parser-build,cpp-parser,format]"
if errorlevel 1 (
    echo [エラー] Python依存パッケージを準備できませんでした。ネットワーク接続と一時フォルダーへの書き込み権限を確認してください。
    exit /b 1
)

echo.
echo Python source testとparser helperのbuild環境が完了しました。Go版Windows配布物のbuildは build_exe.bat を実行してください。
endlocal & exit /b 0

