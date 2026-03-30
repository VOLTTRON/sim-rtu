"""Map registry point definitions to bacpypes3 BACnet objects.

Creates appropriate BACnet object instances (AnalogValueObject,
BinaryOutputObject, MultiStateValueObject, etc.) from PointDefinition
records, wiring their presentValue to the PointStore.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from sim_rtu.points.registry import PointDefinition


def create_bacnet_objects(
    definitions: list[PointDefinition],
) -> list:
    """Create bacpypes3 BACnet objects from point definitions.

    Args:
        definitions: List of parsed registry point definitions.

    Returns:
        List of bacpypes3 object instances ready to attach to a device.
    """
    raise NotImplementedError("TODO: implement BACnet object creation")
