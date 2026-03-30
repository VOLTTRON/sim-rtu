"""First-order RC thermal model for zone temperature simulation.

Models the zone as a single thermal node with:
- R: thermal resistance between outdoor air and zone (degF / BTU-hr)
- C: thermal capacitance of the zone (BTU / degF)
- Q_hvac: heating or cooling power input (BTU/hr, negative for cooling)

Governing equation:
    C * dT_zone/dt = (T_outdoor - T_zone) / R + Q_hvac
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass
class ThermalModel:
    """First-order RC thermal model for a single zone.

    Attributes:
        R: Thermal resistance (degF-hr / BTU).
        C: Thermal capacitance (BTU / degF).
        zone_temp: Current zone temperature (degF).
    """

    R: float
    C: float
    zone_temp: float

    def step(self, outdoor_temp: float, q_hvac: float, dt_hours: float) -> float:
        """Advance the model by one time step.

        Args:
            outdoor_temp: Current outdoor air temperature (degF).
            q_hvac: HVAC heat input (BTU/hr). Positive = heating, negative = cooling.
            dt_hours: Time step size in hours.

        Returns:
            Updated zone temperature (degF).
        """
        raise NotImplementedError("TODO: implement RC thermal model step")
