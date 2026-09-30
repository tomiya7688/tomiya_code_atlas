@echo off
setlocal
cd /d "%~dp0\..\.."

set "REDUCER_REPO=https://github.com/tomiya7688/ai-context-reducer.git"
set "REDUCER_DIR=%CD%\.dev\ai-context-reducer"
set "ACTION=%~1"
if "%ACTION%"=="" set "ACTION=setup"

where git >nul 2>nul
if errorlevel 1 (
  echo [ERROR] git was not found in PATH. 1>&2
  exit /b 1
)

if not exist "%CD%\.dev" mkdir "%CD%\.dev"

if exist "%REDUCER_DIR%\.git" (
  echo [ai-context-reducer] updating cached development checkout...
  git -C "%REDUCER_DIR%" fetch origin main --depth=1
  if errorlevel 1 (
    echo [WARN] Could not fetch latest reducer; using cached checkout. 1>&2
  ) else (
    git -C "%REDUCER_DIR%" checkout -q main
    git -C "%REDUCER_DIR%" reset --hard origin/main >nul
    if errorlevel 1 exit /b 1
  )
) else (
  echo [ai-context-reducer] cloning latest development checkout...
  git clone --depth 1 --branch main "%REDUCER_REPO%" "%REDUCER_DIR%"
  if errorlevel 1 exit /b 1
)

for /f %%i in ('git -C "%REDUCER_DIR%" rev-parse --short HEAD') do set "REDUCER_SHA=%%i"
echo [ai-context-reducer] revision %REDUCER_SHA%

if /i "%ACTION%"=="update" exit /b 0
if /i "%ACTION%"=="setup" (
  call "%REDUCER_DIR%\tools\setup.bat" "%CD%"
  exit /b %errorlevel%
)
if /i "%ACTION%"=="analyze" (
  call "%REDUCER_DIR%\tools\analyze.bat" "%CD%"
  exit /b %errorlevel%
)

echo Usage: reducer.bat [setup^|analyze^|update] 1>&2
exit /b 2
