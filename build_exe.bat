@echo off
setlocal
cd /d "%~dp0"

where py >nul 2>nul
if %errorlevel%==0 (
    set "PYTHON=py -3"
) else (
    set "PYTHON=python"
)

where dotnet >nul 2>nul
if errorlevel 1 (
    echo .NET SDK 10 is required to build the bundled C# Roslyn backend.
    exit /b 1
)

where mvn >nul 2>nul
if errorlevel 1 (
    echo Maven is required to build the bundled JavaParser backend.
    exit /b 1
)
if "%JAVA_HOME%"=="" (
    echo JAVA_HOME must point to the JDK used to build the private Java runtime.
    exit /b 1
)

if not exist ".build\backends" mkdir ".build\backends"
if exist ".build\backends\java" rmdir /S /Q ".build\backends\java"
call mvn -q -f "backend-src\java\pom.xml" package
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

%PYTHON% -m pip install -e ".[exe]"
if errorlevel 1 exit /b 1

if not exist ".build\dist" mkdir ".build\dist"
if not exist ".build\pyinstaller-work" mkdir ".build\pyinstaller-work"
if not exist ".build\spec" mkdir ".build\spec"
%PYTHON% -m PyInstaller --onedir --clean --name tomiya-code-atlas --distpath .build\dist --workpath .build\pyinstaller-work --specpath .build\spec -y app.py
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
echo App build completed: .build\dist\tomiya-code-atlas\tomiya-code-atlas.exe
endlocal
exit /b 0
