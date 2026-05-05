# Mavrog Updater

A small Wails (Go + WebView2) desktop app that keeps the **MavrogBattleCry**
World of Warcraft addon up to date directly from GitHub Releases.

## Features

- Auto-detects the WoW Retail `_retail_/Interface/AddOns` folder (and asks
  you to pick it manually if not found).
- Reads the installed `## Version:` from the `.toc` file.
- Polls `https://api.github.com/repos/Mavrag/MavrogBattleCry-Release/releases/latest`,
  downloads the `.zip` asset, and extracts it into AddOns.
- Auto-checks for updates on launch (toggleable).
- Built-in changelog viewer (release notes from GitHub).
- Self-updates: checks its own GitHub release repo and swaps the running `.exe`.

## Project layout

```
mavrog-updater/
├── app.go                  # bound API surface (frontend ↔ backend)
├── addon.go                # AddOns folder detection, .toc parsing
├── github.go               # GitHub Releases API + asset selection
├── install.go              # download + zip extraction + progress events
├── selfupdate.go           # updater self-replacement logic (.exe -> .old swap)
├── config.go               # %APPDATA%\MavrogUpdater\config.json
├── main.go                 # Wails app entry point
├── frontend/               # Svelte + TS UI
└── release-repo-template/  # files to copy into the GitHub release repo
```

## Configuration constants

Edit `github.go` if you ever rename repos:

```go
const (
    AddonRepoOwner   = "Mavrag"
    AddonRepoName   = "MavrogBattleCry-Release"
    UpdaterRepoOwner = "Mavrag"
    UpdaterRepoName  = "mavrog-updater"
)
```

And `addon.go` for the addon folder name (must match the folder inside the zip
and inside `_retail_/Interface/AddOns`):

```go
const AddonFolderName = "MavrogBattleCry"
```

## Build

Prereqs: Go 1.23+, Node 20+, Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`).

Dev (hot-reload):
```powershell
wails dev
```

Production build:
```powershell
wails build -ldflags "-X main.AppVersion=v0.1.0"
# -> build\bin\MavrogUpdater.exe
```

## Release zip format (the addon repo)

The zip uploaded to `Mavrag/MavrogBattleCry-Release` releases must contain a
top-level folder matching `AddonFolderName`:

```
MavrogBattleCry-1.2.3.zip
└── MavrogBattleCry/
    ├── MavrogBattleCry.toc      # ## Version: 1.2.3
    ├── MavrogBattleCry.lua
    └── ...
```

The picker prefers an asset whose name starts with `MavrogBattleCry`; any
`.zip` is accepted as a fallback.

## Setting up the empty release repo

The repo `Mavrag/MavrogBattleCry-Release` is currently empty. Copy the contents
of [`release-repo-template/`](./release-repo-template) into it:

```
README.md
.github/workflows/release.yml
```

Then, in your **addon source** (where the `.lua` / `.toc` lives), tag and push:
```bash
git tag v0.1.0
git push origin v0.1.0
```

The included workflow zips the addon and creates a GitHub Release with the
asset attached. The updater will pick it up automatically.

If your addon source is in a different repo than the release repo, either:
- run the workflow in the source repo and point its `GITHUB_TOKEN` /
  `softprops/action-gh-release` at the release repo (use a PAT in `secrets`); or
- publish releases manually for now (Releases → Draft → upload the zip).

## How self-update works

`ApplySelfUpdate()`:
1. Downloads the new `MavrogUpdater.exe` to `MavrogUpdater.exe.new`.
2. Renames the running `.exe` → `.exe.old` (Windows allows rename of running
   executables, just not delete).
3. Renames `.new` → original path.
4. Spawns the new exe and exits.
5. On startup, the new process deletes the `.old` leftover.

For this to do anything, you must publish at least one Release on the updater
repo (`Mavrag/mavrog-updater`) with a `.exe` asset. The included
`release-repo-template/.github/workflows/updater-release.yml` does this when
you push a tag like `v0.1.0` to the updater repo.

## Storage

Config is persisted at:
```
%APPDATA%\MavrogUpdater\config.json
```
Keys: `addonsPath`, `autoCheck`, `lastVersion`.
