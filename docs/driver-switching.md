# Switching Between BACnet and NF Driver Modes

Guide for switching the AEMS platform (aems-lib-fastapi) between the Legacy Platform Driver (BACnet) and the Normal Framework Driver (NF REST) when using sim-rtu.

## 1. Overview

sim-rtu exposes two interfaces simultaneously:

- **BACnet/IP** on UDP port 47808 — legacy protocol, used by the VOLTTRON Platform Driver
- **REST API** on HTTP port 8080 — used by the Normal Framework (NF) Driver

The AEMS platform can consume either interface. Only one driver should be active at a time (they share the `platform.driver` identity).

### Data Flow

```
Mode A — Legacy Platform Driver (BACnet)

  sim-rtu :47808 (UDP)
      │
      ▼
  BACnet Proxy Agent (translates BACnet/IP ↔ internal)
      │
      ▼
  Platform Driver Agent (identity: platform.driver)
      │
      ▼
  VOLTTRON message bus → AEMS Manager, Historian, ILC


Mode B — Normal Framework Driver (NF REST)

  sim-rtu :8080 (HTTP)
      │
      ▼
  NF Driver Agent (identity: platform.driver)
      │
      ▼
  VOLTTRON message bus → AEMS Manager, Historian, ILC
```

### When to Use Each

| Mode | Use Case |
|------|----------|
| **Legacy BACnet** | Testing against real BACnet protocol behavior; validating BACnet point mappings; debugging BACnet-specific issues |
| **NF REST** | Faster development iteration; no BACnet proxy needed; simpler config; recommended for most sim-rtu testing |

## 2. Prerequisites

### Both Modes

1. **sim-rtu** built and running with both interfaces enabled:
   ```bash
   cd /path/to/sim-rtu
   make build
   ./bin/sim-rtu --config configs/default.yml
   ```
   Verify: `curl http://127.0.0.1:8080/api/v1/status`

2. **aems-lib-fastapi** installed:
   ```bash
   cd /path/to/aems-lib-fastapi
   pip install -e .
   ```

3. **sim-rtu default.yml** must have both interfaces enabled (this is the default):
   ```yaml
   bacnet:
     enabled: true
     port: 47808

   api:
     enabled: true
     port: 8080
   ```

### BACnet Mode Only

Install the BACnet Python dependencies:
```bash
pip install bacpypes==0.16.7 pyasyncore pyasynchat
```

**Note:** Must be `bacpypes==0.16.7` exactly (not bacpypes3). The `pyasyncore` and `pyasynchat` packages are required for Python 3.12+.

### NF REST Mode Only

No additional dependencies beyond aems-lib-fastapi.

## 3. Mode A — Legacy Platform Driver (BACnet)

### 3.1 config.ini Settings

In `aems-lib-fastapi/config.ini`, set:

```ini
[agents]
platform_driver = true
nf_driver = false
```

### 3.2 Install Config Store Files

Run the setup script from the sim-rtu repo:

```bash
cd /path/to/sim-rtu
./configs/aems-platform/setup.sh /path/to/aems-lib-fastapi/.volttron_home
```

This installs into the config store (`aems_config_store/platform.driver/`):

| File | Purpose |
|------|---------|
| `config` | Platform driver main config (scrape interval, publish mode) |
| `devices/SIM/RTU/Schneider` | Schneider device config (device_id: 86254) |
| `devices/SIM/RTU/OpenStat` | OpenStat device config (device_id: 86255) |
| `devices/SIM/RTU/DENT` | DENT meter config (device_id: 20001) |
| `registry_configs/schneider.csv` | Schneider BACnet point registry (60 points) |
| `registry_configs/openstat.csv` | OpenStat BACnet point registry (66 points) |
| `registry_configs/dent.csv` | DENT meter point registry (40 points) |

Each file has a corresponding `.metadata` file.

### 3.3 BACnet Proxy Config

The setup script also installs the BACnet proxy config to `aems_config_store/platform.bacnet_proxy/config`:

```json
{
    "device_address": "127.0.0.1/24",
    "max_apdu_length": 1024,
    "object_name": "AEMS BACnet Proxy",
    "object_identifier": 599,
    "max_per_request": 24,
    "default_priority": 16,
    "timeout": 10
}
```

The proxy binds a random high UDP port (not 47808 — sim-rtu owns that port).

### 3.4 Device Config Format

Each device config in the config store looks like:

