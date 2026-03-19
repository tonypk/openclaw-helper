# OpenClaw Helper - WSL2 Installation (fallback, PowerShell)
$ErrorActionPreference = "Stop"

Write-Output "##OCH:PROGRESS:10:Checking WSL2 status..."

# Check current WSL status
$wslInfo = wsl.exe --status 2>&1
$wslVer = 0
if ($wslInfo -match "Default Version:\s*2|WSL version:\s*2") {
    $wslVer = 2
}

if ($wslVer -lt 2) {
    Write-Output "##OCH:PROGRESS:20:Installing WSL2 (this may take a few minutes)..."
    try {
        wsl.exe --install --no-launch 2>&1
    } catch {
        $msg = $_.Exception.Message
        if ($msg -match "restart|reboot") {
            Write-Output "##OCH:REBOOT:WSL2 installed - reboot required"
            exit 0
        }
        Write-Output "##OCH:ERROR:WSL2 installation failed: $msg"
        exit 1
    }
}

# Check if Ubuntu is installed
$distros = wsl.exe --list --quiet 2>&1
$hasUbuntu = $distros -match "Ubuntu"

if (-not $hasUbuntu) {
    Write-Output "##OCH:PROGRESS:60:Installing Ubuntu..."
    try {
        wsl.exe --install -d Ubuntu --no-launch 2>&1
    } catch {
        $msg = $_.Exception.Message
        if ($msg -match "restart|reboot") {
            Write-Output "##OCH:REBOOT:Ubuntu installed - reboot required"
            exit 0
        }
        Write-Output "##OCH:ERROR:Ubuntu installation failed: $msg"
        exit 1
    }
}

Write-Output "##OCH:PROGRESS:100:WSL2 + Ubuntu ready"
