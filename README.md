# sim-rtu

Simulated RTU (Rooftop Unit) devices for VOLTTRON. Provides BACnet/IP and REST API interfaces to virtual HVAC equipment for testing and development.

## Supported Devices

- **Schneider thermostat** — 60 BACnet points (analog/binary/multi-state)
- **OpenStat thermostat** — 66 BACnet points with active/inactive filtering
- **DENT power meter** — 39 BACnet points (three-phase electrical measurements)

## Features

- RC lumped-capacitance thermal model with multi-stage heating/cooling
- Configurable weather profiles (static, sine wave)
- Occupancy scheduling with deadband and anti-short-cycle protection
- Three-phase power meter simulation with realistic noise
- 16-level BACnet priority arrays for writable points
- REST API for point inspection and control
- BACnet/IP server (stub — full implementation pending)
- Single static binary, no runtime dependencies

## Quick Start

```bash
make run
```

## Docker

```bash
docker compose up
```

## Configuration

Edit `configs/default.yml` to add/modify devices, thermal parameters, and weather profiles.

## API

```
GET  /api/v1/devices                          — list devices
GET  /api/v1/devices/{id}/points              — all points for a device
GET  /api/v1/devices/{id}/points/{name}       — read a single point
PUT  /api/v1/devices/{id}/points/{name}       — write a point value
GET  /api/v1/status                           — simulation status
POST /api/v1/weather                          — override outdoor temperature
```

## Testing

```bash
make test
```

### Integration Tests

Run the full integration test suite (builds sim-rtu, starts it, runs BACnet and REST API tests):

```bash
./scripts/test-integration.sh
```

Run only the REST API smoke tests (no Go integration tests):

```bash
./scripts/test-integration.sh --api-only
```

Run the Go integration tests directly:

```bash
go test -tags integration -v -timeout 60s ./tests/integration/
```

## Integration with VOLTTRON/AEMS

sim-rtu acts as a BACnet/IP device simulator that the AEMS platform driver
(aems-lib-fastapi) can connect to for testing and development.

### BACnet Connection Parameters

| Parameter | Value |
|-----------|-------|
| Protocol | BACnet/IP over UDP |
| Default port | 47808 |
| Schneider RTU device ID | 86254 |
| OpenStat RTU device ID | 86255 |
| DENT meter device ID | 20001 |

### Configuring the AEMS Platform Driver

1. Start sim-rtu:

   ```bash
   make run
   # or
   docker compose up
   ```

2. Copy the integration config files to the AEMS platform driver config store.
   Pre-built configs are in `configs/integration/`:

   - `aems-driver-config.json` — Schneider RTU (device 86254)
   - `aems-openstat-config.json` — OpenStat RTU (device 86255)
   - `aems-dent-config.json` — DENT meter (device 20001)

3. The `registry_config` in each driver config references the CSV registry
   files. The same CSV files used by sim-rtu (`configs/schneider.csv`,
   `configs/openstat.csv`, `configs/dent.csv`) are compatible with the
   AEMS platform driver.

### Verifying Connectivity

Send a BACnet WhoIs and verify IAm responses for all three devices:

```bash
# Using the REST API to confirm sim-rtu is running:
curl http://localhost:8080/api/v1/devices
curl http://localhost:8080/api/v1/devices/86254/points/ZoneTemperature
```

### Docker Compose (Integration)

Run sim-rtu in Docker with health checks:

```bash
docker compose -f docker-compose.integration.yml up
```

### Supported BACnet Services

- **WhoIs / IAm** — device discovery
- **ReadProperty** — read individual point values
- **ReadPropertyMultiple** — batch read multiple properties
- **WriteProperty** — write setpoints and control outputs with priority
