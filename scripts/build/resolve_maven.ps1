$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$systemMaven = Get-Command mvn.cmd -ErrorAction SilentlyContinue
if ($systemMaven) {
    Write-Output $systemMaven.Source
    exit 0
}

$version = "3.9.16"
$repositoryRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$toolsRoot = Join-Path $repositoryRoot ".build\tools"
$mavenHome = Join-Path $toolsRoot "apache-maven-$version"
$mavenCommand = Join-Path $mavenHome "bin\mvn.cmd"
if (Test-Path -LiteralPath $mavenCommand -PathType Leaf) {
    Write-Output $mavenCommand
    exit 0
}

$archiveUrl = "https://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/$version/apache-maven-$version-bin.zip"
$expectedSha512 = "ed41650d42485cfc243fad22158caf9cbb5dc408ce7a09ddb94dd42a019de929ca43065bfa450612cf12bf78b5cafa3884b96c090de326ff590448c933454af3"
$temporaryDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("tomiya-maven-" + [guid]::NewGuid().ToString("N"))

try {
    $null = New-Item -ItemType Directory -Path $toolsRoot -Force
    $null = New-Item -ItemType Directory -Path $temporaryDirectory -Force
    $archivePath = Join-Path $temporaryDirectory "apache-maven-$version-bin.zip"

    Write-Verbose "Apache Maven $version を公式配布元から取得しています..."
    $null = Invoke-WebRequest -Uri $archiveUrl -OutFile $archivePath -UseBasicParsing
    $hashProvider = [System.Security.Cryptography.SHA512]::Create()
    $archiveStream = [System.IO.File]::OpenRead($archivePath)
    try {
        $actualSha512 = [System.BitConverter]::ToString($hashProvider.ComputeHash($archiveStream)).Replace("-", "").ToLowerInvariant()
    }
    finally {
        $archiveStream.Dispose()
        $hashProvider.Dispose()
    }
    if ($actualSha512 -ne $expectedSha512) {
        throw "Apache MavenのSHA-512検証に失敗しました。ダウンロードしたファイルを使用しません。"
    }

    $null = Expand-Archive -LiteralPath $archivePath -DestinationPath $temporaryDirectory -Force
    $extractedHome = Join-Path $temporaryDirectory "apache-maven-$version"
    if (-not (Test-Path -LiteralPath (Join-Path $extractedHome "bin\mvn.cmd") -PathType Leaf)) {
        throw "Apache Mavenの展開結果にbin\mvn.cmdがありません。"
    }

    if (Test-Path -LiteralPath $mavenHome) {
        Remove-Item -LiteralPath $mavenHome -Recurse -Force
    }
    Move-Item -LiteralPath $extractedHome -Destination $mavenHome
    Write-Output $mavenCommand
}
catch {
    [Console]::Error.WriteLine("[エラー] Apache Mavenを準備できませんでした。ネットワーク接続を確認して再実行してください。")
    [Console]::Error.WriteLine($_.Exception.Message)
    exit 1
}
finally {
    if (Test-Path -LiteralPath $temporaryDirectory) {
        Remove-Item -LiteralPath $temporaryDirectory -Recurse -Force -ErrorAction SilentlyContinue
    }
}
