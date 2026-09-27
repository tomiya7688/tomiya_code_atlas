@echo off
setlocal
cd /d "%~dp0"

where py >nul 2>nul
if %errorlevel%==0 (
    set "PYTHON=py -3"
) else (
    set "PYTHON=python"
)

%PYTHON% -m pip install --upgrade pip build
if errorlevel 1 exit /b 1

if not exist ".build\packages" mkdir ".build\packages"
%PYTHON% -m build --outdir .build\packages
if errorlevel 1 exit /b 1

echo.
echo Build completed. Artifacts are in .build\packages\
endlocal
