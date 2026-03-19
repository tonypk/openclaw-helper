#!/bin/bash
# OpenClaw Helper - Final Verification (fallback)
set -e

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"

##OCH:PROGRESS:20:Starting OpenClaw gateway...
nohup openclaw start > /tmp/openclaw.log 2>&1 &

##OCH:PROGRESS:50:Waiting for gateway to start...

MAX_RETRIES=15
RETRY_INTERVAL=2
for i in $(seq 1 $MAX_RETRIES); do
    if bash -c "echo >/dev/tcp/127.0.0.1/18789" 2>/dev/null; then
        ##OCH:PROGRESS:100:OpenClaw gateway is running!
        exit 0
    fi
    sleep $RETRY_INTERVAL
done

##OCH:ERROR:Gateway did not start within 30 seconds — check /tmp/openclaw.log in WSL
##OCH:DIAG:{"key":"openclaw_log","value":"$(tail -20 /tmp/openclaw.log 2>/dev/null || echo 'no log')"}
exit 1
