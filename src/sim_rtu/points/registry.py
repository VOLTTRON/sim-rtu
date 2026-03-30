"""Parse VOLTTRON platform driver CSV registries into point definitions.

Supports three registry formats:
- Schneider thermostat (10 columns)
- OpenStat thermostat (11 columns, extra 'active' column)
- DENT power meter (10 columns, all read-only)

All three share the same first 10 columns:
    Reference Point Name, Volttron Point Name, Units, Unit Details,
    BACnet Object Type, Property, Writable, Index, Write Priority, Notes
"""

from __future__ import annotations

import csv
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class PointDefinition:
    """Immutable definition of a single BACnet point parsed from a registry CSV."""

    reference_name: str
    volttron_name: str
    units: str
    bacnet_object_type: str
    property_name: str
    writable: bool
    index: int
    write_priority: int | None
    default_value: float | None
    notes: str
    active: bool = True

    @property
    def object_key(self) -> tuple[str, int]:
        """Return the (bacnet_object_type, index) lookup key."""
        return (self.bacnet_object_type, self.index)


def _parse_bool(value: str) -> bool:
    """Parse a boolean string from CSV (TRUE/FALSE)."""
    return value.strip().upper() == "TRUE"


def _parse_default_value(unit_details: str) -> float | None:
    """Extract default value from the Unit Details column.

    Unit Details may contain strings like:
        '(default 72.0)'
        '0-1 (default 0)'
        'State count: 3 (default 2)'
        ''  (empty)

    Returns the numeric default value or None.
    """
    if not unit_details:
        return None

    marker = "(default "
    start = unit_details.find(marker)
    if start == -1:
        return None

    start += len(marker)
    end = unit_details.find(")", start)
    if end == -1:
        return None

    try:
        return float(unit_details[start:end])
    except ValueError:
        return None


def _parse_write_priority(value: str) -> int | None:
    """Parse the write priority column. Returns None if empty or invalid."""
    stripped = value.strip()
    if not stripped:
        return None
    try:
        return int(stripped)
    except ValueError:
        return None


def _parse_index(value: str) -> int:
    """Parse the index column."""
    return int(value.strip())


def parse_registry(path: str | Path) -> list[PointDefinition]:
    """Parse a VOLTTRON platform driver CSV registry file.

    Args:
        path: Path to the CSV registry file.

    Returns:
        List of PointDefinition objects, one per row in the CSV.

    Raises:
        FileNotFoundError: If the registry file does not exist.
        ValueError: If a row has an unexpected number of columns.
    """
    path = Path(path)
    points: list[PointDefinition] = []

    with path.open(newline="", encoding="utf-8") as f:
        reader = csv.reader(f)
        header = next(reader)
        num_cols = len(header)

        # Detect whether the 'active' column is present (OpenStat format)
        has_active_col = (
            num_cols >= 11
            and header[10].strip().lower() == "active"
        )

        for row_num, row in enumerate(reader, start=2):
            if not row or all(cell.strip() == "" for cell in row):
                continue

            if len(row) < 10:
                raise ValueError(
                    f"{path.name}:{row_num}: expected at least 10 columns, "
                    f"got {len(row)}"
                )

            reference_name = row[0].strip()
            volttron_name = row[1].strip()
            units = row[2].strip()
            unit_details = row[3].strip()
            bacnet_object_type = row[4].strip()
            property_name = row[5].strip()
            writable = _parse_bool(row[6])
            index = _parse_index(row[7])
            write_priority = _parse_write_priority(row[8])
            notes = row[9].strip() if len(row) > 9 else ""

            default_value = _parse_default_value(unit_details)

            active = True
            if has_active_col and len(row) > 10:
                active = _parse_bool(row[10])

            point = PointDefinition(
                reference_name=reference_name,
                volttron_name=volttron_name,
                units=units,
                bacnet_object_type=bacnet_object_type,
                property_name=property_name,
                writable=writable,
                index=index,
                write_priority=write_priority if writable else None,
                default_value=default_value,
                notes=notes,
                active=active,
            )
            points.append(point)

    return points
