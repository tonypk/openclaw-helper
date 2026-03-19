#!/bin/bash
# OpenClaw Helper - OpenClaw Verify (fallback)
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"

if command -v openclaw &>/dev/null; then
    echo "##OCH:VERIFY:PASS"
else
    echo "##OCH:VERIFY:FAIL:openclaw command not found"
    exit 1
fi
