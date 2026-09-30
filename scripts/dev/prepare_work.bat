@echo off
setlocal
cd /d "%~dp0\..\.."

call scripts\dev\reducer.bat setup
if errorlevel 1 (
  echo [WARN] ai-context-reducer setup failed; continuing with repository-local context tools. 1>&2
)

where python >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Python was not found in PATH.
  exit /b 1
)

python tools\next_issue.py
if errorlevel 1 exit /b %errorlevel%

python tools\context_tool.py remote-delta --excerpt-lines 40 > .codex\remote_delta.json
if errorlevel 1 exit /b %errorlevel%

python tools\context_tool.py context-pack
if errorlevel 1 exit /b %errorlevel%

echo.
echo Work context ready:
echo   .codex\next_issue.md
echo   .codex\remote_delta.json
echo   .codex\context_pack.md
endlocal
