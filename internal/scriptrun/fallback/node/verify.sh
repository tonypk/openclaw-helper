#!/bin/bash
# OpenClaw Helper - Node.js Verify (fallback)
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"

NODE_VER=$(node -v 2>/dev/null || echo "none")
MAJOR=$(echo "$NODE_VER" | sed 's/^v//' | cut -d. -f1)

if [ "$MAJOR" -ge 22 ] 2>/dev/null; then
    echo "##OCH:VERIFY:PASS"
else
    echo "##OCH:VERIFY:FAIL:Node.js $NODE_VER (need >= 22)"
    exit 1
fi
