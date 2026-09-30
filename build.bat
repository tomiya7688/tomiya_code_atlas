@echo off
setlocal
cd /d "%~dp0"

if not exist ".venv\Scripts\python.exe" (
    echo Project build environment not found. Run setup.bat first.
    exit /b 1
)

if not exist ".build\packages" mkdir ".build\packages"
if not exist ".build\metadata" mkdir ".build\metadata"
".venv\Scripts\python.exe" -m build --outdir .build\packages
if errorlevel 1 exit /b 1

echo.
echo Build completed. Artifacts are in .build\packages\
endlocal
