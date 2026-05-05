<#
.SYNOPSIS
  Publishes one or all MavrogBattlecry zips as GitHub Releases on the
  Mavrag/MavrogBattleCry-Release repo, pulling release notes from CHANGELOG.md.

.PARAMETER Version
  A version string like "1.6.2" or "v1.6.2". Use "all" to publish every zip
  found in -ReleasesDir that doesn't already have a matching release.

.PARAMETER ReleasesDir
  Folder containing MavrogBattlecry-vX.Y.Z.zip files.
  Default: D:\Projects\mavrog-battlecry\Releases

.PARAMETER ChangelogFile
  Path to CHANGELOG.md. Default: D:\Projects\mavrog-battlecry\Docs\CHANGELOG.md

.PARAMETER Repo
  Target release repository. Default: Mavrag/MavrogBattleCry-Release

.EXAMPLE
  .\Publish-Release.ps1 -Version 1.6.2

.EXAMPLE
  .\Publish-Release.ps1 -Version all

.NOTES
  Requires GitHub CLI: https://cli.github.com  (run `gh auth login` once).
#>
[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [string]$Version,

  [string]$ReleasesDir = "D:\Projects\mavrog-battlecry\Releases",
  [string]$ChangelogFile = "D:\Projects\mavrog-battlecry\Docs\CHANGELOG.md",
  [string]$Repo = "Mavrag/MavrogBattleCry-Release"
)

$ErrorActionPreference = "Stop"

function Test-Gh {
  if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
    throw "GitHub CLI ('gh') not found. Install from https://cli.github.com then run 'gh auth login'."
  }
}

function Get-ChangelogSection {
  param([string]$Path, [string]$VersionNumber)
  if (-not (Test-Path $Path)) { return "" }
  $lines = Get-Content -LiteralPath $Path -Encoding UTF8
  $start = -1
  $end   = $lines.Count
  for ($i = 0; $i -lt $lines.Count; $i++) {
    if ($lines[$i] -match "^##\s+$([regex]::Escape($VersionNumber))\b") {
      $start = $i
      break
    }
  }
  if ($start -lt 0) { return "" }
  for ($j = $start + 1; $j -lt $lines.Count; $j++) {
    if ($lines[$j] -match "^##\s+\d") { $end = $j; break }
  }
  ($lines[($start+1)..($end-1)] -join "`n").Trim()
}

function Publish-One {
  param([string]$VersionNumber)

  $tag      = "v$VersionNumber"
  $zipName  = "MavrogBattlecry-$tag.zip"
  $zipPath  = Join-Path $ReleasesDir $zipName
  if (-not (Test-Path $zipPath)) {
    Write-Warning "Skipping $tag : $zipPath not found."
    return
  }

  # Skip if release already exists.
  $exists = $true
  try {
    gh release view $tag --repo $Repo *> $null
  } catch { $exists = $false }
  if ($LASTEXITCODE -ne 0) { $exists = $false }
  if ($exists) {
    Write-Host "  [skip] $tag already exists on $Repo" -ForegroundColor DarkGray
    return
  }

  $notes = Get-ChangelogSection -Path $ChangelogFile -VersionNumber $VersionNumber
  if (-not $notes) { $notes = "Release $tag" }

  $tmp = New-TemporaryFile
  Set-Content -LiteralPath $tmp -Value $notes -Encoding UTF8

  Write-Host "Publishing $tag ..." -ForegroundColor Cyan
  gh release create $tag $zipPath `
    --repo  $Repo `
    --title "MavrogBattlecry $tag" `
    --notes-file $tmp
  Remove-Item $tmp -Force
  if ($LASTEXITCODE -ne 0) { throw "gh release create failed for $tag" }
}

Test-Gh

if ($Version -ieq "all") {
  $zips = Get-ChildItem $ReleasesDir -Filter "MavrogBattlecry-v*.zip" |
          Sort-Object {
            $m = [regex]::Match($_.Name, "v(\d+)\.(\d+)\.(\d+)")
            if ($m.Success) { [version]"$($m.Groups[1]).$($m.Groups[2]).$($m.Groups[3])" } else { [version]"0.0.0" }
          }
  foreach ($z in $zips) {
    $m = [regex]::Match($z.Name, "v(\d+\.\d+\.\d+)")
    if ($m.Success) { Publish-One -VersionNumber $m.Groups[1].Value }
  }
} else {
  $v = $Version.TrimStart('v')
  Publish-One -VersionNumber $v
}

Write-Host "Done." -ForegroundColor Green
