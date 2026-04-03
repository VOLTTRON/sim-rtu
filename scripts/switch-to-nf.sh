#!/bin/bash
set -euo pipefail

# Switch AEMS platform to use Normal Framework (NF) REST driver with sim-rtu.
# Usage: ./scripts/switch-to-nf.sh [aems-lib-fastapi-dir]
#
# This script:
#   1. Updates config.ini to enable nf_driver and disable platform_driver
#   2. Installs NF driver config files
#   3. Checks sim-rtu REST API connectivity
#
# Idempotent — safe to run multiple times.

AEMS_DIR="${1:-/home/debian/repos/aems-lib-fastapi}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SIM_RTU_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_INI="$AEMS_DIR/config.ini"

echo "============================================================"
echo "Switching AEMS to NF REST Driver mode"
echo "============================================================"
echo "AEMS dir:    $AEMS_DIR"
echo "sim-rtu dir: $SIM_RTU_ROOT"
echo ""

# --- Step 1: Validate paths ---
if [ ! -f "$CONFIG_INI" ]; then
    echo "ERROR: config.ini not found at $CONFIG_INI"
    echo "Provide the path to aems-lib-fastapi as the first argument."
    exit 1
fi

# --- Step 2: Update config.ini ---
echo "[1/3] Updating config.ini..."

# Enable nf_driver, disable platform_driver
sed -i 's/^platform_driver\s*=\s*true/platform_driver = false/' "$CONFIG_INI"
sed -i 's/^nf_driver\s*=\s*false/nf_driver = true/' "$CONFIG_INI"

# Verify the change
PD=$(grep '^platform_driver' "$CONFIG_INI" | head -1)
NF=$(grep '^nf_driver' "$CONFIG_INI" | head -1)
echo "  $PD"
echo "  $NF"
echo ""

# --- Step 3: Install NF driver config ---
echo "[2/3] Installing NF driver config files..."
NF_SETUP="$SIM_RTU_ROOT/configs/aems-platform/nf-driver/setup.sh"
VOLTTRON_HOME="$AEMS_DIR/.volttron_home"

if [ -f "$NF_SETUP" ]; then
    bash "$NF_SETUP" "$VOLTTRON_HOME"
else
    # Fallback: copy config directly
    NF_CONFIG_DIR="$AEMS_DIR/configs/nf-driver"
    mkdir -p "$NF_CONFIG_DIR"
    cp "$SIM_RTU_ROOT/configs/aems-platform/nf-driver/config" "$NF_CONFIG_DIR/config"

    # Copy registry CSVs
    for csv in schneider openstat dent; do
        src="$SIM_RTU_ROOT/configs/${csv}.csv"
        if [ -f "$src" ]; then
            cp "$src" "$NF_CONFIG_DIR/${csv}.csv"
        fi
    done
    echo "  Installed to $NF_CONFIG_DIR"
fi
echo ""

# --- Step 4: Check sim-rtu connectivity ---
echo "[3/3] Checking sim-rtu REST API connectivity..."
if curl -sf http://127.0.0.1:8080/api/v1/status > /dev/null 2>&1; then
    echo "  sim-rtu REST API: OK"

    # Quick point read test
    TEMP=$(curl -sf http://127.0.0.1:8080/api/v1/devices/86254/points/ZoneTemperature 2>/dev/null)
    if [ -n "$TEMP" ]; then
        echo "  Sample read (Schneider ZoneTemperature): $TEMP"
    fi
else
    echo "  WARNING: sim-rtu REST API not reachable at http://127.0.0.1:8080"
    echo "  Start sim-rtu before launching AEMS."
fi
echo ""

echo "============================================================"
echo "Switch complete. To start AEMS:"
echo "  cd $AEMS_DIR"
echo "  python orchestrate.py"
echo ""
echo "Or launch the NF driver directly:"
echo "  ./start-legacy-nf-driver.sh"
echo "============================================================"
