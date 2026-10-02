@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0"

call scripts\build\build_go.bat
set "RESULT=%errorlevel%"
endlocal & exit /b %RESULT%
