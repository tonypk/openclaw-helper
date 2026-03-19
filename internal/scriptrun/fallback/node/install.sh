#!/bin/bash
# OpenClaw Helper - Node.js Installation (fallback)
set -e

##OCH:PROGRESS:10:Installing nvm...
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"

##OCH:PROGRESS:40:Installing Node.js 22 LTS...
nvm install 22
nvm alias default 22

##OCH:PROGRESS:80:Verifying Node.js installation...
NODE_VER=$(node -v 2>/dev/null || echo "none")
if [[ "$NODE_VER" == v22* ]] || [[ "$NODE_VER" == v2[3-9]* ]]; then
    ##OCH:PROGRESS:100:Node.js $NODE_VER installed
else
    ##OCH:ERROR:Node.js installation verification failed (got $NODE_VER)
    exit 1
fi
