# sim-rtu

Simulated RTU (Rooftop Unit) devices for VOLTTRON/AEMS testing. Exposes BACnet/IP and REST APIs to virtual HVAC equipment and a power meter.

## Simulated Devices

| Device | Type | Device ID | Points |
|--------|------|-----------|--------|
| Schneider SE8600 | Thermostat | 86254 | 59 |
| OpenStat | Thermostat | 86255 | 65 |
| DENT PowerScout | Power Meter | 20001 | 38 |

## Quick Start

```bash
make build
./bin/sim-rtu --config configs/default.yml
```

Or: `make run` / `docker compose up`

## Interfaces

| Interface | Protocol | Port | Use Case |
|-----------|----------|------|----------|
| BACnet/IP | UDP | 47808 | Legacy Platform Driver |
| NF REST | HTTP POST | 8080 | Normal Framework Driver |
| sim-rtu REST | HTTP | 8080 | Direct device inspection/control |

## AEMS Integration

See **[docs/setup-guide.md](docs/setup-guide.md)** for sim-rtu build/run instructions.

For full AEMS platform setup (connecting to aems-lib-fastapi with any device target), see the **[AEMS Platform Setup Guide](https://github.com/VOLTTRON/aems-lib-fastapi/blob/develop/docs/SETUP_GUIDE.md)**.

## Driver Modes

Two driver modes for AEMS platform integration:

- **NF REST** (recommended) — simpler setup, no BACnet proxy needed
- **Legacy BACnet** — full BACnet/IP protocol, requires BACnet proxy agent

See [docs/driver-switching.md](docs/driver-switching.md) for switching guide.

Helper scripts:
```bash
./scripts/switch-to-nf.sh /path/to/aems-lib-fastapi
./scripts/switch-to-bacnet.sh /path/to/aems-lib-fastapi
./scripts/start-all.sh
```

## API Endpoints

### sim-rtu REST API (port 8080)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/devices` | List devices |
| `GET` | `/api/v1/devices/{id}/points` | All points for a device |
| `GET` | `/api/v1/devices/{id}/points/{name}` | Read a single point |
| `PUT` | `/api/v1/devices/{id}/points/{name}` | Write a point value |
| `GET` | `/api/v1/status` | Simulation status |
| `POST` | `/api/v1/weather` | Override outdoor temperature |

### NF API (port 8080)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v2/bacnet/confirmed-service` | NF ReadPropertyMultiple / WriteProperty |

## BACnet Services

WhoIs/IAm, ReadProperty, ReadPropertyMultiple, WriteProperty with 16-level priority arrays.

## Development

```bash
make build                # Build binary
make run                  # Build and run
make test                 # Unit tests with coverage
make coverage             # Coverage report
./scripts/test-integration.sh  # Full integration tests
```

## Configuration

Config files in `configs/`:
- `default.yml` — main simulator config (devices, thermal model, weather, ports)
- `schneider.csv`, `openstat.csv`, `dent.csv` — BACnet point registries
- `aems-platform/` — AEMS config store files ([README](configs/aems-platform/README.md))

Bind to `0.0.0.0` in `default.yml` for Docker/remote access:
```yaml
api:
  host: "0.0.0.0"
bacnet:
  interface: "0.0.0.0"
```
