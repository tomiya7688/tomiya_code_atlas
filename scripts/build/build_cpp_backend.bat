@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

if not exist ".venv\Scripts\python.exe" (
    echo [エラー] Python build環境がありません。scripts\build\setup.batを実行してください。
    exit /b 1
)

".venv\Scripts\python.exe" -c "import clang.cindex"
if errorlevel 1 (
    echo [エラー] Clang parser bindingがありません。Python 3.12以降でscripts\build\setup.batを実行してください。
    exit /b 1
)

if not exist ".build\pyinstaller\work" mkdir ".build\pyinstaller\work"
if not exist ".build\pyinstaller\spec" mkdir ".build\pyinstaller\spec"
if not exist ".build\dist\backends" mkdir ".build\dist\backends"

".venv\Scripts\python.exe" -m PyInstaller --noconfirm --clean --onedir --name tomiya-cpp-backend --paths "%CD%" --collect-all clang --distpath ".build\dist\backends\cpp" --workpath ".build\pyinstaller\work" --specpath ".build\pyinstaller\spec" "backends\cpp\main.py"
if errorlevel 1 exit /b 1

if not exist ".build\dist\backends\cpp\tomiya-cpp-backend\tomiya-cpp-backend.exe" (
    echo [エラー] Clang parser helperのone-dir EXEが生成されませんでした。
    exit /b 1
)
if not exist ".build\dist\backends\cpp\tomiya-cpp-backend\licenses" mkdir ".build\dist\backends\cpp\tomiya-cpp-backend\licenses"
".venv\Scripts\python.exe" scripts\build\copy_python_runtime_license.py ".build\dist\backends\cpp\tomiya-cpp-backend\licenses\Python-LICENSE.txt"
if errorlevel 1 exit /b 1
copy /y "backends\licenses\PyInstaller-COPYING.txt" ".build\dist\backends\cpp\tomiya-cpp-backend\licenses\PyInstaller-COPYING.txt" >nul
if errorlevel 1 exit /b 1
copy /y "backends\licenses\libclang-ng-LICENSE.txt" ".build\dist\backends\cpp\tomiya-cpp-backend\licenses\LLVM-LICENSE.txt" >nul
if errorlevel 1 exit /b 1
echo C++ Clang parser helper one-dir build complete.
endlocal
exit /b 0
