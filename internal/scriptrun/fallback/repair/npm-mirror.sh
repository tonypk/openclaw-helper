#!/bin/bash
set -e

echo "##OCH:HEAL:REPAIR:npm-mirror.sh=running"

# Try multiple registries
REGISTRIES=(
    "https://registry.npmmirror.com"
    "https://registry.npmjs.org"
)

for reg in "${REGISTRIES[@]}"; do
    echo "Trying registry: $reg"
    if curl -s --max-time 5 "$reg" > /dev/null 2>&1; then
        npm config set registry "$reg"
        echo "Set npm registry to $reg"
        echo "##OCH:HEAL:REPAIR:npm-mirror.sh=success"
        exit 0
    fi
done

echo "All registries unreachable"
echo "##OCH:HEAL:REPAIR:npm-mirror.sh=failed"
exit 1
