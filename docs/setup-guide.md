# Setup Guide: AEMS Platform with RTU Devices

Three scenarios covered:
1. **[Simulated](#scenario-1-simulated-sim-rtu-only)** — sim-rtu for development/testing
2. **[Real Hardware](#scenario-2-real-rtu-hardware)** — Schneider/OpenStat thermostats + DENT meters via BACnet
3. **[Mixed Mode](#scenario-3-mixed-sim--real)** — sim-rtu alongside real devices

---

## Prerequisites (All Scenarios)

- Python 3.10+ (`python3 --version`)
- Git
- Docker (optional, for orchestration pipeline)

| Scenario | Additional Requirements |
|----------|------------------------|
| Simulated | Go 1.24+ |
| Real Hardware | BACnet/IP network, device IPs, BACnet device IDs |
| Mixed | Both of the above |

---

## Scenario 1: Simulated (sim-rtu only)

For development and testing without physical hardware.

### 1.1 Clone Repositories

```bash
mkdir -p ~/repos && cd ~/repos
git clone https://github.com/VOLTTRON/sim-rtu.git
git clone <your-aems-lib-fastapi-repo> aems-lib-fastapi
git clone https://github.com/VOLTTRON/volttron-pnnl-aems.git  # needed for NF driver agent source
```

### 1.2 Build sim-rtu

```bash
cd ~/repos/sim-rtu
make build
```

Output: `bin/sim-rtu`

### 1.3 Configure sim-rtu

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

### 1.4 Start sim-rtu

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

### 1.5 Install aems-lib-fastapi

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

### 1.6 Configure AEMS

```bash
cd ~/repos/aems-lib-fastapi
cp config.ini.example config.ini
```

#### Option A: NF Driver (Recommended)

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

#### Option B: Legacy BACnet Driver

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

### 1.7 Start AEMS Platform

#### Without Docker (development)

```bash
cd ~/repos/aems-lib-fastapi
source .venv/bin/activate

# Terminal 1: Start server
aems-server --host 127.0.0.1 --port 8000

# Terminal 2: Start agents (orchestration pipeline)
python orchestrate.py
```

#### With Docker (orchestration pipeline)

```bash
cd ~/repos/aems-lib-fastapi
source .venv/bin/activate
python orchestrate.py

cd ~/repos/volttron-pnnl-aems/aems-app/docker
docker compose --profile fastapi --profile fastapi-agents up -d
```

### 1.8 Verify (Simulated)

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

---

## Scenario 2: Real RTU Hardware

Connecting to physical Schneider SE8650 / OpenStat thermostats and DENT meters via BACnet/IP.

### 2.1 Hardware Prerequisites

| Component | Details |
|-----------|---------|
| BACnet/IP network | Accessible from AEMS host, UDP port 47808 open |
| Thermostats | Schneider SE8650 or OpenStat with BACnet/IP enabled |
| Power meter | DENT meter (optional) |
| Device IPs | From commissioning or BACnet discovery (see [2.3](#23-discover-bacnet-devices)) |
| Device IDs | BACnet device instance numbers from device configuration |
| NF gateway | Required only if using NF driver — IP, port, OAuth credentials |

### 2.2 Network Requirements

- AEMS host must be on the same subnet as BACnet devices (or have a BACnet router)
- UDP port **47808** must be open between AEMS host and devices
- If using NF gateway: HTTP access to gateway on port 8081

### 2.3 Discover BACnet Devices

Use `bacpypes` to find devices on the network:

```bash
pip install bacpypes==0.16.7

# WhoIs broadcast — lists all BACnet devices
python3 -c "
from bacpypes.app import BIPSimpleApplication
from bacpypes.local.device import LocalDeviceObject
from bacpypes.pdu import Address
from bacpypes.apdu import WhoIsRequest
import time

device = LocalDeviceObject(objectIdentifier=('device', 999), objectName='discovery')
app = BIPSimpleApplication(device, '0.0.0.0')
app.who_is()
time.sleep(3)
for k, v in app.i_am_devices.items():
    print(f'Device {k}: address={v}')
"
```

Or use ping to verify specific device IPs:

```bash
ping -c 1 192.168.1.101   # thermostat 1
ping -c 1 192.168.1.102   # thermostat 2
ping -c 1 192.168.1.100   # DENT meter
```

### 2.4 config.ini for Real Devices

```bash
cd ~/repos/aems-lib-fastapi
cp config.ini.example config.ini
```

Edit the key sections:

```ini
[site]
campus = PNNL
building = SEB
timezone = America/Los_Angeles
gateway_address = 192.168.1.1
stat_type = schneider          # or openstat

[device:1]
name = rtu01
address = 192.168.1.101        # actual device IP
device_id = 1001               # BACnet device instance
stat_type = schneider

[device:2]
name = rtu02
address = 192.168.1.102
device_id = 1002
stat_type = schneider

[meter]
name = meter
address = 192.168.1.100
device_id = 100
registry = dent.csv
```

### 2.5 Driver Configuration

Choose one driver approach:

#### Option A: NF Driver with Real Gateway

For sites using the Normal Framework gateway:

```ini
# config.ini
[agents]
platform_driver = false
nf_driver = true
```

NF driver config (`configs/nf-driver/config`):

```yaml
polling_interval: 60
driver_config:
  url: http://aems-gateway.local:8081   # real NF gateway address
  client_id: my-client                  # OAuth credentials from gateway admin
  client_secret: my-secret
device_list:
- device_id: 1001
  registry_file: schneider.csv
  points_per_request: 25
  topic: PNNL/SEB/RTU01
- device_id: 1002
  registry_file: schneider.csv
  points_per_request: 25
  topic: PNNL/SEB/RTU02
```

#### Option B: BACnet Driver with Real Devices

Direct BACnet/IP — no gateway needed:

```ini
# config.ini
[agents]
platform_driver = true
nf_driver = false
```

Device config (`configs/platform-driver/devices/rtu01.config`):

```json
{
    "driver_config": {
        "device_address": "192.168.1.101",
        "device_id": 1001
    },
    "driver_type": "bacnet",
    "registry_config": "config://registry_configs/schneider.csv",
    "interval": 60,
    "timezone": "US/Pacific",
    "heart_beat_point": "HeartBeat"
}
```

DENT meter config (`configs/platform-driver/devices/meter.config`):

```json
{
    "driver_config": {
        "device_address": "192.168.1.100",
        "device_id": 100
    },
    "driver_type": "bacnet",
    "registry_config": "config://registry_configs/dent.csv",
    "interval": 60,
    "timezone": "US/Pacific"
}
```

### 2.6 Registry CSVs

The same registry files work for both simulated and real devices:

| File | Device | Points |
|------|--------|--------|
| `schneider.csv` | Schneider SE8650 | ~60 points (temps, setpoints, modes, stages) |
| `dent.csv` | DENT power meter | ~40 points (voltage, current, power, PF, THD) |

No modifications needed — these map BACnet object types and indices to named points.

### 2.7 Thermostat Configuration

Site-specific thermostat config (e.g., `configs/aems-manager/schneider.config`):

```json
{
    "campus": "PNNL",
    "building": "SEB",
    "system": "SCHNEIDER",
    "system_status_point": "OccupancyCommand",
    "setpoint_control": 1,
    "local_tz": "US/Pacific",
    "default_setpoints": {
        "UnoccupiedHeatingSetPoint": 65,
        "UnoccupiedCoolingSetPoint": 78,
        "DeadBand": 3,
        "OccupiedSetPoint": 71
    },
    "schedule": {
        "Monday":    {"start": "6:00", "end": "18:00"},
        "Tuesday":   {"start": "6:00", "end": "18:00"},
        "Wednesday": {"start": "6:00", "end": "18:00"},
        "Thursday":  {"start": "6:00", "end": "18:00"},
        "Friday":    {"start": "6:00", "end": "18:00"},
        "Saturday":  "always_off",
        "Sunday":    "always_off"
    },
    "occupancy_values": {
        "occupied": 2,
        "unoccupied": 3
    }
}
```

Key fields to customize per site:

| Field | Description |
|-------|-------------|
| `campus` / `building` / `system` | Must match your topic hierarchy |
| `setpoint_control` | `1` = Schneider dual-setpoint, `0` = OpenStat single-setpoint |
| `default_setpoints` | Unoccupied fallback temperatures |
| `schedule` | Occupied hours per day of week |
| `occupancy_values` | Schneider: `2`/`3`, OpenStat: `1`/`0` |

### 2.8 Verify (Real Hardware)

```bash
# Check BACnet connectivity (UDP 47808)
nc -zu 192.168.1.101 47808

# Read a point through AEMS
curl http://127.0.0.1:8000/devices/PNNL/SEB/RTU01/ZoneTemperature

# Check AEMS logs for polling activity
tail -f /var/log/aems/platform.driver.log

# Write a setpoint
curl -X PUT http://127.0.0.1:8000/devices/PNNL/SEB/RTU01/OccupiedCoolingSetPoint \
     -H "Content-Type: application/json" \
     -d '{"value": 74.0, "priority": 16}'
```

### 2.9 Common Issues (Real Hardware)

| Problem | Fix |
|---------|-----|
| No response from device | Firewall blocking UDP 47808. Check `iptables -L` and device subnet. |
| Wrong values / no data | Verify `device_id` matches the BACnet instance on the physical device. |
| `device_address` confusion | This is the device IP, not the gateway IP. |
| Intermittent reads | BACnet has no retry by default. Reduce `points_per_request` to avoid timeouts. |
| NF gateway auth failure | Check `client_id`/`client_secret` in NF config. Verify gateway is reachable. |
| Wrong occupancy behavior | Schneider uses `2`=occupied/`3`=unoccupied. OpenStat uses `1`/`0`. |

---

## Scenario 3: Mixed (sim + real)

Run sim-rtu for development devices alongside real hardware. Common workflow: develop against simulated devices, then verify on physical RTUs.

### 3.1 Setup

1. Complete [Scenario 1](#scenario-1-simulated-sim-rtu-only) setup (sim-rtu running)
2. Add real devices to `config.ini` alongside simulated ones

### 3.2 config.ini for Mixed Mode

```ini
[site]
campus = PNNL
building = SEB
timezone = America/Los_Angeles
gateway_address = 192.168.1.1
stat_type = schneider

# --- Simulated devices (from sim-rtu) ---
[device:1]
name = sim-schneider
address = 127.0.0.1             # sim-rtu address
device_id = 86254               # sim-rtu default device ID
stat_type = schneider

# --- Real devices ---
[device:2]
name = rtu01
address = 192.168.1.101         # physical device
device_id = 1001
stat_type = schneider

[device:3]
name = rtu02
address = 192.168.1.102
device_id = 1002
stat_type = schneider

[meter]
name = meter
address = 192.168.1.100
device_id = 100
registry = dent.csv
```

### 3.3 Topic Separation

Use different topic prefixes to distinguish simulated vs real:

| Device | Topic | Source |
|--------|-------|--------|
| sim-schneider | `SIM/RTU/Schneider` | sim-rtu on localhost |
| rtu01 | `PNNL/SEB/RTU01` | Real Schneider SE8650 |
| rtu02 | `PNNL/SEB/RTU02` | Real Schneider SE8650 |
| meter | `PNNL/SEB/METER` | Real DENT meter |

### 3.4 Verify Mixed Mode

```bash
# Simulated device
curl http://127.0.0.1:8000/devices/SIM/RTU/Schneider/ZoneTemperature

# Real device
curl http://127.0.0.1:8000/devices/PNNL/SEB/RTU01/ZoneTemperature

# Both should return data
```

---

## Common Operations

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

---

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
| Firewall blocking BACnet | Open UDP 47808: `sudo ufw allow 47808/udp` |
| Wrong BACnet device ID | Run WhoIs discovery (see [2.3](#23-discover-bacnet-devices)) to find actual IDs. |
| NF gateway unreachable | Verify gateway URL, check OAuth credentials, test with `curl http://<gateway>:8081/health`. |
