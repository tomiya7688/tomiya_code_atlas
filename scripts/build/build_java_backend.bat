@echo off
chcp 65001 >nul
setlocal EnableExtensions
cd /d "%~dp0\..\.."

if not defined JAVA_HOME (
    for %%I in (javac.exe) do set "JAVAC_PATH=%%~$PATH:I"
    if defined JAVAC_PATH for %%I in ("%JAVAC_PATH%") do set "JAVA_HOME=%%~dpI.."
)
if not exist "%JAVA_HOME%\bin\javac.exe" (
    echo [エラー] Java parserのbuildにはJDK 25が必要です。JAVA_HOMEを設定してください。
    exit /b 1
)
"%JAVA_HOME%\bin\javac.exe" -version 2>&1 | findstr /b /c:"javac 25." >nul
if errorlevel 1 (
    echo [エラー] Java parserはJDK 25でbuildします。検出したJDK: %JAVA_HOME%
    exit /b 1
)
if not exist "%JAVA_HOME%\bin\jlink.exe" (
    echo [エラー] private Java runtimeの作成にjlinkが必要です。
    exit /b 1
)
where mvn >nul 2>nul
if errorlevel 1 (
    echo [エラー] Java helperのbuildにApache Mavenが必要です。
    exit /b 1
)

if not exist ".build\dist\backends\java" mkdir ".build\dist\backends\java"
if not exist ".build\metadata" mkdir ".build\metadata"
mvn -q -f "backend-src\java\pom.xml" -DskipTests package
if errorlevel 1 exit /b 1
if not exist "backend-src\java\target\tomiya-java-backend.jar" (
    echo [エラー] JavaParser helper JARが生成されませんでした。
    exit /b 1
)
copy /y "backend-src\java\target\tomiya-java-backend.jar" ".build\dist\backends\java\tomiya-java-backend.jar" >nul
if errorlevel 1 exit /b 1

"%JAVA_HOME%\bin\jdeps.exe" --ignore-missing-deps --multi-release 17 --print-module-deps ".build\dist\backends\java\tomiya-java-backend.jar" > ".build\metadata\java-runtime-modules.txt"
if errorlevel 1 exit /b 1
set "JAVA_MODULES="
for /f "usebackq delims=" %%M in (".build\metadata\java-runtime-modules.txt") do set "JAVA_MODULES=%%M"
if not defined JAVA_MODULES (
    echo [エラー] Java helperのruntime moduleを特定できませんでした。
    exit /b 1
)

if exist ".build\dist\backends\java\runtime" rmdir /s /q ".build\dist\backends\java\runtime"
"%JAVA_HOME%\bin\jlink.exe" --module-path "%JAVA_HOME%\jmods" --add-modules %JAVA_MODULES% --strip-debug --no-header-files --no-man-pages --compress=2 --output ".build\dist\backends\java\runtime"
if errorlevel 1 exit /b 1
if not exist ".build\dist\backends\java\runtime\bin\java.exe" (
    echo [エラー] private Java runtimeが生成されませんでした。
    exit /b 1
)
copy /y "backends\licenses\javaparser-NOTICE.txt" ".build\dist\backends\java\javaparser-NOTICE.txt" >nul
if errorlevel 1 exit /b 1
echo JavaParser helperとprivate Java runtimeのone-dir buildが完了しました。
endlocal & exit /b 0
