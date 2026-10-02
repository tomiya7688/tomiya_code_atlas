@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

if not exist ".build\temp" mkdir ".build\temp"
set "TEMP=%CD%\.build\temp"
set "TMP=%TEMP%"

if not exist ".venv\Scripts\python.exe" (
  echo [エラー] Python環境がありません。scripts\build\setup.batを実行してください。
  exit /b 1
)
set "PYTHON=.venv\Scripts\python.exe"
set "VERIFY_DIR=%TEMP%\tomiya-code-atlas-verify-%RANDOM%"
mkdir "%VERIFY_DIR%" >nul 2>nul
if errorlevel 1 exit /b 1

call scripts\build\package.bat
if errorlevel 1 exit /b 1
for /f %%V in ('%PYTHON% -c "from Src.version import __version__; print(__version__)"') do set "PACKAGE_VERSION=%%V"
if not defined PACKAGE_VERSION (
  echo [エラー] Python packageのバージョンを取得できませんでした。
  exit /b 1
)
set "WHEEL=.build\packages\tomiya_code_atlas-%PACKAGE_VERSION%-py3-none-any.whl"
set "SDIST=.build\packages\tomiya_code_atlas-%PACKAGE_VERSION%.tar.gz"
if not exist "%WHEEL%" (
  echo [エラー] Python wheelが生成されませんでした。
  exit /b 1
)
if not exist "%SDIST%" (
  echo [エラー] Python source archiveが生成されませんでした。
  exit /b 1
)
"%PYTHON%" tools\verify_wheel.py "%WHEEL%"
if errorlevel 1 exit /b 1
"%PYTHON%" tools\verify_wheel.py "%SDIST%"
if errorlevel 1 exit /b 1

call build_exe.bat
if errorlevel 1 exit /b 1
if not exist ".build\dist\tomiya-code-atlas\tomiya-code-atlas.exe" (
  echo [エラー] EXEが生成されませんでした。
  exit /b 1
)

>"%VERIFY_DIR%\sample.py" echo def load_config():
>>"%VERIFY_DIR%\sample.py" echo     return {}
>"%VERIFY_DIR%\workflow.yml" echo name: Build
>>"%VERIFY_DIR%\workflow.yml" echo on: [push]
>>"%VERIFY_DIR%\workflow.yml" echo jobs:
>>"%VERIFY_DIR%\workflow.yml" echo   test:
>>"%VERIFY_DIR%\workflow.yml" echo     steps:
>>"%VERIFY_DIR%\workflow.yml" echo       - name: pytest
>>"%VERIFY_DIR%\workflow.yml" echo         run: pytest

call run_dist.bat --version >"%VERIFY_DIR%\version.out"
if errorlevel 1 exit /b 1
findstr /c:"Tomiya Code Atlas" "%VERIFY_DIR%\version.out" >nul
if errorlevel 1 (
  echo [エラー] run_dist.bat で配布アプリを起動できませんでした。
  exit /b 1
)

call run_dist.bat comment "%VERIFY_DIR%\sample.py" >"%VERIFY_DIR%\comment.out"
if errorlevel 1 exit /b 1
findstr /c:"# Retrieves config." "%VERIFY_DIR%\comment.out" >nul
if errorlevel 1 (
  echo [エラー] コメント生成結果の確認に失敗しました。
  exit /b 1
)

call run_dist.bat ci "%VERIFY_DIR%\workflow.yml" --output "%VERIFY_DIR%\ci.mmd"
if errorlevel 1 exit /b 1
findstr /c:"flowchart LR" "%VERIFY_DIR%\ci.mmd" >nul
if errorlevel 1 (
  echo [エラー] CI Mermaid出力の確認に失敗しました。
  exit /b 1
)

call scripts\build\run_source.bat --version >"%VERIFY_DIR%\source-version.out"
if errorlevel 1 exit /b 1
findstr /c:"Tomiya Code Atlas" "%VERIFY_DIR%\source-version.out" >nul
if errorlevel 1 (
  echo [エラー] scripts\build\run_source.bat でソース版アプリを起動できませんでした。
  exit /b 1
)

"%PYTHON%" -m compileall -q Src tests
if errorlevel 1 exit /b 1
"%PYTHON%" tools\verify_distribution.py --exe .build\dist\tomiya-code-atlas\tomiya-code-atlas.exe --gui
if errorlevel 1 exit /b 1
call scripts\dev\context.bat policy-check
if errorlevel 1 exit /b 1
"%PYTHON%" -m pytest --basetemp "%VERIFY_DIR%\pytest-temp"
if errorlevel 1 exit /b 1
git diff --check
if errorlevel 1 exit /b 1

rmdir /s /q "%VERIFY_DIR%" >nul 2>nul
echo.
echo buildと検証が完了しました。
exit /b 0
