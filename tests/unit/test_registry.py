"""Tests for the registry CSV parser."""

from __future__ import annotations

from pathlib import Path

from sim_rtu.points.registry import PointDefinition, parse_registry


class TestParseRegistry:
    """Tests for parse_registry across all three CSV formats."""

    def test_schneider_parses_all_points(self, schneider_csv: Path) -> None:
        points = parse_registry(schneider_csv)
        assert len(points) == 59

    def test_openstat_parses_all_points(self, openstat_csv: Path) -> None:
        points = parse_registry(openstat_csv)
        assert len(points) == 65

    def test_dent_parses_all_points(self, dent_csv: Path) -> None:
        points = parse_registry(dent_csv)
        assert len(points) == 39

    def test_schneider_first_point(self, schneider_csv: Path) -> None:
        points = parse_registry(schneider_csv)
        p = points[0]
        assert p.reference_name == "Effective Setpoint"
        assert p.volttron_name == "EffectiveZoneTemperatureSetPoint"
        assert p.units == "degreesFahrenheit"
        assert p.bacnet_object_type == "analogInput"
        assert p.property_name == "presentValue"
        assert p.writable is True
        assert p.index == 329
        assert p.write_priority == 16

    def test_dent_points_are_read_only(self, dent_csv: Path) -> None:
        points = parse_registry(dent_csv)
        for p in points:
            assert p.writable is False
            assert p.write_priority is None

    def test_openstat_active_column(self, openstat_csv: Path) -> None:
        points = parse_registry(openstat_csv)
        active_points = [p for p in points if p.active]
        inactive_points = [p for p in points if not p.active]
        assert len(active_points) > 0
        assert len(inactive_points) > 0

    def test_default_value_parsing(self, schneider_csv: Path) -> None:
        points = parse_registry(schneider_csv)
        by_name = {p.volttron_name: p for p in points}
        occ_heat = by_name["OccupiedHeatingSetPoint"]
        assert occ_heat.default_value == 72.0
        occ_cool = by_name["OccupiedCoolingSetPoint"]
        assert occ_cool.default_value == 75.0

    def test_object_key_tuple(self, schneider_csv: Path) -> None:
        points = parse_registry(schneider_csv)
        p = points[0]
        assert p.object_key == ("analogInput", 329)

    def test_point_definition_is_frozen(self, schneider_csv: Path) -> None:
        points = parse_registry(schneider_csv)
        import pytest

        with pytest.raises(AttributeError):
            points[0].volttron_name = "changed"  # type: ignore[misc]

    def test_dent_default_values_are_none(self, dent_csv: Path) -> None:
        points = parse_registry(dent_csv)
        for p in points:
            assert p.default_value is None

    def test_all_dent_points_active_by_default(self, dent_csv: Path) -> None:
        points = parse_registry(dent_csv)
        for p in points:
            assert p.active is True
