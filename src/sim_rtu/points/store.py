"""PointStore: central value registry for simulated BACnet points.

Provides thread-safe read/write access by either:
- volttron_name (string key)
- BACnet object key (bacnet_object_type, index) tuple

Writable points use a PriorityArray; read-only points store a single value.
"""

from __future__ import annotations

import asyncio
from dataclasses import dataclass, field

from sim_rtu.points.priority_array import PriorityArray
from sim_rtu.points.registry import PointDefinition


@dataclass
class _PointSlot:
    """Internal storage for a single point's value and metadata."""

    definition: PointDefinition
    priority_array: PriorityArray | None
    value: float | None


@dataclass
class PointStore:
    """Central value registry for all points in a simulated device.

    Initialize from a list of PointDefinition objects. Supports concurrent
    access via an asyncio.Lock.
    """

    _by_name: dict[str, _PointSlot] = field(default_factory=dict, init=False)
    _by_object_key: dict[tuple[str, int], _PointSlot] = field(
        default_factory=dict, init=False
    )
    _lock: asyncio.Lock = field(default_factory=asyncio.Lock, init=False)

    @classmethod
    def from_definitions(cls, definitions: list[PointDefinition]) -> PointStore:
        """Create a PointStore from a list of point definitions.

        Writable points get a PriorityArray initialized with the default value
        at priority 16. Read-only points store the default value directly.
        """
        store = cls()

        for defn in definitions:
            if defn.writable:
                pa = PriorityArray(default_value=defn.default_value)
                if defn.default_value is not None and defn.write_priority is not None:
                    pa.write(defn.write_priority, defn.default_value)
                slot = _PointSlot(
                    definition=defn,
                    priority_array=pa,
                    value=None,
                )
            else:
                slot = _PointSlot(
                    definition=defn,
                    priority_array=None,
                    value=defn.default_value,
                )

            store._by_name[defn.volttron_name] = slot
            store._by_object_key[defn.object_key] = slot

        return store

    async def read_by_name(self, volttron_name: str) -> float | None:
        """Read the active value of a point by its Volttron name.

        Raises:
            KeyError: If the point name is not found.
        """
        async with self._lock:
            slot = self._by_name[volttron_name]
            return self._active_value(slot)

    async def read_by_object_key(
        self, bacnet_object_type: str, index: int
    ) -> float | None:
        """Read the active value of a point by its BACnet object type and index.

        Raises:
            KeyError: If the (object_type, index) pair is not found.
        """
        async with self._lock:
            slot = self._by_object_key[(bacnet_object_type, index)]
            return self._active_value(slot)

    async def write_by_name(
        self,
        volttron_name: str,
        value: float,
        priority: int = 16,
    ) -> None:
        """Write a value to a writable point by its Volttron name.

        Raises:
            KeyError: If the point name is not found.
            TypeError: If the point is not writable.
        """
        async with self._lock:
            slot = self._by_name[volttron_name]
            self._write_slot(slot, value, priority)

    async def write_by_object_key(
        self,
        bacnet_object_type: str,
        index: int,
        value: float,
        priority: int = 16,
    ) -> None:
        """Write a value to a writable point by its BACnet object key.

        Raises:
            KeyError: If the (object_type, index) pair is not found.
            TypeError: If the point is not writable.
        """
        async with self._lock:
            slot = self._by_object_key[(bacnet_object_type, index)]
            self._write_slot(slot, value, priority)

    async def set_internal(self, volttron_name: str, value: float) -> None:
        """Set a point value directly, bypassing the priority array.

        Used by the simulation engine to update read-only sensor values
        (e.g., zone temperature, outdoor temperature).

        Raises:
            KeyError: If the point name is not found.
        """
        async with self._lock:
            slot = self._by_name[volttron_name]
            if slot.priority_array is not None:
                slot.priority_array = PriorityArray(default_value=value)
            slot.value = value

    async def read_all(self) -> dict[str, float | None]:
        """Return a snapshot of all point names and their active values."""
        async with self._lock:
            return {
                name: self._active_value(slot)
                for name, slot in self._by_name.items()
            }

    @property
    def point_names(self) -> list[str]:
        """Return all registered Volttron point names."""
        return list(self._by_name.keys())

    @property
    def definitions(self) -> list[PointDefinition]:
        """Return all registered point definitions."""
        return [slot.definition for slot in self._by_name.values()]

    @staticmethod
    def _active_value(slot: _PointSlot) -> float | None:
        if slot.priority_array is not None:
            return slot.priority_array.active_value()
        return slot.value

    @staticmethod
    def _write_slot(slot: _PointSlot, value: float, priority: int) -> None:
        if slot.priority_array is None:
            raise TypeError(
                f"Point '{slot.definition.volttron_name}' is not writable"
            )
        slot.priority_array.write(priority, value)
