@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

if not exist ".venv\Scripts\python.exe" (
  echo [エラー] Pythonのtest環境がありません。scripts\build\setup.batを実行してください。
  exit /b 1
)

if not exist ".build\temp" mkdir ".build\temp"
set "TEMP=%CD%\.build\temp"
set "TMP=%TEMP%"
set "VERIFY_DIR=%TEMP%\tomiya-code-atlas-verify-%RANDOM%"
mkdir "%VERIFY_DIR%" >nul 2>nul
if errorlevel 1 exit /b 1

call build_exe.bat
if errorlevel 1 exit /b 1

call run_dist.bat --version >"%VERIFY_DIR%\version.out"
if errorlevel 1 exit /b 1
findstr /c:"Tomiya Code Atlas (Go)" "%VERIFY_DIR%\version.out" >nul
if errorlevel 1 (
  echo [エラー] Go版EXEのversion smoke testに失敗しました。
  exit /b 1
)

call run_dist.bat --help >"%VERIFY_DIR%\help.out"
if errorlevel 1 exit /b 1
findstr /c:"--help, -h" "%VERIFY_DIR%\help.out" >nul
if errorlevel 1 (
  echo [エラー] Go版EXEのhelp smoke testに失敗しました。
  exit /b 1
)

.venv\Scripts\python.exe scripts\build\smoke_python_backend.py ".build\dist\backends\tomiya-python-backend\tomiya-python-backend.exe"
if errorlevel 1 exit /b 1
.venv\Scripts\python.exe tools\csharp_backend_smoke.py ".build\dist\backends\csharp\tomiya-csharp-backend.exe"
if errorlevel 1 exit /b 1
.venv\Scripts\python.exe tools\cpp_backend_smoke.py ".build\dist\backends\cpp\tomiya-cpp-backend\tomiya-cpp-backend.exe"
if errorlevel 1 exit /b 1
.venv\Scripts\python.exe tools\java_backend_smoke.py ".build\dist\backends\java\runtime\bin\java.exe" -jar ".build\dist\backends\java\tomiya-java-backend.jar"
if errorlevel 1 exit /b 1

".venv\Scripts\python.exe" -m compileall -q Src tests tools
if errorlevel 1 exit /b 1
".venv\Scripts\python.exe" -m pytest -p no:cacheprovider --ignore-glob="pytest-cache-files-*" --basetemp "%VERIFY_DIR%\pytest-temp"
if errorlevel 1 exit /b 1
call scripts\dev\context.bat policy-check
if errorlevel 1 exit /b 1
git diff --check
if errorlevel 1 exit /b 1

rmdir /s /q "%VERIFY_DIR%" >nul 2>nul
echo.
echo Go配布EXE、Python/GDScript、C# Roslyn、C++ Clang、JavaParser helper、Python source testsの検証が完了しました。
exit /b 0
