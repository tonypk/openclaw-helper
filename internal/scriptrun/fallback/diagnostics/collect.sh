#!/bin/bash
# OpenClaw Helper - Diagnostic Data Collector (fallback)
# Collects environment information for troubleshooting.

echo "##OCH:DETAIL:Collecting diagnostic data..."

# OS info
OS_INFO=$(cat /etc/os-release 2>/dev/null | head -5 || echo "unknown")
echo "##OCH:DIAG:{\"key\":\"os_info\",\"value\":\"$(echo "$OS_INFO" | tr '\n' ' ')\"}"

# Memory
MEM=$(free -h 2>/dev/null | grep Mem || echo "unknown")
echo "##OCH:DIAG:{\"key\":\"memory\",\"value\":\"$MEM\"}"

# Disk
DISK=$(df -h / 2>/dev/null | tail -1 || echo "unknown")
echo "##OCH:DIAG:{\"key\":\"disk\",\"value\":\"$DISK\"}"

# Node version
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"
NODE_VER=$(node -v 2>/dev/null || echo "not installed")
echo "##OCH:DIAG:{\"key\":\"node_version\",\"value\":\"$NODE_VER\"}"

# npm version
NPM_VER=$(npm -v 2>/dev/null || echo "not installed")
echo "##OCH:DIAG:{\"key\":\"npm_version\",\"value\":\"$NPM_VER\"}"

# OpenClaw
OC_PATH=$(which openclaw 2>/dev/null || echo "not found")
echo "##OCH:DIAG:{\"key\":\"openclaw_path\",\"value\":\"$OC_PATH\"}"

# Recent openclaw log
OC_LOG=$(tail -30 /tmp/openclaw.log 2>/dev/null || echo "no log")
echo "##OCH:DIAG:{\"key\":\"openclaw_log\",\"value\":\"$(echo "$OC_LOG" | tr '\n' '|')\"}"

echo "##OCH:PROGRESS:100:Diagnostic data collected"
