@echo off
setlocal
cd /d "%~dp0\..\.."

where python >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Python was not found in PATH.
  exit /b 1
)

if "%~1"=="" (
  echo Usage: context.bat command [options]
  echo Run: python tools\context_tool.py --help
  exit /b 2
)

python tools\context_tool.py %*
exit /b %errorlevel%
