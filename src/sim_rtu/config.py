"""Configuration loading and validation.

Parses the simulator YAML configuration file into typed Pydantic models.
"""

from __future__ import annotations

from pathlib import Path
from typing import Any

from pydantic import BaseModel, Field


class ThermalConfig(BaseModel):
    """Thermal model parameters for a thermostat device."""

    R: float = Field(description="Thermal resistance (degF/BTU-hr)")
    C: float = Field(description="Thermal capacitance (BTU/degF)")
    initial_zone_temp: float = Field(default=72.0)
    cooling_capacity_stage1: float = Field(default=18000)
    cooling_capacity_stage2: float = Field(default=18000)
    heating_capacity_stage1: float = Field(default=20000)
    heating_capacity_stage2: float = Field(default=20000)


class WeatherConfig(BaseModel):
    """Outdoor temperature profile configuration."""

    type: str = Field(default="sine_wave")
    mean: float = Field(default=85.0)
    amplitude: float = Field(default=15.0)
    phase_offset: float = Field(default=14.0)


class PowerConfig(BaseModel):
    """Power meter simulation parameters."""

    base_load_kw: float = Field(default=5.0)
    hvac_load_per_stage_kw: float = Field(default=3.0)


class DeviceConfig(BaseModel):
    """Configuration for a single simulated device."""

    name: str
    type: str
    device_id: int
    registry: str
    thermal: ThermalConfig | None = None
    weather: WeatherConfig | None = None
    power: PowerConfig | None = None


class BACnetConfig(BaseModel):
    """BACnet/IP server configuration."""

    enabled: bool = True
    interface: str = "0.0.0.0"
    port: int = 47808


class APIConfig(BaseModel):
    """REST API server configuration."""

    enabled: bool = True
    host: str = "0.0.0.0"
    port: int = 8080


class SimulatorConfig(BaseModel):
    """Top-level simulator configuration."""

    tick_interval: float = 1.0
    time_scale: float = 1.0


class AppConfig(BaseModel):
    """Root configuration model."""

    simulator: SimulatorConfig = Field(default_factory=SimulatorConfig)
    devices: list[DeviceConfig] = Field(default_factory=list)
    bacnet: BACnetConfig = Field(default_factory=BACnetConfig)
    api: APIConfig = Field(default_factory=APIConfig)


def load_config(path: str | Path) -> AppConfig:
    """Load and validate a YAML configuration file.

    Args:
        path: Path to the YAML configuration file.

    Returns:
        Validated AppConfig instance.

    Raises:
        FileNotFoundError: If the config file does not exist.
    """
    raise NotImplementedError("TODO: implement YAML loading")
