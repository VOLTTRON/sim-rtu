#!/bin/bash
set -euo pipefail

AEMS_HOME="${1:-$HOME/.volttron_home}"
CONFIG_STORE="$AEMS_HOME/aems_config_store"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Installing AEMS platform configs to: $CONFIG_STORE"
echo ""

# Create directories
mkdir -p "$CONFIG_STORE/platform.driver/devices/SIM/RTU"
mkdir -p "$CONFIG_STORE/platform.driver/registry_configs"
mkdir -p "$CONFIG_STORE/platform.bacnet_proxy"

# Copy platform driver main config
cp "$SCRIPT_DIR/platform-driver/config" "$CONFIG_STORE/platform.driver/config"
echo '{"type": "json"}' > "$CONFIG_STORE/platform.driver/config.metadata"

# Copy device configs
for device in Schneider OpenStat DENT; do
    cp "$SCRIPT_DIR/platform-driver/devices/SIM/RTU/$device" \
       "$CONFIG_STORE/platform.driver/devices/SIM/RTU/$device"
    echo '{"type": "json"}' > "$CONFIG_STORE/platform.driver/devices/SIM/RTU/${device}.metadata"
done

# Copy registry CSVs and metadata
for csv in schneider openstat dent; do
    cp "$SCRIPT_DIR/platform-driver/registry_configs/${csv}.csv" \
       "$CONFIG_STORE/platform.driver/registry_configs/${csv}.csv"
    cp "$SCRIPT_DIR/platform-driver/registry_configs/${csv}.csv.metadata" \
       "$CONFIG_STORE/platform.driver/registry_configs/${csv}.csv.metadata"
done

# Copy BACnet proxy config
cp "$SCRIPT_DIR/bacnet-proxy/config" "$CONFIG_STORE/platform.bacnet_proxy/config"
echo '{"type": "json"}' > "$CONFIG_STORE/platform.bacnet_proxy/config.metadata"

echo "Installed configs:"
find "$CONFIG_STORE" -type f | sort
echo ""
echo "Done. Start AEMS with:"
echo "  aems-server --host 127.0.0.1 --port 8000 --volttron-home $AEMS_HOME"
