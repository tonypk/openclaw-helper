#!/bin/bash
# OpenClaw Helper - Gateway Verify (fallback)
if bash -c "echo >/dev/tcp/127.0.0.1/18789" 2>/dev/null; then
    echo "##OCH:VERIFY:PASS"
else
    echo "##OCH:VERIFY:FAIL:Gateway not listening on port 18789"
    exit 1
fi
