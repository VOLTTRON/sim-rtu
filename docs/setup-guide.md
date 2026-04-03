# sim-rtu Setup Guide

Build and run the RTU simulator. For full AEMS platform setup (connecting sim-rtu to aems-lib-fastapi), see the [AEMS Platform Setup Guide](https://github.com/VOLTTRON/aems-lib-fastapi/blob/develop/docs/SETUP_GUIDE.md).

## Prerequisites

- Go 1.24+ (`go version`)
- Git

## Build

```bash
git clone https://github.com/VOLTTRON/sim-rtu.git
cd sim-rtu
make build
```

Output: `bin/sim-rtu`

## Configure

Default config works out of the box for localhost development:

```bash
cat configs/default.yml
```

Key settings:

| Setting | Default | Notes |
|---------|---------|-------|
| `api.host` | `127.0.0.1` | Change to `0.0.0.0` for Docker/remote access |
| `api.port` | `8080` | REST + NF API |
| `bacnet.interface` | `127.0.0.1` | Change to `0.0.0.0` for Docker/remote access |
| `bacnet.port` | `47808` | Standard BACnet/IP |

## Run

```bash
./bin/sim-rtu --config configs/default.yml
```

Or use the helper script (builds, starts in tmux, waits for ready):

```bash
./scripts/start-all.sh
```

## Verify

```bash
# Simulator status
curl http://127.0.0.1:8080/api/v1/status

# List devices
curl http://127.0.0.1:8080/api/v1/devices

# Read a point
curl http://127.0.0.1:8080/api/v1/devices/86254/points/ZoneTemperature

# Write a setpoint
curl -X PUT http://127.0.0.1:8080/api/v1/devices/86254/points/OccupiedCoolingSetPoint \
     -H "Content-Type: application/json" \
     -d '{"value": 73.0, "priority": 16}'

# Override weather
curl -X POST http://127.0.0.1:8080/api/v1/weather \
     -H "Content-Type: application/json" \
     -d '{"temperature": 95.0}'
```

## Docker

```bash
docker compose up
```

Bind to `0.0.0.0` in `configs/default.yml` for Docker access:

```yaml
api:
  host: "0.0.0.0"
bacnet:
  interface: "0.0.0.0"
```

## AEMS Driver Config Scripts

These scripts install sim-rtu device configs into an aems-lib-fastapi checkout:

```bash
# NF driver (recommended)
./scripts/switch-to-nf.sh /path/to/aems-lib-fastapi

# Legacy BACnet driver
./scripts/switch-to-bacnet.sh /path/to/aems-lib-fastapi
```

See [driver-switching.md](driver-switching.md) for details.

## Next Steps

Connect sim-rtu to the AEMS platform: [AEMS Platform Setup Guide](https://github.com/VOLTTRON/aems-lib-fastapi/blob/develop/docs/SETUP_GUIDE.md)
