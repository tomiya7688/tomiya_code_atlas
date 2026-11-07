@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

if not exist ".venv\Scripts\python.exe" (
    echo [エラー] Python build環境がありません。scripts\build\setup.batを実行してください。
    exit /b 1
)

if not exist ".build\temp" mkdir ".build\temp"
if errorlevel 1 exit /b 1
set "TEMP=%CD%\.build\temp"
set "TMP=%TEMP%"

if not exist ".build\pyinstaller\config" mkdir ".build\pyinstaller\config"
if errorlevel 1 exit /b 1
set "PYINSTALLER_CONFIG_DIR=%CD%\.build\pyinstaller\config"

if not exist ".build\pyinstaller\work" mkdir ".build\pyinstaller\work"
if errorlevel 1 exit /b 1
if not exist ".build\pyinstaller\spec" mkdir ".build\pyinstaller\spec"
if errorlevel 1 exit /b 1
if not exist ".build\dist\backends" mkdir ".build\dist\backends"
if errorlevel 1 exit /b 1

if exist ".build\dist\backends\tomiya-python-backend" rmdir /s /q ".build\dist\backends\tomiya-python-backend"
if errorlevel 1 (
    echo [エラー] 以前のPython parser配布先を削除できませんでした。
    exit /b 1
)

".venv\Scripts\python.exe" -m PyInstaller --noconfirm --clean --onedir --name tomiya-python-backend --paths "%CD%" --collect-data certifi --distpath ".build\dist\backends" --workpath ".build\pyinstaller\work" --specpath ".build\pyinstaller\spec" "parser-backends\python\main.py"
if errorlevel 1 exit /b 1

if not exist ".build\dist\backends\tomiya-python-backend\tomiya-python-backend.exe" (
    echo [エラー] CPython AST helperのone-dir EXEが生成されませんでした。
    exit /b 1
)

".venv\Scripts\python.exe" tools\verify_python_tls.py --bundle-root ".build\dist\backends\tomiya-python-backend"
if errorlevel 1 (
    echo [エラー] Python parser配布物に安全なcertifi CA bundleが含まれていません。
    exit /b 1
)

if not exist ".build\dist\backends\tomiya-python-backend\licenses" mkdir ".build\dist\backends\tomiya-python-backend\licenses"
if errorlevel 1 exit /b 1
".venv\Scripts\python.exe" scripts\build\copy_python_runtime_license.py ".build\dist\backends\tomiya-python-backend\licenses\Python-LICENSE.txt"
if errorlevel 1 exit /b 1
copy /y "parser-backends\licenses\PyInstaller-COPYING.txt" ".build\dist\backends\tomiya-python-backend\licenses\PyInstaller-COPYING.txt" >nul
if errorlevel 1 exit /b 1
copy /y "parser-backends\licenses\tree-sitter-LICENSE.txt" ".build\dist\backends\tomiya-python-backend\licenses\tree-sitter-LICENSE.txt" >nul
if errorlevel 1 exit /b 1
copy /y "parser-backends\licenses\tree-sitter-gdscript-LICENSE.txt" ".build\dist\backends\tomiya-python-backend\licenses\tree-sitter-gdscript-LICENSE.txt" >nul
if errorlevel 1 exit /b 1
echo Python AST and GDScript Tree-sitter parser helper one-dir build complete.
endlocal
exit /b 0
