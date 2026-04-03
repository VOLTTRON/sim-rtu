#!/bin/bash
set -euo pipefail

# Switch AEMS platform to use Legacy BACnet driver with sim-rtu.
# Usage: ./scripts/switch-to-bacnet.sh [aems-lib-fastapi-dir]
#
# This script:
#   1. Updates config.ini to enable platform_driver and disable nf_driver
#   2. Installs BACnet config store files (if setup.sh exists)
#   3. Verifies BACnet Python dependencies
#   4. Checks sim-rtu connectivity
#
# Idempotent — safe to run multiple times.

AEMS_DIR="${1:-/home/debian/repos/aems-lib-fastapi}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SIM_RTU_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_INI="$AEMS_DIR/config.ini"

echo "============================================================"
echo "Switching AEMS to Legacy BACnet Driver mode"
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
echo "[1/4] Updating config.ini..."

# Enable platform_driver, disable nf_driver
sed -i 's/^platform_driver\s*=\s*false/platform_driver = true/' "$CONFIG_INI"
sed -i 's/^nf_driver\s*=\s*true/nf_driver = false/' "$CONFIG_INI"

# Verify the change
PD=$(grep '^platform_driver' "$CONFIG_INI" | head -1)
NF=$(grep '^nf_driver' "$CONFIG_INI" | head -1)
echo "  $PD"
echo "  $NF"
echo ""

# --- Step 3: Install config store files ---
echo "[2/4] Installing BACnet config store files..."
SETUP_SCRIPT="$SIM_RTU_ROOT/configs/aems-platform/setup.sh"
VOLTTRON_HOME="$AEMS_DIR/.volttron_home"

if [ -f "$SETUP_SCRIPT" ]; then
    bash "$SETUP_SCRIPT" "$VOLTTRON_HOME"
else
    echo "  WARNING: setup.sh not found at $SETUP_SCRIPT"
    echo "  Config store files may need manual installation."
fi
echo ""

# --- Step 4: Check BACnet dependencies ---
echo "[3/4] Checking BACnet Python dependencies..."
MISSING=""

python3 -c "import bacpypes" 2>/dev/null || MISSING="$MISSING bacpypes"
python3 -c "import asyncore" 2>/dev/null || {
    python3 -c "import pyasyncore" 2>/dev/null || MISSING="$MISSING pyasyncore"
}

if [ -n "$MISSING" ]; then
    echo "  WARNING: Missing packages:$MISSING"
    echo "  Install with: pip install bacpypes==0.16.7 pyasyncore pyasynchat"
else
    echo "  All BACnet dependencies found."
fi
echo ""

# --- Step 5: Check sim-rtu connectivity ---
echo "[4/4] Checking sim-rtu connectivity..."
if curl -sf http://127.0.0.1:8080/api/v1/status > /dev/null 2>&1; then
    echo "  sim-rtu REST API: OK"
else
    echo "  WARNING: sim-rtu REST API not reachable at http://127.0.0.1:8080"
    echo "  Start sim-rtu before launching AEMS."
fi

if ss -uln 2>/dev/null | grep -q ':47808 '; then
    echo "  sim-rtu BACnet port 47808: OK"
else
    echo "  WARNING: Nothing listening on UDP 47808"
    echo "  Ensure sim-rtu is running with bacnet.enabled: true"
fi
echo ""

echo "============================================================"
echo "Switch complete. To start AEMS:"
echo "  cd $AEMS_DIR"
echo "  python orchestrate.py"
echo "============================================================"
