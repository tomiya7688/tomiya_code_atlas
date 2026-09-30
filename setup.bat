@echo off
setlocal
cd /d "%~dp0"

where py >nul 2>nul
if not errorlevel 1 (
    set "PYTHON=py -3.12"
) else (
    where python >nul 2>nul
    if errorlevel 1 (
        echo Python 3.12 was not found. Install it and add Python to PATH.
        exit /b 1
    )
    set "PYTHON=python"
)

%PYTHON% -c "import sys; raise SystemExit(0 if sys.version_info[:2] == (3, 12) else 1)" >nul 2>nul
if errorlevel 1 (
    echo Python 3.12 is required. Install it, then run setup.bat again.
    exit /b 1
)

if not exist ".venv\Scripts\python.exe" (
    %PYTHON% -m venv .venv
    if errorlevel 1 exit /b 1
)

".venv\Scripts\python.exe" -c "import sys; raise SystemExit(0 if sys.version_info[:2] == (3, 12) else 1)" >nul 2>nul
if errorlevel 1 (
    echo The existing .venv does not use Python 3.12. Remove .venv and run setup.bat again.
    exit /b 1
)

if not exist ".build\metadata" mkdir ".build\metadata"
if errorlevel 1 exit /b 1

".venv\Scripts\python.exe" -m pip install --upgrade pip
if errorlevel 1 exit /b 1
".venv\Scripts\python.exe" -m pip install -e ".[test,exe]" build
if errorlevel 1 exit /b 1

echo.
echo Setup completed. Run the source with run.bat or build the app with build_exe.bat.
endlocal & exit /b 0
