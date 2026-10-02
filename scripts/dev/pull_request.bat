@echo off
setlocal
cd /d "%~dp0\..\.."

where python >nul 2>nul
if errorlevel 1 (
    echo ERROR: Python was not found in PATH.
    exit /b 1
)

call scripts\build\verify.bat
if errorlevel 1 exit /b 1

python tools\create_pr.py
exit /b %errorlevel%
