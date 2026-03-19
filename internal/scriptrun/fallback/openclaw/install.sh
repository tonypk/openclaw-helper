#!/bin/bash
# OpenClaw Helper - OpenClaw Installation (fallback)
set -e

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"

##OCH:PROGRESS:10:Installing OpenClaw via npm...
npm install -g openclaw

##OCH:PROGRESS:70:Verifying OpenClaw installation...
if command -v openclaw &>/dev/null; then
    ##OCH:PROGRESS:100:OpenClaw installed successfully
else
    ##OCH:ERROR:OpenClaw installation could not be verified
    exit 1
fi
