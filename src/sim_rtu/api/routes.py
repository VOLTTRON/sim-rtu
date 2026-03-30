"""REST API route definitions for the simulator.

Endpoints:
- GET  /devices                    - List all devices
- GET  /devices/{device_id}/points - List all points for a device
- GET  /devices/{device_id}/points/{point_name} - Read a point value
- PUT  /devices/{device_id}/points/{point_name} - Write a point value
- GET  /simulation/status          - Get simulation status
- POST /simulation/pause           - Pause simulation
- POST /simulation/resume          - Resume simulation
"""

from __future__ import annotations

from fastapi import APIRouter


def create_router() -> APIRouter:
    """Create the API router with all endpoint definitions.

    Returns:
        Configured APIRouter.
    """
    router = APIRouter()

    @router.get("/devices")
    async def list_devices() -> dict:
        """List all simulated devices."""
        raise NotImplementedError("TODO: implement device listing")

    @router.get("/devices/{device_id}/points")
    async def list_points(device_id: int) -> dict:
        """List all points for a device."""
        raise NotImplementedError("TODO: implement point listing")

    @router.get("/devices/{device_id}/points/{point_name}")
    async def read_point(device_id: int, point_name: str) -> dict:
        """Read a point value."""
        raise NotImplementedError("TODO: implement point read")

    @router.put("/devices/{device_id}/points/{point_name}")
    async def write_point(device_id: int, point_name: str) -> dict:
        """Write a point value."""
        raise NotImplementedError("TODO: implement point write")

    @router.get("/simulation/status")
    async def simulation_status() -> dict:
        """Get simulation status."""
        raise NotImplementedError("TODO: implement status endpoint")

    return router