```json
{
    "driver_config": {
        "device_address": "127.0.0.1",
        "device_id": 86254
    },
    "driver_type": "bacnet",
    "registry_config": "config://registry_configs/schneider.csv",
    "interval": 60,
    "timezone": "US/Pacific",
    "heart_beat_point": "HeartBeat"
}
```

When running in Docker, change `device_address` to `172.17.0.1` (docker0 bridge) to reach the host.

### 3.5 Start AEMS

```bash
cd /path/to/aems-lib-fastapi
python orchestrate.py
# or
aems-server --host 127.0.0.1 --port 8000
```

### 3.6 Verify

```bash
# Check sim-rtu is serving BACnet
ss -uln | grep 47808

# Read a point through AEMS
curl http://127.0.0.1:8000/devices/SIM/RTU/Schneider/ZoneTemperature

# Read DENT meter
curl http://127.0.0.1:8000/devices/SIM/RTU/DENT/WholeBuildingPower
```

## 4. Mode B — Normal Framework Driver (NF REST)

### 4.1 config.ini Settings

In `aems-lib-fastapi/config.ini`, set:

```ini
[agents]
platform_driver = false
nf_driver = true
```

### 4.2 NF Driver Config

The NF driver reads a YAML config file. Install the sim-rtu NF config:

```bash
cd /path/to/sim-rtu
./configs/aems-platform/nf-driver/setup.sh /path/to/aems-lib-fastapi/.volttron_home
```

Or manually copy the config:

```bash
mkdir -p /path/to/aems-lib-fastapi/configs/nf-driver
cp /path/to/sim-rtu/configs/aems-platform/nf-driver/config \
   /path/to/aems-lib-fastapi/configs/nf-driver/config
```

The NF driver config for sim-rtu:

```yaml
driver_config:
  url: http://127.0.0.1:8080
device_list:
- device_id: 86254
  registry_file: schneider.csv
  topic: SIM/RTU/Schneider
- device_id: 86255
  registry_file: openstat.csv
  topic: SIM/RTU/OpenStat
- device_id: 20001
  registry_file: dent.csv
  topic: SIM/RTU/DENT
polling_interval: 60
```

When running in Docker, change `url` to `http://172.17.0.1:8080`.

### 4.3 config.ini NF Section

Ensure the `[nf_driver]` section points to the correct config:

```ini
[nf_driver]
agent_dir = /volttron-pnnl-aems/aems-edge/Normal
identity = platform.driver
config_file = configs/nf-driver/config
description = Normal Framework BACnet driver for sim-rtu devices
```

### 4.4 Start AEMS

```bash
cd /path/to/aems-lib-fastapi
python orchestrate.py
```

Or launch the NF driver directly:

```bash
./start-legacy-nf-driver.sh
```

### 4.5 Verify

```bash
# Check sim-rtu REST API is up
curl http://127.0.0.1:8080/api/v1/devices

# Read a point through AEMS
curl http://127.0.0.1:8000/devices/SIM/RTU/Schneider/ZoneTemperature

# Read all three devices
curl http://127.0.0.1:8000/devices/SIM/RTU/Schneider
curl http://127.0.0.1:8000/devices/SIM/RTU/OpenStat
curl http://127.0.0.1:8000/devices/SIM/RTU/DENT
```

## 5. Switching Between Modes

### Switch from BACnet (A) to NF REST (B)

```bash
# 1. Stop AEMS
#    (Ctrl+C the running process, or kill the container)

# 2. Edit config.ini
sed -i 's/^platform_driver = true/platform_driver = false/' config.ini
sed -i 's/^nf_driver = false/nf_driver = true/' config.ini

# 3. Install NF driver config (if not already done)
/path/to/sim-rtu/configs/aems-platform/nf-driver/setup.sh .volttron_home

# 4. Start AEMS
python orchestrate.py

# 5. Verify
curl http://127.0.0.1:8000/devices/SIM/RTU/Schneider/ZoneTemperature
```

Or use the helper script:
```bash
/path/to/sim-rtu/scripts/switch-to-nf.sh /path/to/aems-lib-fastapi
```

### Switch from NF REST (B) to BACnet (A)

```bash
# 1. Stop AEMS

# 2. Edit config.ini
sed -i 's/^platform_driver = false/platform_driver = true/' config.ini
sed -i 's/^nf_driver = true/nf_driver = false/' config.ini

# 3. Install BACnet configs (if not already done)
/path/to/sim-rtu/configs/aems-platform/setup.sh .volttron_home

# 4. Verify BACnet dependencies
pip install bacpypes==0.16.7 pyasyncore pyasynchat

# 5. Start AEMS
python orchestrate.py

# 6. Verify
curl http://127.0.0.1:8000/devices/SIM/RTU/Schneider/ZoneTemperature
```

