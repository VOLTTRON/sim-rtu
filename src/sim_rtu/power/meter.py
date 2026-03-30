"""Power meter simulation for DENT-style three-phase power meters.

Computes simulated electrical measurements:
- Per-phase and total current, voltage, power
- Apparent power, reactive power, power factor
- Harmonic distortion
- Phase angles

Power consumption is driven by:
- Base building load (constant)
- HVAC load (proportional to active cooling/heating stages)
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class MeterReading:
    """Snapshot of all power meter measurements."""

    # Per-phase currents (A)
    current_a: float = 0.0
    current_b: float = 0.0
    current_c: float = 0.0

    # Per-phase voltages L-N (V)
    voltage_an: float = 120.0
    voltage_bn: float = 120.0
    voltage_cn: float = 120.0

    # Total power (kW)
    total_power_kw: float = 0.0

    # Per-phase power (kW)
    power_a: float = 0.0
    power_b: float = 0.0
    power_c: float = 0.0

    # Power factor
    power_factor: float = 1.0

    # Frequency (Hz)
    frequency: float = 60.0


class PowerMeterSimulator:
    """Simulates a three-phase power meter.

    Args:
        base_load_kw: Constant base building load (kW).
        hvac_load_per_stage_kw: Additional load per active HVAC stage (kW).
        nominal_voltage: Nominal line-to-neutral voltage (V).
        nominal_frequency: Nominal line frequency (Hz).
    """

    def __init__(
        self,
        base_load_kw: float = 5.0,
        hvac_load_per_stage_kw: float = 3.0,
        nominal_voltage: float = 120.0,
        nominal_frequency: float = 60.0,
    ) -> None:
        self._base_load_kw = base_load_kw
        self._hvac_load_per_stage_kw = hvac_load_per_stage_kw
        self._nominal_voltage = nominal_voltage
        self._nominal_frequency = nominal_frequency

    def compute(self, active_hvac_stages: int = 0) -> MeterReading:
        """Compute power meter readings for the current HVAC state.

        Args:
            active_hvac_stages: Number of active HVAC compressor stages (0-4).

        Returns:
            MeterReading with all computed electrical measurements.
        """
        raise NotImplementedError("TODO: implement power calculations")
