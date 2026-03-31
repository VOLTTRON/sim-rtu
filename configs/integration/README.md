# AEMS Integration Configs

Driver configuration files for connecting the AEMS platform driver (aems-lib-fastapi)
to sim-rtu's BACnet/IP server.

## Files

- `aems-driver-config.json` - Schneider RTU thermostat (device_id: 86254)
- `aems-openstat-config.json` - OpenStat RTU thermostat (device_id: 86255)
- `aems-dent-config.json` - DENT power meter (device_id: 20001)

## Usage

These configs are used by the VOLTTRON platform driver's config store.
The `registry_config` field references the CSV registry files that define
the BACnet point mappings. The same CSV files used by sim-rtu are compatible
with the AEMS platform driver.

## Connection Parameters

- **BACnet/IP address**: 127.0.0.1:47808 (default sim-rtu port)
- **Protocol**: BACnet/IP over UDP
- **Supported services**: WhoIs/IAm, ReadProperty, ReadPropertyMultiple, WriteProperty
