"""Thermostat controller with multi-stage heating/cooling logic.

Implements staged HVAC control based on:
- Zone temperature vs. setpoints (heating and cooling)
- Deadband
- Number of heating/cooling stages
- Anti-short-cycle timer
- Occupancy mode (occupied, unoccupied, standby)
"""

from __future__ import annotations

from dataclasses import dataclass
from enum import Enum


class HVACMode(Enum):
    """Active HVAC operating mode."""

    OFF = "off"
    HEATING_STAGE1 = "heating_stage1"
    HEATING_STAGE2 = "heating_stage2"
    COOLING_STAGE1 = "cooling_stage1"
    COOLING_STAGE2 = "cooling_stage2"


@dataclass
class ControllerState:
    """Current state of the thermostat controller."""

    mode: HVACMode = HVACMode.OFF
    heating_demand: float = 0.0
    cooling_demand: float = 0.0
    fan_status: bool = False
    stage1_cooling: bool = False
    stage2_cooling: bool = False
    stage1_heating: bool = False
    stage2_heating: bool = False


class ThermostatController:
    """Multi-stage thermostat controller.

    Args:
        num_cooling_stages: Number of cooling stages (1 or 2).
        num_heating_stages: Number of heating stages (1 or 2).
        deadband: Temperature deadband (degF).
        anti_short_cycle_minutes: Minimum compressor off time (minutes).
    """

    def __init__(
        self,
        num_cooling_stages: int = 2,
        num_heating_stages: int = 2,
        deadband: float = 3.0,
        anti_short_cycle_minutes: float = 2.0,
    ) -> None:
        self._num_cooling_stages = num_cooling_stages
        self._num_heating_stages = num_heating_stages
        self._deadband = deadband
        self._anti_short_cycle_minutes = anti_short_cycle_minutes

    def update(
        self,
        zone_temp: float,
        heating_setpoint: float,
        cooling_setpoint: float,
        dt_seconds: float,
    ) -> ControllerState:
        """Compute new controller state based on current zone temperature.

        Args:
            zone_temp: Current zone temperature (degF).
            heating_setpoint: Active heating setpoint (degF).
            cooling_setpoint: Active cooling setpoint (degF).
            dt_seconds: Time since last update (seconds).

        Returns:
            Updated ControllerState.
        """
        raise NotImplementedError("TODO: implement staging logic")
