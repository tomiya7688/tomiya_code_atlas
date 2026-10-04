@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

where dotnet >nul 2>nul
if errorlevel 1 (
    echo [エラー] C# Roslyn helperのbuildには.NET 10 SDKが必要です。
    echo 入手先: https://dotnet.microsoft.com/download/dotnet/10.0
    exit /b 1
)

dotnet --list-sdks | findstr /b /c:"10." >nul
if errorlevel 1 (
    echo [エラー] C# Roslyn helperのbuildには.NET 10 SDKが必要です。
    exit /b 1
)

if not exist ".build\dist\backends\csharp" mkdir ".build\dist\backends\csharp"
dotnet publish "backend-src\csharp\Tomiya.CSharp.Backend.csproj" -c Release -r win-x64 --self-contained true -p:PublishSingleFile=false -p:PublishTrimmed=false -o ".build\dist\backends\csharp"
if errorlevel 1 exit /b 1

if not exist ".build\dist\backends\csharp\tomiya-csharp-backend.exe" (
    echo [エラー] C# Roslyn helperのEXEが生成されませんでした。
    exit /b 1
)

if not exist ".build\dist\backends\csharp\licenses" mkdir ".build\dist\backends\csharp\licenses"
copy /y "backends\licenses\roslyn-LICENSE.txt" ".build\dist\backends\csharp\licenses\roslyn-LICENSE.txt" >nul
if errorlevel 1 exit /b 1
for %%I in (dotnet.exe) do set "DOTNET_EXE=%%~$PATH:I"
for %%I in ("%DOTNET_EXE%") do set "DOTNET_INSTALL_DIR=%%~dpI"
copy /y "%DOTNET_INSTALL_DIR%LICENSE.txt" ".build\dist\backends\csharp\licenses\dotnet-LICENSE.txt" >nul
if errorlevel 1 exit /b 1
copy /y "%DOTNET_INSTALL_DIR%ThirdPartyNotices.txt" ".build\dist\backends\csharp\licenses\dotnet-ThirdPartyNotices.txt" >nul
if errorlevel 1 exit /b 1

echo C# Roslyn helper build complete: .build\dist\backends\csharp
endlocal & exit /b 0
