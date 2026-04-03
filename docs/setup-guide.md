# Setup Guide: sim-rtu + AEMS Platform

From zero to a running sim-rtu simulator with AEMS platform reading and writing points.

## Prerequisites

- Go 1.24+ (`go version`)
- Python 3.10+ (`python3 --version`)
- Git
- Docker (optional, for orchestration pipeline)

## 1. Clone Repositories

```bash
mkdir -p ~/repos && cd ~/repos
git clone https://github.com/VOLTTRON/sim-rtu.git
git clone <your-aems-lib-fastapi-repo> aems-lib-fastapi
git clone https://github.com/VOLTTRON/volttron-pnnl-aems.git  # needed for NF driver agent source
```

## 2. Build sim-rtu

```bash
cd ~/repos/sim-rtu
make build
```

Output: `bin/sim-rtu`

## 3. Configure sim-rtu

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

## 4. Start sim-rtu

```bash
./bin/sim-rtu --config configs/default.yml
```

Or use the helper script (builds, starts in tmux, waits for ready):

```bash
./scripts/start-all.sh
```

Verify:

```bash
curl http://127.0.0.1:8080/api/v1/status
curl http://127.0.0.1:8080/api/v1/devices
```

## 5. Install aems-lib-fastapi

```bash
cd ~/repos/aems-lib-fastapi
python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[pipeline,drivers]"
```

For Python 3.12+ with BACnet legacy driver:

```bash
pip install bacpypes==0.16.7 pyasyncore pyasynchat
```

## 6. Configure AEMS

```bash
cd ~/repos/aems-lib-fastapi
cp config.ini.example config.ini
```

### Option A: NF Driver (Recommended)

Edit `config.ini`:

```ini
[agents]
platform_driver = false
nf_driver = true
```

Install NF driver configs:

```bash
~/repos/sim-rtu/scripts/switch-to-nf.sh ~/repos/aems-lib-fastapi
```

### Option B: Legacy BACnet Driver

Edit `config.ini`:

```ini
[agents]
platform_driver = true
nf_driver = false
```

Install BACnet configs:

```bash
~/repos/sim-rtu/scripts/switch-to-bacnet.sh ~/repos/aems-lib-fastapi
```

## 7. Start AEMS Platform

### Without Docker (development)

```bash
cd ~/repos/aems-lib-fastapi
source .venv/bin/activate

# Terminal 1: Start server
aems-server --host 127.0.0.1 --port 8000

# Terminal 2: Start agents (orchestration pipeline)
python orchestrate.py
```

### With Docker (orchestration pipeline)

```bash
cd ~/repos/aems-lib-fastapi
source .venv/bin/activate
python orchestrate.py

cd ~/repos/volttron-pnnl-aems/aems-app/docker
docker compose --profile fastapi --profile fastapi-agents up -d
```

## 8. Verify Everything Works

```bash
# sim-rtu health
curl http://127.0.0.1:8080/api/v1/status

# Read a point directly from sim-rtu
curl http://127.0.0.1:8080/api/v1/devices/86254/points/ZoneTemperature

# Read a point through AEMS
curl http://127.0.0.1:8000/devices/SIM/RTU/Schneider/ZoneTemperature

# Write a setpoint through AEMS
curl -X PUT http://127.0.0.1:8000/devices/SIM/RTU/Schneider/OccupiedCoolingSetPoint \
     -H "Content-Type: application/json" \
     -d '{"value": 73.0, "priority": 16}'

# Read all three devices
curl http://127.0.0.1:8000/devices/SIM/RTU/Schneider
curl http://127.0.0.1:8000/devices/SIM/RTU/OpenStat
curl http://127.0.0.1:8000/devices/SIM/RTU/DENT
```

## 9. Common Operations

**Read a point (sim-rtu direct):**
```bash
curl http://127.0.0.1:8080/api/v1/devices/86254/points/ZoneTemperature
```

**Write a setpoint (sim-rtu direct):**
```bash
curl -X PUT http://127.0.0.1:8080/api/v1/devices/86254/points/OccupiedCoolingSetPoint \
     -H "Content-Type: application/json" \
     -d '{"value": 73.0, "priority": 16}'
```

**Check simulation status:**
```bash
curl http://127.0.0.1:8080/api/v1/status
```

**Override weather (outdoor temperature):**
```bash
curl -X POST http://127.0.0.1:8080/api/v1/weather \
     -H "Content-Type: application/json" \
     -d '{"temperature": 95.0}'
```

**Switch driver modes:** See [driver-switching.md](driver-switching.md).

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `Address already in use` on 47808 | Another process on that UDP port. `lsof -i UDP:47808` to find it. |
| `No module named asyncore` | Python 3.12+ removed asyncore. `pip install pyasyncore pyasynchat` |
| `No response from device` (BACnet) | Check `device_address` in config. Use `172.17.0.1` when AEMS runs in Docker. |
| Connection refused on 8080 | sim-rtu not running or bound to `127.0.0.1`. Use `0.0.0.0` for Docker. |
| AEMS not polling | Check `polling_interval` in NF config. Default is 60s — wait a full cycle. |
| `bacpypes` import errors | Must be `bacpypes==0.16.7` exactly (not bacpypes3). |
| Config changes not taking effect | AEMS reads configs at startup. Restart after running setup scripts. |
| `orchestrate.py` exits code 2 | `volttron-pnnl-aems/aems-edge` not found. Use `--aems-edge-path`. |
| Docker: can't reach sim-rtu | Use `172.17.0.1` (docker0 bridge) or container name if same network. |
