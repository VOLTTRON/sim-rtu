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
