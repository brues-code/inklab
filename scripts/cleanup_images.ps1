$dbPath = "data\inklab.db"
$imgDir = "data\npc_images"

# Helper to get the hash name
function Get-HashPath($url) {
    if ([string]::IsNullOrEmpty($url)) { return $null }
    $md5 = [System.Security.Cryptography.MD5]::Create()
    $hashBytes = $md5.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($url))
    $hashString = [System.BitConverter]::ToString($hashBytes).Replace("-", "").ToLower()
    
    $ext = ".jpg"
    if ($url -like "* .png*") { $ext = ".png" }
    elseif ($url -like "* .gif*") { $ext = ".gif" }
    elseif ($url -like "* .webp*") { $ext = ".webp" }
    
    return Join-Path $imgDir ($hashString + $ext)
}

Write-Host "Fetching metadata from database..."
# Get data from sqlite. Output as CSV or space-separated.
# We use -separator "|" to be safe with names.
$query = "SELECT entry, map_url, model_image_url, map_image_local, model_image_local FROM creature_metadata;"
$results = sqlite3 $dbPath -separator "|" $query

Write-Host "Processing $($results.Count) records..."

foreach ($line in $results) {
    $parts = $line.Split("|")
    if ($parts.Length -lt 5) { continue }
    
    $entry = $parts[0]
    $mapUrl = $parts[1]
    $modelUrl = $parts[2]
    $mapLocal = $parts[3]
    $modelLocal = $parts[4]
    
    $newMapLocal = $mapLocal
    $newModelLocal = $modelLocal

    # Process Map
    if (-not [string]::IsNullOrEmpty($mapUrl)) {
        $expected = Get-HashPath $mapUrl
        if (-not [string]::IsNullOrEmpty($mapLocal) -and $mapLocal -ne $expected) {
            if (Test-Path -LiteralPath $mapLocal) {
                if (Test-Path -LiteralPath $expected) {
                    Remove-Item -LiteralPath $mapLocal -Force
                }
                else {
                    Move-Item -LiteralPath $mapLocal -Destination $expected -Force
                }
                $newMapLocal = $expected
            }
        }
    }

    # Process Model
    if (-not [string]::IsNullOrEmpty($modelUrl)) {
        $expected = Get-HashPath $modelUrl
        if (-not [string]::IsNullOrEmpty($modelLocal) -and $modelLocal -ne $expected) {
            if (Test-Path -LiteralPath $modelLocal) {
                if (Test-Path -LiteralPath $expected) {
                    Remove-Item -LiteralPath $modelLocal -Force
                }
                else {
                    Move-Item -LiteralPath $modelLocal -Destination $expected -Force
                }
                $newModelLocal = $expected
            }
        }
    }

    if ($newMapLocal -ne $mapLocal -or $newModelLocal -ne $modelLocal) {
        # Update DB
        sqlite3 $dbPath "UPDATE creature_metadata SET map_image_local = '$newMapLocal', model_image_local = '$newModelLocal' WHERE entry = $entry;"
    }
}

Write-Host "Cleaning up orphaned files..."
Get-ChildItem $imgDir -Filter "map_*" | Remove-Item -Force
Get-ChildItem $imgDir -Filter "model_*" | Remove-Item -Force

Write-Host "Deduplication complete!"
