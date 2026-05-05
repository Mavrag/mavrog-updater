# MavrogBattleCry-Release

This repository hosts releases of the **MavrogBattleCry** WoW addon.
The Mavrog Updater desktop app reads from this repo's GitHub Releases.

## How releases work

The updater queries:
```
GET https://api.github.com/repos/Mavrag/MavrogBattleCry-Release/releases/latest
```
and downloads the first `.zip` asset whose name starts with `MavrogBattleCry`.

The zip **must contain a top-level folder named `MavrogBattleCry/`** with the
addon files (including a `.toc` whose `## Version:` line matches the release
tag, e.g. `## Version: 1.2.3`).

## Layout that the updater expects

```
MavrogBattleCry-1.2.3.zip
└── MavrogBattleCry/
    ├── MavrogBattleCry.toc        # ## Version: 1.2.3
    ├── MavrogBattleCry.lua
    └── ...
```

## Publishing a new release (manual)

1. Zip your addon so the structure matches above.
2. Name the zip `MavrogBattleCry-<version>.zip`.
3. On GitHub: **Releases → Draft a new release**.
4. Tag = `v<version>` (e.g. `v1.2.3`).
5. Upload the zip as an asset and publish.

## Publishing automatically

Two workflows are provided in `.github/workflows/`:

- **`release.yml`** — triggers when you push a tag like `v1.2.3`. It zips the
  addon source and creates a Release with the asset attached.
  Place this workflow in the addon **source** repo (where the addon code lives).

- **`mirror-release.yml`** — copies the release zip into this release-only repo.
  Use this if your addon source is in a private repo and this repo is only for
  publishing public releases.

If your source already lives in this same repo, `release.yml` is enough.
