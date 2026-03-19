# OpenClaw Helper - WSL2 Verify (fallback, PowerShell)
$wslInfo = wsl.exe --status 2>&1
$wslVer = 0
if ($wslInfo -match "Default Version:\s*2|WSL version:\s*2") {
    $wslVer = 2
}

$distros = wsl.exe --list --quiet 2>&1
$hasUbuntu = $distros -match "Ubuntu"

if ($wslVer -ge 2 -and $hasUbuntu) {
    Write-Output "##OCH:VERIFY:PASS"
} else {
    Write-Output "##OCH:VERIFY:FAIL:WSL version=$wslVer, Ubuntu=$hasUbuntu"
    exit 1
}
