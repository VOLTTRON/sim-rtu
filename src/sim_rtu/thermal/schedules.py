"""Occupancy schedule handling for thermostat devices.

Determines occupied/unoccupied/standby state based on:
- Day-of-week schedules with start/end times
- Override commands
- Standby timers (PIR/motion-based)
"""

from __future__ import annotations

from enum import Enum


class OccupancyState(Enum):
    """Occupancy states for a thermostat zone."""

    OCCUPIED = "occupied"
    UNOCCUPIED = "unoccupied"
    STANDBY = "standby"


class OccupancySchedule:
    """Day-of-week occupancy schedule.

    Args:
        schedule: Dict mapping day name to either:
            - {"start": "HH:MM", "end": "HH:MM"} for occupied periods
            - "always_off" for fully unoccupied days
            - "always_on" for fully occupied days
    """

    def __init__(self, schedule: dict | None = None) -> None:
        self._schedule = schedule or {}

    def state_at(self, day_name: str, hour: float) -> OccupancyState:
        """Return occupancy state for the given day and hour.

        Args:
            day_name: Day of week (e.g., "Monday", "Tuesday").
            hour: Hour of day as float (e.g., 14.5 = 2:30 PM).

        Returns:
            OccupancyState for the given time.
        """
        raise NotImplementedError("TODO: implement schedule lookup")
