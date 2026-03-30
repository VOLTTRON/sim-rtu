"""BACnet 16-level priority array for writable points.

BACnet defines 16 priority levels (1 = highest, 16 = lowest).
The active value is the value written at the highest (lowest-numbered)
priority level. Relinquishing a priority clears that slot. If all
slots are empty, the active value falls back to a configurable default.
"""

from __future__ import annotations

from dataclasses import dataclass, field

_NUM_PRIORITIES = 16


@dataclass
class PriorityArray:
    """A 16-level BACnet priority array.

    Attributes:
        default_value: Fallback value when all priority slots are empty.
    """

    default_value: float | None = None
    _slots: list[float | None] = field(
        default_factory=lambda: [None] * _NUM_PRIORITIES,
        init=False,
        repr=False,
    )

    def write(self, priority: int, value: float) -> None:
        """Write a value at the given priority level (1-16).

        Args:
            priority: BACnet priority level (1 = highest, 16 = lowest).
            value: The value to write.

        Raises:
            ValueError: If priority is not in range 1-16.
        """
        self._validate_priority(priority)
        self._slots[priority - 1] = value

    def relinquish(self, priority: int) -> None:
        """Clear the value at the given priority level.

        Args:
            priority: BACnet priority level (1-16).

        Raises:
            ValueError: If priority is not in range 1-16.
        """
        self._validate_priority(priority)
        self._slots[priority - 1] = None

    def active_value(self) -> float | None:
        """Return the value at the highest (lowest-numbered) occupied priority.

        Returns the default_value if all priority slots are empty.
        """
        for slot_value in self._slots:
            if slot_value is not None:
                return slot_value
        return self.default_value

    def active_priority(self) -> int | None:
        """Return the priority level of the active value, or None if all empty."""
        for i, slot_value in enumerate(self._slots):
            if slot_value is not None:
                return i + 1
        return None

    def to_dict(self) -> dict[int, float | None]:
        """Return a dict mapping priority level (1-16) to slot value."""
        return {i + 1: v for i, v in enumerate(self._slots)}

    @staticmethod
    def _validate_priority(priority: int) -> None:
        if not (1 <= priority <= _NUM_PRIORITIES):
            raise ValueError(
                f"Priority must be 1-{_NUM_PRIORITIES}, got {priority}"
            )
