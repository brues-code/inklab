# release.ps1 — prepare (and optionally publish) an InkLab release.
#
# Releases are built by .github/workflows/release.yml on a `v*` tag push, and
# `wails build` embeds whatever data/inklab.db is committed at that tag. Your
# day-to-day scrapes are NOT committed, so this script bakes them into the
# release: it promotes your locally-scraped ('local') rows to 'official', bumps
# the embedded DB revision (so users' DBs refresh-merge on next launch), commits
# the promoted DB + version bump, and tags the release. Your working DB is then
# restored to its 'local'-tagged state so your own RebuildSpawnZones never wipes
# your object scrapes.
#
# Usage:
#   pwsh scripts/release.ps1 -Version v0.6.3            # prepare: commit + tag locally
#   pwsh scripts/release.ps1 -Version v0.6.3 -Push      # also push -> triggers CI release
#
# Prerequisite: close `wails dev` first (it locks data/inklab.db).

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^v\d+\.\d+\.\d+$')]
    [string]$Version,

    [switch]$Push
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot   # repo root (scripts/..)
Set-Location $root

$dbPath = Join-Path $root 'data/inklab.db'
$embedFile = Join-Path $root 'embedded_data.go'

if (-not (Test-Path $dbPath)) { throw "data/inklab.db not found at $dbPath" }

# Fail fast if the DB is locked (wails dev open) — promote would silently skip.
try {
    $fs = [System.IO.File]::Open($dbPath, 'Open', 'ReadWrite', 'None')
    $fs.Close()
} catch {
    throw "data/inklab.db is locked. Close 'wails dev' (and any DB viewer) and retry."
}

# 1. Bump embeddedDBVersion so users' extracted DBs refresh-merge to this build.
$embed = Get-Content $embedFile -Raw
$m = [regex]::Match($embed, 'const embeddedDBVersion = (\d+)')
if (-not $m.Success) { throw "Could not find 'const embeddedDBVersion = N' in embedded_data.go" }
$oldVer = [int]$m.Groups[1].Value
$newVer = $oldVer + 1
$embed = [regex]::Replace($embed, 'const embeddedDBVersion = \d+', "const embeddedDBVersion = $newVer")
Set-Content $embedFile $embed -NoNewline -Encoding utf8
Write-Host "Bumped embeddedDBVersion: $oldVer -> $newVer"

# 2. Snapshot the working (local-tagged) DB, then promote local -> official.
$backup = "$dbPath.relbak"
Copy-Item $dbPath $backup -Force
try {
    Write-Host "Promoting local scrapes to official..."
    & go run ./cmd/promotedb data
    if ($LASTEXITCODE -ne 0) { throw "promotedb failed" }

    # 3. Commit the promoted DB + version bump.
    & git add embedded_data.go data/inklab.db
    if ($LASTEXITCODE -ne 0) { throw "git add failed" }
    & git commit -m "release: $Version (embedded data v$newVer)"
    if ($LASTEXITCODE -ne 0) { throw "git commit failed" }

    # 4. Tag the release.
    & git tag $Version
    if ($LASTEXITCODE -ne 0) { throw "git tag failed" }
}
finally {
    # 5. Restore the working DB to its local-tagged state regardless of outcome,
    #    so daily rebuilds keep protecting your scrapes. (The promoted blob is
    #    already captured in the commit.)
    Move-Item $backup $dbPath -Force
    Write-Host "Restored working data/inklab.db (local provenance)."
}

if ($Push) {
    Write-Host "Pushing commit + tag $Version (triggers CI release)..."
    & git push origin HEAD
    if ($LASTEXITCODE -ne 0) { throw "git push failed" }
    & git push origin $Version
    if ($LASTEXITCODE -ne 0) { throw "git push tag failed" }
    Write-Host "✓ Pushed. Watch the Release workflow on GitHub."
} else {
    Write-Host ""
    Write-Host "✓ Prepared $Version locally (data v$newVer). To publish, run:"
    Write-Host "    git push origin HEAD && git push origin $Version"
    Write-Host "  or re-run with -Push."
}
