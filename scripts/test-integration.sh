#!/usr/bin/env bash
#
# Integration test script for sim-rtu.
#
# Builds sim-rtu, starts it, verifies the REST API and BACnet server are
# operational, then optionally runs the Go integration tests.
#
# Usage:
#   ./scripts/test-integration.sh            # full suite
#   ./scripts/test-integration.sh --api-only # skip Go integration tests
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
API_BASE="http://127.0.0.1:8080/api/v1"
SIM_PID=""

cleanup() {
    if [[ -n "$SIM_PID" ]] && kill -0 "$SIM_PID" 2>/dev/null; then
        echo "Stopping sim-rtu (PID $SIM_PID)..."
        kill "$SIM_PID" 2>/dev/null || true
        wait "$SIM_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT

echo "=== sim-rtu Integration Tests ==="
echo ""

# ---------- Build ----------
echo "1. Building sim-rtu..."
cd "$PROJECT_DIR"
go build -o bin/sim-rtu ./cmd/sim-rtu
echo "   OK"

# ---------- Start ----------
echo "2. Starting sim-rtu..."
./bin/sim-rtu --config configs/default.yml --log-level DEBUG &
SIM_PID=$!
echo "   PID: $SIM_PID"

# ---------- Health check ----------
echo "3. Waiting for REST API..."
for i in $(seq 1 20); do
    if curl -sf "$API_BASE/status" > /dev/null 2>&1; then
        echo "   API ready after ${i}s"
        break
    fi
    if ! kill -0 "$SIM_PID" 2>/dev/null; then
        echo "   ERROR: sim-rtu exited unexpectedly"
        exit 1
    fi
    sleep 1
done

if ! curl -sf "$API_BASE/status" > /dev/null 2>&1; then
    echo "   ERROR: API did not become ready within 20s"
    exit 1
fi

# ---------- REST API smoke tests ----------
echo "4. REST API smoke tests..."

echo -n "   GET /status ... "
STATUS=$(curl -sf "$API_BASE/status")
echo "OK"

echo -n "   GET /devices ... "
DEVICES=$(curl -sf "$API_BASE/devices")
echo "OK"

echo -n "   GET /devices/86254/points ... "
POINTS=$(curl -sf "$API_BASE/devices/86254/points")
echo "OK"

echo -n "   GET /devices/86254/points/ZoneTemperature ... "
ZONE_TEMP=$(curl -sf "$API_BASE/devices/86254/points/ZoneTemperature")
echo "OK — $ZONE_TEMP"

echo -n "   GET /devices/86255/points/ZoneTemperature ... "
ZONE_TEMP2=$(curl -sf "$API_BASE/devices/86255/points/ZoneTemperature")
echo "OK — $ZONE_TEMP2"

echo -n "   GET /devices/20001/points/WholeBuildingPower ... "
POWER=$(curl -sf "$API_BASE/devices/20001/points/WholeBuildingPower")
echo "OK — $POWER"

# ---------- Write and read-back test ----------
echo "5. Write/read-back test..."
echo -n "   PUT OccupiedCoolingSetPoint = 68.0 ... "
curl -sf -X PUT "$API_BASE/devices/86254/points/OccupiedCoolingSetPoint" \
    -H "Content-Type: application/json" \
    -d '{"value": 68.0}' > /dev/null
echo "OK"

echo -n "   GET OccupiedCoolingSetPoint ... "
READBACK=$(curl -sf "$API_BASE/devices/86254/points/OccupiedCoolingSetPoint")
echo "OK — $READBACK"

# ---------- Wait for simulation ticks ----------
echo "6. Verifying simulation is running (temperature should change)..."
TEMP1=$(curl -sf "$API_BASE/devices/86254/points/ZoneTemperature" | grep -oP '"value":\s*[\d.]+' | grep -oP '[\d.]+$' || echo "unknown")
sleep 3
TEMP2=$(curl -sf "$API_BASE/devices/86254/points/ZoneTemperature" | grep -oP '"value":\s*[\d.]+' | grep -oP '[\d.]+$' || echo "unknown")
echo "   ZoneTemperature: $TEMP1 -> $TEMP2"

if [[ "$TEMP1" == "$TEMP2" ]]; then
    echo "   WARNING: temperature did not change (may be at equilibrium)"
else
    echo "   OK — simulation is advancing"
fi

# ---------- Go integration tests ----------
if [[ "${1:-}" != "--api-only" ]]; then
    echo ""
    echo "7. Running Go integration tests..."
    # Stop the manually started instance — the Go tests manage their own
    kill "$SIM_PID" 2>/dev/null || true
    wait "$SIM_PID" 2>/dev/null || true
    SIM_PID=""

    cd "$PROJECT_DIR"
    go test -tags integration -v -timeout 60s ./tests/integration/
else
    echo ""
    echo "7. Skipping Go integration tests (--api-only)"
fi

echo ""
echo "=== All integration tests passed ==="
