"""Simulation loop orchestrating thermal model updates and point synchronization.

The engine runs at a configurable tick interval, advancing simulated time
and updating all device models each tick. It coordinates:
- Thermal model updates (zone temperature, HVAC staging)
- Power meter calculations
- Point store synchronization
- BACnet object value updates
"""

from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from sim_rtu.config import AppConfig


class SimulationEngine:
    """Main simulation loop.

    Args:
        config: The application configuration.
    """

    def __init__(self, config: AppConfig) -> None:
        self._config = config

    async def start(self) -> None:
        """Start the simulation loop.

        Initializes all device models and begins the tick loop.
        """
        raise NotImplementedError("TODO: implement simulation loop")

    async def stop(self) -> None:
        """Gracefully stop the simulation loop."""
        raise NotImplementedError("TODO: implement graceful shutdown")

    async def _tick(self, dt: float) -> None:
        """Execute one simulation tick.

        Args:
            dt: Time step in seconds (real time * time_scale).
        """
        raise NotImplementedError("TODO: implement single tick")
