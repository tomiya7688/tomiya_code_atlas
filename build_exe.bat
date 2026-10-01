@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0"

if not exist ".build\temp" mkdir ".build\temp"
set "TEMP=%CD%\.build\temp"
set "TMP=%TEMP%"

if not exist ".venv\Scripts\python.exe" (
    echo buildに必要なPython環境を準備します。
    call scripts\build\setup.bat
    if errorlevel 1 (
        echo [エラー] Python環境を準備できませんでした。上に表示された原因を解消して再実行してください。
        exit /b 1
    )
)

where dotnet >nul 2>nul
if errorlevel 1 (
    echo [エラー] C# backendのbuildに.NET SDK 10が必要です。
    echo 入手先: https://dotnet.microsoft.com/ja-jp/download/dotnet/10.0
    exit /b 1
)
dotnet --list-sdks | findstr /b "10." >nul
if errorlevel 1 (
    echo [エラー] .NET SDK 10が見つかりません。SDK 10をインストールして再実行してください。
    echo 入手先: https://dotnet.microsoft.com/ja-jp/download/dotnet/10.0
    exit /b 1
)

if not defined JAVA_HOME (
    for /f "delims=" %%J in ('where javac 2^>nul') do if not defined JAVA_HOME for %%K in ("%%J") do set "JAVA_HOME=%%~dpK.."
)
if defined JAVA_HOME for %%J in ("%JAVA_HOME%") do set "JAVA_HOME=%%~fJ"
if "%JAVA_HOME%"=="" (
    echo [エラー] Java backendのbuildにJDK 25が必要です。JDK 25をインストールしてJAVA_HOMEを設定するか、binフォルダーをPATHに追加してください。
    echo 入手先: "https://adoptium.net/temurin/releases/?version=25^&os=windows^&arch=x64^&package=jdk"
    exit /b 1
)
if not exist "%JAVA_HOME%\bin\java.exe" (
    echo [エラー] JAVA_HOMEにbin\java.exeがありません: %JAVA_HOME%
    echo JAVA_HOMEをJDK 25のフォルダーに設定してください。
    exit /b 1
)
"%JAVA_HOME%\bin\javac.exe" -version 2>&1 | findstr /c:"25." >nul
if errorlevel 1 (
    echo [エラー] JDK 25が必要です。JAVA_HOMEをJDK 25のフォルダーに設定してください: %JAVA_HOME%
    echo 入手先: "https://adoptium.net/temurin/releases/?version=25^&os=windows^&arch=x64^&package=jdk"
    exit /b 1
)

set "MAVEN_CMD="
for /f "usebackq delims=" %%M in (`powershell -NoProfile -ExecutionPolicy Bypass -File "scripts\build\resolve_maven.ps1"`) do set "MAVEN_CMD=%%M"
if not defined MAVEN_CMD (
    echo [エラー] Apache Mavenを準備できませんでした。ネットワーク接続を確認して再実行してください。
    exit /b 1
)

if not exist ".build\backends" mkdir ".build\backends"
if exist ".build\backends\java" rmdir /S /Q ".build\backends\java"
call "%MAVEN_CMD%" -q -f "backend-src\java\pom.xml" package
if errorlevel 1 exit /b 1
mkdir ".build\backends\java"
copy /Y "backend-src\java\target\tomiya-java-backend.jar" ".build\backends\java\tomiya-java-backend.jar" >nul
if errorlevel 1 exit /b 1
if exist ".build\backends\java\runtime" rmdir /S /Q ".build\backends\java\runtime"
xcopy /E /I /Y "%JAVA_HOME%\*" ".build\backends\java\runtime\" >nul
if errorlevel 1 exit /b 1

if exist ".build\backends\csharp" rmdir /S /Q ".build\backends\csharp"
dotnet publish "backend-src\csharp\Tomiya.CSharp.Backend.csproj" -c Release -r win-x64 --self-contained true -p:PublishSingleFile=false -p:PublishTrimmed=false -o ".build\backends\csharp"
if errorlevel 1 exit /b 1

if not exist ".build\metadata" mkdir ".build\metadata"

if not exist ".build\dist" mkdir ".build\dist"
if exist ".build\dist\tomiya-code-atlas" rmdir /S /Q ".build\dist\tomiya-code-atlas"
if errorlevel 1 exit /b 1
if not exist ".build\pyinstaller-work" mkdir ".build\pyinstaller-work"
if not exist ".build\spec" mkdir ".build\spec"
if not exist ".build\generated-assets" mkdir ".build\generated-assets"
powershell -NoProfile -ExecutionPolicy Bypass -File "scripts\build\generate_app_icon.ps1" -OutputDirectory ".build\generated-assets"
if errorlevel 1 exit /b 1
for %%I in (".build\generated-assets\atlas-kun.ico") do set "APP_ICON=%%~fI"
".venv\Scripts\python.exe" -m PyInstaller --onedir --clean --icon "%APP_ICON%" --name tomiya-code-atlas --distpath .build\dist --workpath .build\pyinstaller-work --specpath .build\spec -y app.py
if errorlevel 1 exit /b 1

if not exist ".build\dist\tomiya-code-atlas\assets" mkdir ".build\dist\tomiya-code-atlas\assets"
copy /Y ".build\generated-assets\atlas-kun-256.png" ".build\dist\tomiya-code-atlas\assets\atlas-kun-256.png" >nul
if errorlevel 1 exit /b 1
copy /Y "assets\characters\atlas-kun\atlas-kun-64.png" ".build\dist\tomiya-code-atlas\assets\atlas-kun-64.png" >nul
if errorlevel 1 exit /b 1
copy /Y ".build\generated-assets\atlas-kun.ico" ".build\dist\tomiya-code-atlas\assets\atlas-kun.ico" >nul
if errorlevel 1 exit /b 1
copy /Y "assets\characters\atlas-kun\LICENSE.md" ".build\dist\tomiya-code-atlas\assets\LICENSE.md" >nul
if errorlevel 1 exit /b 1

if exist "config" (
    if not exist ".build\dist\tomiya-code-atlas\config" mkdir ".build\dist\tomiya-code-atlas\config"
    xcopy /E /I /Y "config\*" ".build\dist\tomiya-code-atlas\config\" >nul
    if errorlevel 1 exit /b 1
)

if exist "backends" (
    if not exist ".build\dist\tomiya-code-atlas\backends" mkdir ".build\dist\tomiya-code-atlas\backends"
    xcopy /E /I /Y "backends\*" ".build\dist\tomiya-code-atlas\backends\" >nul
    if errorlevel 1 exit /b 1
)
if exist ".build\backends" xcopy /E /I /Y ".build\backends\*" ".build\dist\tomiya-code-atlas\backends\" >nul
if errorlevel 1 exit /b 1

if not exist ".build\dist\tomiya-code-atlas\tomiya-code-atlas.exe" (
    echo PyInstaller output missing: .build\dist\tomiya-code-atlas\tomiya-code-atlas.exe
    exit /b 1
)
if not exist ".build\dist\tomiya-code-atlas\backends\java\tomiya-java-backend.jar" (
    echo Bundled Java helper missing from onedir output.
    exit /b 1
)
if not exist ".build\dist\tomiya-code-atlas\backends\java\runtime\bin\java.exe" (
    echo Bundled private Java runtime missing from onedir output.
    exit /b 1
)

echo.
echo Windowsアプリをbuildしました: .build\dist\tomiya-code-atlas\tomiya-code-atlas.exe
endlocal
exit /b 0
