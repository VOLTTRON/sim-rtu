# AEMS Platform Config Package

Configuration files for connecting aems-lib-fastapi to sim-rtu. Supports both BACnet (legacy) and NF REST driver modes.

For full setup instructions, see **[docs/setup-guide.md](../../docs/setup-guide.md)**.

## Contents

```
aems-platform/
├── platform-driver/         # Legacy BACnet driver configs
│   ├── config               # Platform driver main config
│   ├── devices/SIM/RTU/     # Per-device configs (Schneider, OpenStat, DENT)
│   └── registry_configs/    # BACnet point registry CSVs
├── bacnet-proxy/
│   └── config               # BACnet proxy agent config
├── nf-driver/
│   ├── config               # NF driver config (URL + device list)
│   └── setup.sh             # NF config installer
├── setup.sh                 # BACnet config installer
└── README.md
```

## Quick Setup

### NF Driver (Recommended)

```bash
./configs/aems-platform/nf-driver/setup.sh /path/to/aems-lib-fastapi/.volttron_home
```

Or use the helper script:

```bash
./scripts/switch-to-nf.sh /path/to/aems-lib-fastapi
```

### Legacy BACnet Driver

```bash
./configs/aems-platform/setup.sh /path/to/aems-lib-fastapi/.volttron_home
```

Or use the helper script:

```bash
./scripts/switch-to-bacnet.sh /path/to/aems-lib-fastapi
```

## Device Summary

| Device | Device ID | Points | Config File |
|--------|-----------|--------|-------------|
| Schneider SE8600 | 86254 | 59 | `devices/SIM/RTU/Schneider` |
| OpenStat | 86255 | 65 | `devices/SIM/RTU/OpenStat` |
| DENT PowerScout | 20001 | 38 | `devices/SIM/RTU/DENT` |

## Docker Note

When AEMS runs in Docker and sim-rtu runs on the host, change device addresses to `172.17.0.1` (docker0 bridge):

- BACnet: `"device_address": "172.17.0.1"` in each device config
- NF: `url: http://172.17.0.1:8080` in NF driver config
