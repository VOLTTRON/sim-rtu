"""Shared test fixtures for sim-rtu tests."""

from __future__ import annotations

from pathlib import Path

import pytest

CONFIGS_DIR = Path(__file__).resolve().parent.parent / "configs"


@pytest.fixture
def schneider_csv() -> Path:
    """Path to the Schneider thermostat registry CSV."""
    return CONFIGS_DIR / "schneider.csv"


@pytest.fixture
def openstat_csv() -> Path:
    """Path to the OpenStat thermostat registry CSV."""
    return CONFIGS_DIR / "openstat.csv"


@pytest.fixture
def dent_csv() -> Path:
    """Path to the DENT power meter registry CSV."""
    return CONFIGS_DIR / "dent.csv"
