Add-Type -AssemblyName System.Drawing

$baseDir = "D:\Github\orzu-pos"
$assetsIconsDir = Join-Path $baseDir "assets\icons"
$webIconsDir = Join-Path $baseDir "web\icons"

New-Item -ItemType Directory -Force -Path $assetsIconsDir | Out-Null
New-Item -ItemType Directory -Force -Path $webIconsDir | Out-Null

$iconMapping = @{
    "kassa"     = "Glossy Burgundy POS Receipt Icon.png"
    "admin"     = "Maroon Analytics Dashboard Icon.png"
    "installer" = "Burgundy Shop Badge and POS Icon for website.png"
    "server"    = "Glossy Burgundy Circular Emblem.png"
    "backup"    = "Glossy Burgundy Shield Refresh Emblem.png"
    "db"        = "Burgundy Return Package Icon.png"
}

function Convert-PngToIco {
    param(
        [string]$PngPath,
        [string]$IcoPath
    )

    $sizes = @(256, 128, 64, 48, 32, 16)
    $srcImg = [System.Drawing.Image]::FromFile($PngPath)
    $pngStreams = @()

    foreach ($size in $sizes) {
        $bmp = New-Object System.Drawing.Bitmap $size, $size
        $g = [System.Drawing.Graphics]::FromImage($bmp)
        $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
        $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
        $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
        $g.DrawImage($srcImg, 0, 0, $size, $size)
        $g.Dispose()

        $ms = New-Object System.IO.MemoryStream
        $bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
        $bmp.Dispose()
        $pngStreams += $ms
    }
    $srcImg.Dispose()

    $fs = [System.IO.File]::Create($IcoPath)
    $bw = New-Object System.IO.BinaryWriter($fs)

    $bw.Write([uint16]0)
    $bw.Write([uint16]1)
    $bw.Write([uint16]$sizes.Count)

    $offset = 6 + (16 * $sizes.Count)
    for ($i = 0; $i -lt $sizes.Count; $i++) {
        $s = $sizes[$i]
        $w = if ($s -ge 256) { [byte]0 } else { [byte]$s }
        $h = if ($s -ge 256) { [byte]0 } else { [byte]$s }
        $bw.Write([byte]$w)
        $bw.Write([byte]$h)
        $bw.Write([byte]0)
        $bw.Write([byte]0)
        $bw.Write([uint16]1)
        $bw.Write([uint16]32)
        $len = [uint32]$pngStreams[$i].Length
        $bw.Write([uint32]$len)
        $bw.Write([uint32]$offset)
        $offset += $len
    }

    for ($i = 0; $i -lt $sizes.Count; $i++) {
        $bytes = $pngStreams[$i].ToArray()
        $bw.Write($bytes)
        $pngStreams[$i].Dispose()
    }

    $bw.Flush()
    $bw.Close()
    $fs.Close()
    Write-Host "Created ICO: $IcoPath"
}

function Save-ResizedPng {
    param(
        [string]$SrcPath,
        [string]$DestPath,
        [int]$Width,
        [int]$Height
    )
    $srcImg = [System.Drawing.Image]::FromFile($SrcPath)
    $bmp = New-Object System.Drawing.Bitmap $Width, $Height
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
    $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
    $g.DrawImage($srcImg, 0, 0, $Width, $Height)
    $g.Dispose()
    $srcImg.Dispose()
    $bmp.Save($DestPath, [System.Drawing.Imaging.ImageFormat]::Png)
    $bmp.Dispose()
    Write-Host "Created PNG: $DestPath"
}

Write-Host "Processing icons for ORZU POS..."

foreach ($key in $iconMapping.Keys) {
    $srcFile = Join-Path $baseDir $iconMapping[$key]
    if (Test-Path $srcFile) {
        $icoFile = Join-Path $assetsIconsDir "$key.ico"
        Convert-PngToIco -PngPath $srcFile -IcoPath $icoFile

        $webPng = Join-Path $webIconsDir "$key.png"
        Save-ResizedPng -SrcPath $srcFile -DestPath $webPng -Width 256 -Height 256
    } else {
        Write-Warning "Source file not found: $srcFile"
    }
}

# Copy logo.png from installer icon
Copy-Item (Join-Path $webIconsDir "installer.png") (Join-Path $webIconsDir "logo.png") -Force

# Generate rsrc.syso for each cmd folder
$rsrcExe = "D:\go_workspace\bin\rsrc.exe"

$moduleMap = @{
    "cmd\orzu-kassa"     = "kassa.ico"
    "cmd\orzu-server"    = "server.ico"
    "cmd\orzu-installer" = "installer.ico"
    "cmd\orzu-backup"    = "backup.ico"
    "cmd\orzu-db"        = "db.ico"
}

foreach ($modPath in $moduleMap.Keys) {
    $icoName = $moduleMap[$modPath]
    $icoFullPath = Join-Path $assetsIconsDir $icoName
    $targetSyso = Join-Path (Join-Path $baseDir $modPath) "rsrc.syso"

    Write-Host "Generating syso for $modPath using $icoName..."
    & $rsrcExe -ico $icoFullPath -o $targetSyso
}

Write-Host "Icon generation and syso bundling complete!"
