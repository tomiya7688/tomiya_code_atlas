@echo off
setlocal
cd /d "%~dp0"

set "APP=.build\dist\tomiya-code-atlas\tomiya-code-atlas.exe"
if not exist "%APP%" (
    echo The built app was not found: %APP%
    echo Run build_exe.bat first, or use the downloaded distribution folder directly.
    exit /b 1
)

"%APP%" %*
set "RESULT=%errorlevel%"
endlocal & exit /b %RESULT%
