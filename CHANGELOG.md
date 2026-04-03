# Changelog

## [0.1.0] - 2026-04-02

### Added
- Initial Go implementation with three simulated devices
- BACnet/IP server (WhoIs, ReadProperty, WriteProperty, ReadPropertyMultiple)
- Normal Framework compatible REST API (`/api/v2/bacnet/confirmed-service`)
- sim-rtu REST API for direct device inspection and control
- RC thermal model with multi-stage HVAC controller
- Three-phase power meter simulation
- AEMS platform integration configs (BACnet + NF driver)
- Driver switching guide and helper scripts
- Docker support (Dockerfile, docker-compose)
- Integration test suite

### Devices
- Schneider SE8600 thermostat (59 BACnet points, device ID 86254)
- OpenStat thermostat (65 BACnet points, device ID 86255)
- DENT PowerScout meter (38 BACnet points, device ID 20001)
