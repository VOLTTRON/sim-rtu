"""Tests for the PointStore."""

from __future__ import annotations

import pytest

from sim_rtu.points.priority_array import PriorityArray
from sim_rtu.points.registry import PointDefinition
from sim_rtu.points.store import PointStore


def _make_writable_point(
    name: str = "TestPoint",
    index: int = 1,
    default: float | None = 72.0,
    priority: int = 16,
) -> PointDefinition:
    return PointDefinition(
        reference_name=f"Ref {name}",
        volttron_name=name,
        units="degreesFahrenheit",
        bacnet_object_type="analogValue",
        property_name="presentValue",
        writable=True,
        index=index,
        write_priority=priority,
        default_value=default,
        notes="",
    )


def _make_readonly_point(
    name: str = "ReadOnly",
    index: int = 100,
    default: float | None = 5.0,
) -> PointDefinition:
    return PointDefinition(
        reference_name=f"Ref {name}",
        volttron_name=name,
        units="kilowatts",
        bacnet_object_type="analogInput",
        property_name="presentValue",
        writable=False,
        index=index,
        write_priority=None,
        default_value=default,
        notes="",
    )


class TestPointStore:
    @pytest.mark.asyncio
    async def test_read_writable_point_returns_default(self) -> None:
        store = PointStore.from_definitions([_make_writable_point()])
        val = await store.read_by_name("TestPoint")
        assert val == 72.0

    @pytest.mark.asyncio
    async def test_read_readonly_point_returns_default(self) -> None:
        store = PointStore.from_definitions([_make_readonly_point()])
        val = await store.read_by_name("ReadOnly")
        assert val == 5.0

    @pytest.mark.asyncio
    async def test_write_and_read_by_name(self) -> None:
        store = PointStore.from_definitions([_make_writable_point()])
        await store.write_by_name("TestPoint", 68.0, priority=8)
        val = await store.read_by_name("TestPoint")
        assert val == 68.0

    @pytest.mark.asyncio
    async def test_write_raises_on_readonly(self) -> None:
        store = PointStore.from_definitions([_make_readonly_point()])
        with pytest.raises(TypeError):
            await store.write_by_name("ReadOnly", 10.0)

    @pytest.mark.asyncio
    async def test_read_by_object_key(self) -> None:
        store = PointStore.from_definitions([_make_writable_point(index=42)])
        val = await store.read_by_object_key("analogValue", 42)
        assert val == 72.0

    @pytest.mark.asyncio
    async def test_write_by_object_key(self) -> None:
        store = PointStore.from_definitions([_make_writable_point(index=42)])
        await store.write_by_object_key("analogValue", 42, 65.0, priority=8)
        val = await store.read_by_object_key("analogValue", 42)
        assert val == 65.0

    @pytest.mark.asyncio
    async def test_set_internal_updates_value(self) -> None:
        store = PointStore.from_definitions([_make_readonly_point()])
        await store.set_internal("ReadOnly", 99.0)
        val = await store.read_by_name("ReadOnly")
        assert val == 99.0

    @pytest.mark.asyncio
    async def test_read_all_returns_all_points(self) -> None:
        defs = [
            _make_writable_point("A", index=1),
            _make_readonly_point("B", index=2),
        ]
        store = PointStore.from_definitions(defs)
        snapshot = await store.read_all()
        assert "A" in snapshot
        assert "B" in snapshot

    @pytest.mark.asyncio
    async def test_point_names(self) -> None:
        defs = [
            _make_writable_point("Alpha", index=1),
            _make_writable_point("Beta", index=2),
        ]
        store = PointStore.from_definitions(defs)
        assert set(store.point_names) == {"Alpha", "Beta"}

    @pytest.mark.asyncio
    async def test_read_unknown_name_raises(self) -> None:
        store = PointStore.from_definitions([])
        with pytest.raises(KeyError):
            await store.read_by_name("NoSuchPoint")
