"""Tests for the BACnet PriorityArray."""

from __future__ import annotations

import pytest

from sim_rtu.points.priority_array import PriorityArray


class TestPriorityArray:
    def test_empty_array_returns_default(self) -> None:
        pa = PriorityArray(default_value=72.0)
        assert pa.active_value() == 72.0

    def test_empty_array_no_default_returns_none(self) -> None:
        pa = PriorityArray()
        assert pa.active_value() is None

    def test_write_and_read(self) -> None:
        pa = PriorityArray()
        pa.write(8, 68.0)
        assert pa.active_value() == 68.0

    def test_higher_priority_wins(self) -> None:
        pa = PriorityArray()
        pa.write(16, 72.0)
        pa.write(8, 68.0)
        assert pa.active_value() == 68.0

    def test_relinquish_falls_to_lower(self) -> None:
        pa = PriorityArray()
        pa.write(16, 72.0)
        pa.write(8, 68.0)
        pa.relinquish(8)
        assert pa.active_value() == 72.0

    def test_relinquish_all_falls_to_default(self) -> None:
        pa = PriorityArray(default_value=70.0)
        pa.write(8, 68.0)
        pa.relinquish(8)
        assert pa.active_value() == 70.0

    def test_active_priority(self) -> None:
        pa = PriorityArray()
        pa.write(16, 72.0)
        pa.write(8, 68.0)
        assert pa.active_priority() == 8

    def test_active_priority_empty(self) -> None:
        pa = PriorityArray()
        assert pa.active_priority() is None

    def test_invalid_priority_raises(self) -> None:
        pa = PriorityArray()
        with pytest.raises(ValueError):
            pa.write(0, 72.0)
        with pytest.raises(ValueError):
            pa.write(17, 72.0)
        with pytest.raises(ValueError):
            pa.relinquish(0)

    def test_to_dict(self) -> None:
        pa = PriorityArray()
        pa.write(8, 68.0)
        pa.write(16, 72.0)
        d = pa.to_dict()
        assert d[8] == 68.0
        assert d[16] == 72.0
        assert d[1] is None

    def test_overwrite_same_priority(self) -> None:
        pa = PriorityArray()
        pa.write(8, 68.0)
        pa.write(8, 70.0)
        assert pa.active_value() == 70.0
