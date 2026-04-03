#!/bin/bash
set -euo pipefail

# Install NF driver config files into the AEMS config store for sim-rtu integration.
# Usage: ./setup.sh [volttron-home-dir]
#
# Idempotent — safe to run multiple times.

AEMS_HOME="${1:-$HOME/.volttron_home}"
CONFIG_DIR="$AEMS_HOME/configs/nf-driver"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SIM_RTU_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "Installing NF driver configs for sim-rtu"
echo "  Target: $CONFIG_DIR"
echo ""

# Create config directory
mkdir -p "$CONFIG_DIR"

# Copy NF driver config
cp "$SCRIPT_DIR/config" "$CONFIG_DIR/config"

# Copy registry CSVs from sim-rtu configs
for csv in schneider openstat dent; do
    src="$SIM_RTU_ROOT/configs/${csv}.csv"
    if [ -f "$src" ]; then
        cp "$src" "$CONFIG_DIR/${csv}.csv"
    else
        echo "WARNING: Registry file not found: $src"
    fi
done

echo "Installed files:"
ls -la "$CONFIG_DIR/"
echo ""
echo "NF driver config installed. To activate:"
echo ""
echo "  1. In config.ini, set:"
echo "     [agents]"
echo "     platform_driver = false"
echo "     nf_driver = true"
echo ""
echo "  2. Ensure [nf_driver] config_file points to: configs/nf-driver/config"
echo ""
echo "  3. Restart AEMS"
