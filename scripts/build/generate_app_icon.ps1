param(
    [string]$OutputDirectory = ""
)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Drawing

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$characterAssetRoot = Join-Path $repositoryRoot "assets\characters\atlas-kun"
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $repositoryRoot ".build\generated-assets"
}
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null

$largeImagePath = Join-Path $characterAssetRoot "atlas-kun-1920.png"
$smallImagePath = Join-Path $characterAssetRoot "atlas-kun-64.png"
foreach ($path in @($largeImagePath, $smallImagePath)) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "キャラクター素材が見つかりません: $path"
    }
}

$largeImage = [System.Drawing.Bitmap]::FromFile($largeImagePath)
$smallImage = [System.Drawing.Bitmap]::FromFile($smallImagePath)
try {
    if ($largeImage.Width -ne 1920 -or $largeImage.Height -ne 1920) {
        throw "高解像度素材は1920x1920である必要があります: $($largeImage.Width)x$($largeImage.Height)"
    }
    if ($smallImage.Width -ne 64 -or $smallImage.Height -ne 64) {
        throw "小サイズ素材は64x64である必要があります: $($smallImage.Width)x$($smallImage.Height)"
    }

    function New-IconPngBytes([System.Drawing.Image]$Source, [int]$Size, [bool]$PixelArt) {
        $bitmap = [System.Drawing.Bitmap]::new(
            $Size,
            $Size,
            [System.Drawing.Imaging.PixelFormat]::Format32bppArgb
        )
        $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
        $stream = [System.IO.MemoryStream]::new()
        try {
            $graphics.Clear([System.Drawing.Color]::Transparent)
            $graphics.CompositingMode = [System.Drawing.Drawing2D.CompositingMode]::SourceCopy
            if ($PixelArt) {
                $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::NearestNeighbor
                $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::None
            }
            else {
                $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
                $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
                $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
            }
            $graphics.DrawImage(
                $Source,
                [System.Drawing.Rectangle]::new(0, 0, $Size, $Size),
                [System.Drawing.Rectangle]::new(0, 0, $Source.Width, $Source.Height),
                [System.Drawing.GraphicsUnit]::Pixel
            )
            $bitmap.Save($stream, [System.Drawing.Imaging.ImageFormat]::Png)
            return ,$stream.ToArray()
        }
        finally {
            $stream.Dispose()
            $graphics.Dispose()
            $bitmap.Dispose()
        }
    }

    $appPng = New-IconPngBytes $largeImage 256 $false
    [System.IO.File]::WriteAllBytes((Join-Path $OutputDirectory "atlas-kun-256.png"), $appPng)

    $sizes = @(256, 128, 64, 48, 32, 24, 16)
    $frames = foreach ($size in $sizes) {
        $usePixelArt = $size -le 64
        $source = if ($usePixelArt) { $smallImage } else { $largeImage }
        [pscustomobject]@{
            Size = $size
            Bytes = New-IconPngBytes $source $size $usePixelArt
        }
    }

    $iconPath = Join-Path $OutputDirectory "atlas-kun.ico"
    $file = [System.IO.File]::Create($iconPath)
    $writer = [System.IO.BinaryWriter]::new($file)
    try {
        $writer.Write([UInt16]0)
        $writer.Write([UInt16]1)
        $writer.Write([UInt16]$frames.Count)
        $offset = 6 + (16 * $frames.Count)
        foreach ($frame in $frames) {
            $dimension = if ($frame.Size -eq 256) { [byte]0 } else { [byte]$frame.Size }
            $writer.Write($dimension)
            $writer.Write($dimension)
            $writer.Write([byte]0)
            $writer.Write([byte]0)
            $writer.Write([UInt16]1)
            $writer.Write([UInt16]32)
            $writer.Write([UInt32]$frame.Bytes.Length)
            $writer.Write([UInt32]$offset)
            $offset += $frame.Bytes.Length
        }
        foreach ($frame in $frames) {
            $writer.Write($frame.Bytes)
        }
    }
    finally {
        $writer.Dispose()
        $file.Dispose()
    }
}
finally {
    $largeImage.Dispose()
    $smallImage.Dispose()
}

Write-Output "アプリ用アイコンを生成しました: $OutputDirectory"
