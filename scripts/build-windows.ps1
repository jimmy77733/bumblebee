$ErrorActionPreference = "Stop"
$Repo = Split-Path -Parent $PSScriptRoot
Set-Location $Repo
$env:CGO_ENABLED = "1"

$Out = Join-Path $Repo "dist\windows"
New-Item -ItemType Directory -Force -Path $Out | Out-Null

go build -trimpath -ldflags "-s -w -H windowsgui" -o (Join-Path $Out "Bumblebee.exe") .\cmd\bumblebee-ui
go build -trimpath -ldflags "-s -w" -o (Join-Path $Out "bumblebee-cli.exe") .\cmd\bumblebee

$IntelOut = Join-Path $Out "threat_intel"
if (Test-Path $IntelOut) {
    Remove-Item -Recurse -Force $IntelOut
}
Copy-Item -Recurse -Force (Join-Path $Repo "threat_intel") $IntelOut

$sha = (git -C $Repo log -1 --format=%H -- threat_intel).Trim()
Set-Content -Path (Join-Path $IntelOut ".upstream-revision") -Value $sha -Encoding ascii

Write-Host "Built $Out"
