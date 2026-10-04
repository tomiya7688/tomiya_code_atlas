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

if defined JAVA_HOME set "JAVAC=%JAVA_HOME%\bin\javac.exe"
if not defined JAVAC for %%I in (javac.exe) do set "JAVAC=%%~$PATH:I"
if not defined JAVAC (
    echo [エラー] 全parser helperのbuildにはJDK 25が必要です。JAVA_HOMEを設定してください。
    exit /b 1
)
for %%I in ("%JAVAC%") do set "JAVA_HOME=%%~dpI.."
if not exist "%JAVA_HOME%\bin\jlink.exe" (
    echo [エラー] private Java runtime作成用のjlinkがJDKにありません。
    exit /b 1
)
"%JAVAC%" -version 2>&1 | findstr /b /c:"javac 25." >nul
if errorlevel 1 (
    echo [エラー] parser helperのbuildにはJDK 25が必要です。JAVA_HOMEを確認してください。
    exit /b 1
)
where mvn >nul 2>nul
if errorlevel 1 (
    echo [エラー] Java parser helperのbuildにはApache Mavenが必要です。
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
".venv\Scripts\python.exe" -m pip install -e ".[test,parser-build,cpp-parser]"
if errorlevel 1 (
    echo [エラー] Python依存パッケージを準備できませんでした。ネットワーク接続と一時フォルダーへの書き込み権限を確認してください。
    exit /b 1
)

echo.
echo Python source testとparser helperのbuild環境が完了しました。Go版Windows配布物のbuildは build_exe.bat を実行してください。
endlocal & exit /b 0
