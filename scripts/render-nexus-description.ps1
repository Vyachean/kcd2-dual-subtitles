param(
    [string]$ReadmePath = "README.md",
    [string]$OutputPath = "NEXUS_DESCRIPTION.bbcode.txt",
    [switch]$Check
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $ReadmePath -PathType Leaf)) {
    throw "README source not found: $ReadmePath"
}

$startMarker = "<!-- nexus-description:start -->"
$endMarker = "<!-- nexus-description:end -->"
$lines = @(Get-Content -LiteralPath $ReadmePath)
$startIndex = [Array]::IndexOf($lines, $startMarker)
$endIndex = [Array]::IndexOf($lines, $endMarker)

if ($startIndex -lt 0 -or $endIndex -lt 0 -or $endIndex -le $startIndex + 1) {
    throw "README must contain one non-empty nexus-description:start/end section"
}

$output = [Collections.Generic.List[string]]::new()
$inCodeBlock = $false

for ($index = $startIndex + 1; $index -lt $endIndex; $index++) {
    $line = $lines[$index]

    if ($line -match '^```') {
        if ($inCodeBlock) {
            $output.Add("[/code]")
        }
        else {
            $output.Add("[code]")
        }
        $inCodeBlock = -not $inCodeBlock
        continue
    }

    if (-not $inCodeBlock) {
        if ($line -match '^## (.+)$') {
            $line = "[b]$($Matches[1])[/b]"
        }

        $line = [regex]::Replace($line, '\*\*(.+?)\*\*', '[b]$1[/b]')
        $line = [regex]::Replace($line, '`([^`]+)`', '$1')
        $line = [regex]::Replace($line, '<(https?://[^>]+)>', '$1')
    }

    $output.Add($line)
}

if ($inCodeBlock) {
    throw "Unclosed fenced code block in Nexus README section"
}

$rendered = (($output -join "`n").Trim() + "`n")

if ($Check) {
    if (-not (Test-Path -LiteralPath $OutputPath -PathType Leaf)) {
        throw "Generated Nexus description is missing: $OutputPath"
    }

    $current = (Get-Content -LiteralPath $OutputPath -Raw) -replace "`r`n", "`n"
    if ($current -ne $rendered) {
        throw "$OutputPath is stale; run scripts/render-nexus-description.ps1 and commit the result"
    }

    Write-Host "$OutputPath is up to date"
    exit 0
}

$parent = Split-Path -Parent $OutputPath
if ($parent) {
    New-Item -ItemType Directory -Force -Path $parent | Out-Null
}

$rendered | Set-Content -LiteralPath $OutputPath -Encoding utf8 -NoNewline
Write-Host "Created $OutputPath"
