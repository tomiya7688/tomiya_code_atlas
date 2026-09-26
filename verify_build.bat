@echo off
setlocal EnableExtensions
cd /d "%~dp0"

where python >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Python was not found in PATH.
  exit /b 1
)
set "PYTHON=python"
where py >nul 2>nul
if not errorlevel 1 ( set "WHEEL_PYTHON=py -3" ) else ( set "WHEEL_PYTHON=%PYTHON%" )
set "VERIFY_DIR=%TEMP%\tomiya-code-atlas-verify-%RANDOM%"
mkdir "%VERIFY_DIR%" >nul 2>nul
if errorlevel 1 exit /b 1

call build.bat
if errorlevel 1 exit /b 1
if not exist "dist\tomiya_code_atlas-*.whl" (
  echo [ERROR] Python wheel was not generated.
  exit /b 1
)
if not exist "dist\tomiya_code_atlas-*.tar.gz" (
  echo [ERROR] Python source archive was not generated.
  exit /b 1
)
set "WHEEL="
for %%W in (dist\tomiya_code_atlas-*.whl) do set "WHEEL=%%~fW"
%WHEEL_PYTHON% tools\verify_wheel.py "%WHEEL%"
if errorlevel 1 exit /b 1
set "SDIST="
for %%S in (dist\tomiya_code_atlas-*.tar.gz) do set "SDIST=%%~fS"
%WHEEL_PYTHON% tools\verify_wheel.py "%SDIST%"
if errorlevel 1 exit /b 1

call build_exe.bat
if errorlevel 1 exit /b 1
if not exist "dist\tomiya-code-atlas\tomiya-code-atlas.exe" (
  echo [ERROR] EXE was not generated.
  exit /b 1
)

>"%VERIFY_DIR%\sample.py" echo def load_config():
>>"%VERIFY_DIR%\sample.py" echo     return {}
>"%VERIFY_DIR%\workflow.yml" echo name: Build
>>"%VERIFY_DIR%\workflow.yml" echo on: [push]
>>"%VERIFY_DIR%\workflow.yml" echo jobs:
>>"%VERIFY_DIR%\workflow.yml" echo   test:
>>"%VERIFY_DIR%\workflow.yml" echo     steps:
>>"%VERIFY_DIR%\workflow.yml" echo       - name: pytest
>>"%VERIFY_DIR%\workflow.yml" echo         run: pytest

"dist\tomiya-code-atlas\tomiya-code-atlas.exe" comment "%VERIFY_DIR%\sample.py" >"%VERIFY_DIR%\comment.out"
if errorlevel 1 exit /b 1
findstr /c:"# Retrieves config." "%VERIFY_DIR%\comment.out" >nul
if errorlevel 1 (
  echo [ERROR] Comment output verification failed.
  exit /b 1
)

"dist\tomiya-code-atlas\tomiya-code-atlas.exe" ci "%VERIFY_DIR%\workflow.yml" --output "%VERIFY_DIR%\ci.mmd"
if errorlevel 1 exit /b 1
findstr /c:"flowchart LR" "%VERIFY_DIR%\ci.mmd" >nul
if errorlevel 1 (
  echo [ERROR] CI Mermaid output verification failed.
  exit /b 1
)

%PYTHON% -m compileall -q Src tests
if errorlevel 1 exit /b 1
python tools\verify_distribution.py --exe dist\tomiya-code-atlas\tomiya-code-atlas.exe --gui
if errorlevel 1 exit /b 1
call context.bat policy-check
if errorlevel 1 exit /b 1
%PYTHON% -m pytest --basetemp "%VERIFY_DIR%\pytest-temp"
if errorlevel 1 exit /b 1
git diff --check
if errorlevel 1 exit /b 1

rmdir /s /q "%VERIFY_DIR%" >nul 2>nul
echo.
echo Build and verification completed successfully.
exit /b 0