Or use the helper script:
```bash
/path/to/sim-rtu/scripts/switch-to-bacnet.sh /path/to/aems-lib-fastapi
```

## 6. Running Both Simultaneously

Both drivers use the identity `platform.driver`, so they **cannot run simultaneously** in the same AEMS instance. Only one can be active.

If you need to compare outputs side-by-side:

1. Run two separate AEMS instances with different `VOLTTRON_HOME` directories
2. Configure one with BACnet, the other with NF REST
3. Start them on different ports (e.g., 8000 and 8001)

**Caveat:** Both drivers publish to the same VOLTTRON topics (`SIM/RTU/Schneider`, etc.). If writing to the message bus, ensure only one instance is connected to avoid write conflicts.

## 7. Quick Reference

| | Legacy (BACnet) | NF Driver (REST) |
|---|---|---|
| **Protocol** | BACnet/IP (UDP 47808) | HTTP POST (port 8080) |
| **Requires BACnet proxy** | Yes | No |
| **Extra Python deps** | bacpypes==0.16.7, pyasyncore, pyasynchat | None |
| **config.ini toggle** | `platform_driver = true` | `nf_driver = true` |
| **sim-rtu interface** | `bacnet.enabled: true` | `api.enabled: true` |
| **Driver identity** | `platform.driver` | `platform.driver` |
| **Config store path** | `aems_config_store/platform.driver/` | `configs/nf-driver/config` |
| **Device config format** | JSON per device (config store) | Single YAML with device_list |
| **Registry files** | CSV in config store + .metadata | CSV referenced by filename |
| **Docker host address** | `172.17.0.1` | `http://172.17.0.1:8080` |
| **Setup script** | `configs/aems-platform/setup.sh` | `configs/aems-platform/nf-driver/setup.sh` |
| **Complexity** | Higher (proxy + per-device configs) | Lower (single config file) |

## 8. Troubleshooting

### Port Conflicts

**Symptom:** "Address already in use" on UDP 47808.

sim-rtu and the BACnet proxy cannot both bind port 47808. The proxy uses a random high port by default. If you see this error:

```bash
# Find what's using the port
lsof -i UDP:47808

# Kill stale processes if needed
kill <PID>
```

### bacpypes Version Issues

**Symptom:** Import errors or "No module named asyncore".

```bash
# Must be exactly this version
pip install bacpypes==0.16.7

# Python 3.12+ removed asyncore/asynchat from stdlib
pip install pyasyncore pyasynchat
```

### Config Store Stale Entries

**Symptom:** Old device configs still active after switching modes.

The config store is read at startup. After changing configs:

1. Stop AEMS completely
2. Re-run the appropriate setup script
3. Start AEMS again

To inspect the config store:
```bash
find .volttron_home/aems_config_store -type f | sort
```

### Device Address — Localhost vs Docker Bridge

**Symptom:** "No response from device" or connection refused.

| Scenario | BACnet device_address | NF driver URL |
|----------|----------------------|---------------|
| Both on host | `127.0.0.1` | `http://127.0.0.1:8080` |
| AEMS in Docker, sim-rtu on host | `172.17.0.1` | `http://172.17.0.1:8080` |
| Both in Docker (same network) | container name | `http://sim-rtu:8080` |

Update the device configs or NF driver config accordingly.

### NF Driver Not Polling

**Symptom:** No data appearing on the message bus.

1. Verify sim-rtu REST API is reachable from where AEMS runs:
   ```bash
   curl http://127.0.0.1:8080/api/v1/devices
   ```
2. Check the NF driver config URL matches sim-rtu's API address
3. Check `polling_interval` is set (default: 60 seconds)
4. Check AEMS logs for connection errors:
   ```bash
   tail -f log_platform.driver.log
   ```

### Registry CSV Mismatches

**Symptom:** Points read as null or wrong values.

The registry CSV files must match the points sim-rtu exposes. Use the same CSV files from `sim-rtu/configs/`:
- `schneider.csv` (60 points)
- `openstat.csv` (66 points)
- `dent.csv` (40 points)

Verify with the REST API:
```bash
curl http://127.0.0.1:8080/api/v1/devices/86254/points | python3 -m json.tool
```
