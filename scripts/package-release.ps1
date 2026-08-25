param(
    [Parameter(Mandatory = $true)]
    [string]$Version,

    [string]$ExecutablePath = "dist/kcd2-dual-subtitles.exe",

    [string]$OutputDirectory = "dist"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ($Version -notmatch '^[A-Za-z0-9][A-Za-z0-9.-]*$') {
    throw "Invalid package version: '$Version'"
}

if (-not (Test-Path -LiteralPath $ExecutablePath -PathType Leaf)) {
    throw "Executable not found: $ExecutablePath"
}

foreach ($requiredFile in @("README.md", "LICENSE", "NEXUS_DESCRIPTION.bbcode.txt")) {
    if (-not (Test-Path -LiteralPath $requiredFile -PathType Leaf)) {
        throw "$requiredFile not found in repository root"
    }
}

& (Join-Path $PSScriptRoot "render-nexus-description.ps1") -Check

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null

$archiveName = "kcd2-dual-subtitles-$Version-windows-x64.zip"
$archivePath = Join-Path $OutputDirectory $archiveName
$packageDirectory = Join-Path $OutputDirectory "package-$Version"
$validationDirectory = Join-Path $OutputDirectory "validate-$Version"
$releaseChecksumsPath = Join-Path $OutputDirectory "SHA256SUMS.txt"

Remove-Item -LiteralPath $archivePath -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $packageDirectory -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $validationDirectory -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $releaseChecksumsPath -Force -ErrorAction SilentlyContinue

try {
    New-Item -ItemType Directory -Path $packageDirectory | Out-Null

    Copy-Item -LiteralPath $ExecutablePath -Destination (Join-Path $packageDirectory "kcd2-dual-subtitles.exe")
    Copy-Item -LiteralPath "README.md" -Destination (Join-Path $packageDirectory "README.md")
    Copy-Item -LiteralPath "LICENSE" -Destination (Join-Path $packageDirectory "LICENSE")

    $packageFiles = @(
        "kcd2-dual-subtitles.exe",
        "README.md",
        "LICENSE"
    )

    $packageChecksumLines = foreach ($name in $packageFiles) {
        $path = Join-Path $packageDirectory $name
        $hash = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
        "$hash  $name"
    }
    $packageChecksumLines | Set-Content -LiteralPath (Join-Path $packageDirectory "SHA256SUMS.txt") -Encoding ascii

    Compress-Archive -Path (Join-Path $packageDirectory "*") -DestinationPath $archivePath -CompressionLevel Optimal

    New-Item -ItemType Directory -Path $validationDirectory | Out-Null
    Expand-Archive -LiteralPath $archivePath -DestinationPath $validationDirectory

    $expectedEntries = @(
        "kcd2-dual-subtitles.exe",
        "README.md",
        "LICENSE",
        "SHA256SUMS.txt"
    ) | Sort-Object

    $actualEntries = Get-ChildItem -LiteralPath $validationDirectory -File -Recurse |
        ForEach-Object { [IO.Path]::GetRelativePath($validationDirectory, $_.FullName).Replace('\', '/') } |
        Sort-Object

    $entryDifference = Compare-Object -ReferenceObject $expectedEntries -DifferenceObject $actualEntries
    if ($entryDifference) {
        throw "Release archive contains an unexpected file set: $($actualEntries -join ', ')"
    }

    $directories = Get-ChildItem -LiteralPath $validationDirectory -Directory -Recurse
    if ($directories) {
        throw "Release archive contains unexpected directories"
    }

    $extractedChecksumLines = Get-Content -LiteralPath (Join-Path $validationDirectory "SHA256SUMS.txt")
    $expectedExtractedChecksumLines = foreach ($name in $packageFiles) {
        $path = Join-Path $validationDirectory $name
        $hash = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
        "$hash  $name"
    }
    if (($extractedChecksumLines -join "`n") -ne ($expectedExtractedChecksumLines -join "`n")) {
        throw "Release archive checksum validation failed"
    }

    $releaseAssets = @(
        @{ Path = $ExecutablePath; Name = "kcd2-dual-subtitles.exe" },
        @{ Path = $archivePath; Name = $archiveName }
    )

    $releaseChecksumLines = foreach ($asset in $releaseAssets) {
        $hash = (Get-FileHash -LiteralPath $asset.Path -Algorithm SHA256).Hash.ToLowerInvariant()
        "$hash  $($asset.Name)"
    }
    $releaseChecksumLines | Set-Content -LiteralPath $releaseChecksumsPath -Encoding ascii

    Write-Host "Created $archivePath"
    Write-Host "Created $releaseChecksumsPath"
}
finally {
    Remove-Item -LiteralPath $packageDirectory -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $validationDirectory -Recurse -Force -ErrorAction SilentlyContinue
}
