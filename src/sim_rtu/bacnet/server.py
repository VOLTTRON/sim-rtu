"""BACnet/IP device server using bacpypes3.

Creates one BACnet device object per simulated device, with BACnet objects
mapped from the registry CSV. Listens on a configurable interface and port
for BACnet/IP requests (ReadProperty, WriteProperty, etc.).
"""

from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from sim_rtu.config import BACnetConfig
    from sim_rtu.points.store import PointStore


class BACnetServer:
    """BACnet/IP server exposing simulated device objects.

    Args:
        config: BACnet server configuration.
    """

    def __init__(self, config: BACnetConfig) -> None:
        self._config = config

    async def start(self, devices: dict[int, PointStore]) -> None:
        """Start the BACnet/IP server with the given device stores.

        Args:
            devices: Mapping of BACnet device ID to PointStore.
        """
        raise NotImplementedError("TODO: implement BACnet server startup")

    async def stop(self) -> None:
        """Stop the BACnet/IP server."""
        raise NotImplementedError("TODO: implement BACnet server shutdown")
