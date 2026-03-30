# sim-rtu

Simulated RTU devices (thermostat, power meter) for VOLTTRON platform driver testing.

## Supported Device Types

- **Schneider thermostat** -- 61-point BACnet registry (analogValue, binaryOutput, multiStateValue)
- **OpenStat thermostat** -- 67-point BACnet registry with active flag support
- **DENT power meter** -- 40-point read-only BACnet registry (current, voltage, power, harmonics)

## Quick Start

```bash
pip install -e ".[dev]"
sim-rtu --config configs/default.yml
```

The simulator exposes:
- **BACnet/IP server** on port 47808 (one device object per configured device)
- **REST API** on port 8080 for inspection and control

## Project Status

Early development -- registry parser and point store are implemented; thermal model, BACnet server, REST API, and simulation engine are stubbed.
