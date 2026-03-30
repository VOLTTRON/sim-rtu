"""Outdoor temperature profile generators.

Provides synthetic outdoor air temperature profiles for simulation:
- Sine wave: diurnal cycle with configurable mean, amplitude, and phase
- Constant: fixed temperature (useful for testing)
- CSV file: replay recorded weather data

The sine wave model uses:
    T(t) = mean + amplitude * sin(2*pi*(t - phase_offset) / 24)
where t is the hour of day.
"""

from __future__ import annotations

from abc import ABC, abstractmethod


class WeatherProfile(ABC):
    """Abstract base class for outdoor temperature profiles."""

    @abstractmethod
    def temperature_at(self, sim_time_hours: float) -> float:
        """Return outdoor air temperature at the given simulation time.

        Args:
            sim_time_hours: Simulation time in hours from start.

        Returns:
            Outdoor air temperature in degF.
        """


class SineWaveWeather(WeatherProfile):
    """Diurnal sine wave outdoor temperature profile.

    Args:
        mean: Average daily temperature (degF).
        amplitude: Temperature swing amplitude (degF).
        phase_offset: Hour of day for peak temperature.
    """

    def __init__(
        self,
        mean: float = 85.0,
        amplitude: float = 15.0,
        phase_offset: float = 14.0,
    ) -> None:
        self._mean = mean
        self._amplitude = amplitude
        self._phase_offset = phase_offset

    def temperature_at(self, sim_time_hours: float) -> float:
        """Return outdoor temperature at the given simulation time."""
        raise NotImplementedError("TODO: implement sine wave calculation")


class ConstantWeather(WeatherProfile):
    """Fixed outdoor temperature (useful for testing).

    Args:
        temperature: Constant outdoor temperature (degF).
    """

    def __init__(self, temperature: float = 85.0) -> None:
        self._temperature = temperature

    def temperature_at(self, sim_time_hours: float) -> float:
        """Return the fixed outdoor temperature."""
        return self._temperature
