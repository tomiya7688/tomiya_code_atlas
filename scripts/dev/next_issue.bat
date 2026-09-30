@echo off
setlocal
cd /d "%~dp0\..\.."

where gh >nul 2>nul
if errorlevel 1 (
  echo [ERROR] GitHub CLI ^(gh^) was not found.
  echo Install GitHub CLI and run: gh auth login
  exit /b 1
)

where python >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Python was not found in PATH.
  exit /b 1
)

python tools\next_issue.py
if errorlevel 1 exit /b %errorlevel%

echo.
echo Highest-priority issue context is ready in .codex\next_issue.md
endlocal
