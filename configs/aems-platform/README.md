# AEMS Platform Config Package for sim-rtu

Complete configuration package for connecting the AEMS platform (aems-lib-fastapi) to sim-rtu's BACnet/IP simulator.

## What's Included

```
aems-platform/
├── platform-driver/
│   ├── config                          # Platform driver main config
│   ├── devices/SIM/RTU/
│   │   ├── Schneider                   # Schneider SE8600 RTU thermostat (device_id: 86254)
│   │   ├── OpenStat                    # OpenStat RTU thermostat (device_id: 86255)
│   │   └── DENT                        # DENT PowerScout meter (device_id: 20001)
│   └── registry_configs/
│       ├── schneider.csv               # Schneider BACnet point registry (60 points)
│       ├── openstat.csv                # OpenStat BACnet point registry (66 points)
│       └── dent.csv                    # DENT meter point registry (40 points)
├── bacnet-proxy/
│   └── config                          # BACnet proxy agent config
├── setup.sh                            # Automated installer
└── README.md                           # This file
```

## Prerequisites

1. **sim-rtu** built and running (BACnet on 127.0.0.1:47808)
2. **aems-lib-fastapi** installed (`pip install -e .` from the aems-lib-fastapi repo)
3. **Python packages** for BACnet support:
   - `bacpypes==0.16.7` (must be this exact version)
   - `pyasyncore` (required for Python 3.12+)
   - `pyasynchat` (required for Python 3.12+)

```bash
pip install bacpypes==0.16.7 pyasyncore pyasynchat
```

## Setup

### Step 1: Start sim-rtu

```bash
cd /path/to/sim-rtu
make build
./bin/sim-rtu --config configs/default.yml --log-level INFO
```

Or use the convenience script:

```bash
./scripts/start-all.sh
```

### Step 2: Install AEMS configs

```bash
# Default location (~/.volttron_home)
./configs/aems-platform/setup.sh

# Custom location
./configs/aems-platform/setup.sh /path/to/my/volttron_home
```

### Step 3: Start AEMS

```bash
aems-server --host 127.0.0.1 --port 8000 --volttron-home /path/to/volttron_home
```

## Verifying Connectivity

### Check sim-rtu is running

```bash
# REST API status
curl http://127.0.0.1:8080/api/v1/status

# List simulated devices
curl http://127.0.0.1:8080/api/v1/devices
```

### Read points via AEMS API

```bash
# Read Schneider zone temperature
curl http://127.0.0.1:8000/devices/SIM/RTU/Schneider/ZoneTemperature

# Read all Schneider points
curl http://127.0.0.1:8000/devices/SIM/RTU/Schneider

# Read OpenStat zone temperature
curl http://127.0.0.1:8000/devices/SIM/RTU/OpenStat/ZoneTemperature

# Read DENT whole building power
curl http://127.0.0.1:8000/devices/SIM/RTU/DENT/WholeBuildingPower
```

### Write points via AEMS API

```bash
# Set Schneider occupied cooling setpoint
curl -X PUT http://127.0.0.1:8000/devices/SIM/RTU/Schneider/OccupiedCoolingSetPoint \
     -H "Content-Type: application/json" \
     -d '{"value": 73.0, "priority": 16}'

# Set OpenStat occupancy command (1=Occupied, 2=Unoccupied)
curl -X PUT http://127.0.0.1:8000/devices/SIM/RTU/OpenStat/OccupancyCommand \
     -H "Content-Type: application/json" \
     -d '{"value": 1, "priority": 8}'
```

## Device Summary

| Device | Type | Device ID | BACnet Port | Key Points |
|--------|------|-----------|-------------|------------|
| Schneider | Thermostat (SE8600) | 86254 | 47808 | ZoneTemperature, OccupiedCoolingSetPoint, OccupiedHeatingSetPoint, OccupancyCommand |
| OpenStat | Thermostat | 86255 | 47808 | ZoneTemperature, OccupiedCoolingSetPoint, OccupiedHeatingSetPoint, OccupancyCommand |
| DENT | Power Meter | 20001 | 47808 | WholeBuildingPower, Current, VoltageLL, PowerFactor |

## Configuration Details

### Platform Driver Config

- **Scrape interval**: 50ms between device polls
- **Publish mode**: Breadth-first-all + depth-first (all topics published)
- **Device poll interval**: 60 seconds per device

### BACnet Proxy Config

- **Listen address**: 127.0.0.1/24 (localhost subnet)
- **APDU length**: 1024 bytes
- **Object ID**: 599
- **Max per request**: 24 properties (ReadPropertyMultiple batch size)
- **Write priority**: 16 (default, lowest BACnet priority)
- **Timeout**: 10 seconds per request

## Troubleshooting

### "No response from device"

- Verify sim-rtu is running: `curl http://127.0.0.1:8080/api/v1/status`
- Check BACnet port is not blocked: `ss -uln | grep 47808`
- Ensure only one BACnet listener on port 47808 (sim-rtu uses this port)
- The BACnet proxy must use a different port (it binds dynamically by default)

### "CSV registry not found"

- The registry_config paths use `config://` prefix which resolves to the config store
- Run `setup.sh` again to ensure CSV files and `.metadata` files are in place
- Each CSV needs a corresponding `.csv.metadata` file with `{"type": "csv"}`

### "bacpypes import error"

- Must use `bacpypes==0.16.7` (not bacpypes3)
- For Python 3.12+, install `pyasyncore` and `pyasynchat`:
  ```bash
  pip install bacpypes==0.16.7 pyasyncore pyasynchat
  ```

### "Address already in use" (UDP 47808)

- Only one process can bind UDP port 47808
- sim-rtu uses 47808 for its BACnet server
- The BACnet proxy should NOT specify port 47808; it picks a random high port
- If you see this error, check for stale processes: `lsof -i UDP:47808`

### Device config changes not taking effect

- AEMS reads configs from the config store at startup
- After running `setup.sh`, restart the AEMS server
- Check that `.metadata` files exist alongside each config file
