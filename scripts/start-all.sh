#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== Starting sim-rtu Full Stack ==="
echo ""

# 1. Build sim-rtu
echo "Building sim-rtu..."
cd "$REPO_DIR"
make build

# 2. Start sim-rtu in tmux
echo "Starting sim-rtu simulator..."
tmux new-session -d -s sim-rtu \
    "$REPO_DIR/bin/sim-rtu --config $REPO_DIR/configs/default.yml --log-level INFO 2>&1 | tee /tmp/sim-rtu.log"

# Wait for REST API to be ready
echo -n "Waiting for sim-rtu..."
for i in $(seq 1 30); do
    if curl -sf http://127.0.0.1:8080/api/v1/status > /dev/null 2>&1; then
        echo " ready!"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo " TIMEOUT - check /tmp/sim-rtu.log"
        exit 1
    fi
    echo -n "."
    sleep 1
done

# 3. Show device summary
echo ""
echo "=== Simulated Devices ==="
curl -s http://127.0.0.1:8080/api/v1/devices | python3 -m json.tool 2>/dev/null || \
    curl -s http://127.0.0.1:8080/api/v1/devices
echo ""

echo "=== sim-rtu running ==="
echo "  REST API: http://127.0.0.1:8080/api/v1/"
echo "  BACnet:   127.0.0.1:47808 (UDP)"
echo "  tmux:     tmux attach -t sim-rtu"
echo ""
echo "To start AEMS platform:"
echo "  1. Install configs: ./configs/aems-platform/setup.sh /path/to/aems-home"
echo "  2. Start aems-server"
echo "  3. Start bacnet-proxy agent"
echo "  4. Start platform-driver agent"
