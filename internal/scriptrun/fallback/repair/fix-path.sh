#!/bin/bash
set -e

echo "##OCH:HEAL:REPAIR:fix-path.sh=running"

# Rebuild PATH with known locations
PATHS_TO_ADD=(
    "$HOME/.nvm/versions/node/$(ls -1 $HOME/.nvm/versions/node/ 2>/dev/null | tail -1)/bin"
    "$HOME/.local/bin"
    "/usr/local/bin"
)

BASHRC="$HOME/.bashrc"

for p in "${PATHS_TO_ADD[@]}"; do
    if [ -d "$p" ] && ! echo "$PATH" | grep -q "$p"; then
        echo "export PATH=\"$p:\$PATH\"" >> "$BASHRC"
        export PATH="$p:$PATH"
        echo "Added $p to PATH"
    fi
done

# Source nvm if available
if [ -s "$HOME/.nvm/nvm.sh" ]; then
    source "$HOME/.nvm/nvm.sh"
    echo "Sourced nvm"
fi

echo "##OCH:HEAL:REPAIR:fix-path.sh=success"
exit 0
