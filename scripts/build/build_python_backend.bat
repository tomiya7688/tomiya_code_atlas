@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

if not exist ".venv\Scripts\python.exe" (
    echo [エラー] Python build環境がありません。scripts\build\setup.batを実行してください。
    exit /b 1
)

if not exist ".build\pyinstaller\work" mkdir ".build\pyinstaller\work"
if not exist ".build\pyinstaller\spec" mkdir ".build\pyinstaller\spec"
if not exist ".build\dist\backends" mkdir ".build\dist\backends"

".venv\Scripts\python.exe" -m PyInstaller --noconfirm --clean --onedir --name tomiya-python-backend --paths "%CD%" --distpath ".build\dist\backends" --workpath ".build\pyinstaller\work" --specpath ".build\pyinstaller\spec" "backends\python\main.py"
if errorlevel 1 exit /b 1

if not exist ".build\dist\backends\tomiya-python-backend\tomiya-python-backend.exe" (
    echo [エラー] CPython AST helperのone-dir EXEが生成されませんでした。
    exit /b 1
)

if not exist ".build\dist\backends\tomiya-python-backend\licenses" mkdir ".build\dist\backends\tomiya-python-backend\licenses"
".venv\Scripts\python.exe" scripts\build\copy_python_runtime_license.py ".build\dist\backends\tomiya-python-backend\licenses\Python-LICENSE.txt"
if errorlevel 1 exit /b 1
copy /y "backends\licenses\PyInstaller-COPYING.txt" ".build\dist\backends\tomiya-python-backend\licenses\PyInstaller-COPYING.txt" >nul
if errorlevel 1 exit /b 1
copy /y "backends\licenses\tree-sitter-LICENSE.txt" ".build\dist\backends\tomiya-python-backend\licenses\tree-sitter-LICENSE.txt" >nul
if errorlevel 1 exit /b 1
copy /y "backends\licenses\tree-sitter-gdscript-LICENSE.txt" ".build\dist\backends\tomiya-python-backend\licenses\tree-sitter-gdscript-LICENSE.txt" >nul
if errorlevel 1 exit /b 1
echo CPython AST helper one-dir build complete.
endlocal
exit /b 0
