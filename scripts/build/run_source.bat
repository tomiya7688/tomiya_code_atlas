@echo off
setlocal
cd /d "%~dp0\..\.."

if not exist ".venv\Scripts\python.exe" (
    echo Project environment not found. Run scripts\build\setup.bat first.
    exit /b 1
)

".venv\Scripts\python.exe" app.py %*
set "RESULT=%errorlevel%"
endlocal & exit /b %RESULT%
