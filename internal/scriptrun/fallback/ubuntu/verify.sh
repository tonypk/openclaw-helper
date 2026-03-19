#!/bin/bash
# OpenClaw Helper - Ubuntu Verify (fallback)
if grep -qc Ubuntu /etc/os-release 2>/dev/null; then
    ##OCH:VERIFY:PASS
else
    ##OCH:VERIFY:FAIL:Ubuntu not found
    exit 1
fi
