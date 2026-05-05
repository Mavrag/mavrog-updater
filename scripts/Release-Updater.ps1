<#
.SYNOPSIS
  End-to-end release script for Mavrog Updater.
  - Ensures GitHub repo exists (creates it if missing).
  - Ensures `origin` remote is wired up and main branch is pushed.
  - Kills any running MavrogUpdater.exe so the build can overwrite it.
  - Builds MavrogUpdater.exe with the version stamped in.
  - Tags the commit and creates a GitHub Release with the .exe attached.

.PARAMETER Version
  Version to release, e.g. "0.2.0" or "v0.2.0".

.PARAMETER Repo
  owner/name on GitHub. Default: Mavrag/mavrog-updater.

.PARAMETER Notes
  Optional release notes. If omitted, GitHub auto-generates from commits.

.PARAMETER SkipBuild
  Skip the wails build step (reuse existing build\bin\MavrogUpdater.exe).

.EXAMPLE
  .\scripts\Release-Updater.ps1 -Version 0.2.0

.NOTES
  Requires: git, gh (https://cli.github.com), Go, Node, wails CLI.
  Run `gh auth login` once before using.
#>
[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [string]$Version,

  [string]$Repo = "Mavrag/mavrog-updater",
  [string]$Notes = "",
  [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $ProjectRoot

# ---- Helpers ------------------------------------------------------------

function Fail($msg) { Write-Host "ERROR: $msg" -ForegroundColor Red; exit 1 }
function Info($msg) { Write-Host ">> $msg" -ForegroundColor Cyan }
function Ok  ($msg) { Write-Host "   $msg" -ForegroundColor DarkGray }

function Require-Cmd($name, $hint) {
  if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
    Fail "'$name' not found. $hint"
  }
}

# ---- Prereqs ------------------------------------------------------------

$env:Path = "C:\Program Files\Go\bin;$env:USERPROFILE\go\bin;$env:Path"

Require-Cmd git "Install Git for Windows."
Require-Cmd gh  "Install GitHub CLI from https://cli.github.com then run 'gh auth login'."
if (-not $SkipBuild) {
  Require-Cmd go    "Install Go 1.23+."
  Require-Cmd wails "Run: go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0"
  Require-Cmd npm   "Install Node 20+."
}

# Auth check
gh auth status *> $null
if ($LASTEXITCODE -ne 0) { Fail "gh is not authenticated. Run: gh auth login" }

# ---- Normalise version -------------------------------------------------

$vPlain = $Version.TrimStart('v','V')
if ($vPlain -notmatch '^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$') {
  Fail "Version must look like 1.2.3 or v1.2.3 (got '$Version')."
}
$tag = "v$vPlain"
Info "Releasing $tag to $Repo"

# ---- Git working tree sanity -------------------------------------------

if (-not (Test-Path ".git")) { Fail "Not a git repo. Run this from the project root." }

$dirty = (git status --porcelain)
if ($dirty) {
  Write-Host "   Uncommitted changes detected:" -ForegroundColor Yellow
  git status --short
  $ans = Read-Host "Continue anyway? (y/N)"
  if ($ans -ne 'y') { Fail "Aborted." }
}

# ---- Ensure GitHub repo exists -----------------------------------------

Info "Checking GitHub repo $Repo ..."
$exists = $true
gh repo view $Repo *> $null
if ($LASTEXITCODE -ne 0) { $exists = $false }

if (-not $exists) {
  Info "Repo not found. Creating $Repo (public) ..."
  gh repo create $Repo --public --confirm *> $null
  if ($LASTEXITCODE -ne 0) { Fail "Failed to create repo $Repo." }
  Ok "Created."
} else {
  Ok "Repo exists."
}

# ---- Ensure origin remote points at the repo ---------------------------

$expectedUrl = "https://github.com/$Repo.git"
$currentUrl = (git remote get-url origin 2>$null)
if (-not $currentUrl) {
  Info "Adding origin remote -> $expectedUrl"
  git remote add origin $expectedUrl | Out-Null
} elseif ($currentUrl -ne $expectedUrl -and $currentUrl -ne ($expectedUrl -replace '\.git$','')) {
  Write-Host "   origin is '$currentUrl', expected '$expectedUrl'" -ForegroundColor Yellow
  $ans = Read-Host "Update origin to $expectedUrl ? (y/N)"
  if ($ans -eq 'y') { git remote set-url origin $expectedUrl | Out-Null }
}

# ---- Ensure branch is pushed with upstream -----------------------------

$branch = (git rev-parse --abbrev-ref HEAD).Trim()
if ($branch -eq "HEAD") { Fail "Detached HEAD. Checkout a branch first." }

git rev-parse --abbrev-ref --symbolic-full-name "@{u}" 2>$null | Out-Null
if ($LASTEXITCODE -ne 0) {
  Info "Branch '$branch' has no upstream. Pushing with -u ..."
  git push -u origin $branch
  if ($LASTEXITCODE -ne 0) { Fail "git push failed." }
} else {
  Info "Pushing latest commits on $branch ..."
  git push
  if ($LASTEXITCODE -ne 0) { Fail "git push failed." }
}

# ---- Kill running exe so build can overwrite it ------------------------

$running = Get-Process -Name "MavrogUpdater" -ErrorAction SilentlyContinue
if ($running) {
  Info "Stopping running MavrogUpdater.exe ..."
  $running | Stop-Process -Force
  Start-Sleep -Milliseconds 500
}

# ---- Build -------------------------------------------------------------

$exe = Join-Path $ProjectRoot "build\bin\MavrogUpdater.exe"

if (-not $SkipBuild) {
  Info "Building with AppVersion=$tag ..."
  wails build -clean -trimpath -ldflags "-s -w -X main.AppVersion=$tag"
  if ($LASTEXITCODE -ne 0) { Fail "wails build failed." }
}

if (-not (Test-Path $exe)) { Fail "Build output not found: $exe" }
$sizeMB = [math]::Round((Get-Item $exe).Length / 1MB, 2)
Ok "Built: $exe ($sizeMB MB)"

# ---- Tag ---------------------------------------------------------------

$tagExists = (git tag --list $tag)
if ($tagExists) {
  Write-Host "   Local tag $tag already exists." -ForegroundColor Yellow
} else {
  Info "Tagging $tag ..."
  git tag -a $tag -m "Release $tag"
}

Info "Pushing tag $tag ..."
git push origin $tag
if ($LASTEXITCODE -ne 0) {
  # Tag might already be on remote.
  Write-Host "   (tag may already be pushed; continuing)" -ForegroundColor DarkGray
}

# ---- Release -----------------------------------------------------------

gh release view $tag --repo $Repo *> $null
$releaseExists = ($LASTEXITCODE -eq 0)

if ($releaseExists) {
  Info "Release $tag already exists. Uploading asset (clobber)..."
  gh release upload $tag $exe --repo $Repo --clobber
} else {
  Info "Creating GitHub Release $tag ..."
  $ghArgs = @(
    "release","create",$tag,$exe,
    "--repo",$Repo,
    "--title","Mavrog Updater $tag"
  )
  if (-not $Notes) {
    # Build default notes from commits since last tag
    $prevTag = (git tag --sort=-version:refname | Select-Object -Skip 1 -First 1)
    if ($prevTag) {
      $commits = @(git log "$prevTag..HEAD" --pretty=format:"- %s" --no-merges)
    } else {
      $commits = @(git log --pretty=format:"- %s" --no-merges)
    }
    $commitsText = ($commits -join "`n")
    $changelogUrl = if ($prevTag) { "https://github.com/$Repo/compare/$prevTag...$tag" } else { "" }
    $Notes = "## What's Changed`n`n$commitsText"
    if ($changelogUrl) { $Notes += "`n`n**Full Changelog**: $changelogUrl" }
  }
  $tmp = New-TemporaryFile
  Set-Content -LiteralPath $tmp -Value $Notes -Encoding UTF8
  $ghArgs += @("--notes-file",$tmp)
  gh @ghArgs
  if ($LASTEXITCODE -ne 0) { Fail "gh release create failed." }
}

Write-Host ""
Write-Host "Done. Release is live: https://github.com/$Repo/releases/tag/$tag" -ForegroundColor Green
