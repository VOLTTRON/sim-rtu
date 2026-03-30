"""Entry point for the sim-rtu simulator.

Usage:
    sim-rtu --config configs/default.yml
    python -m sim_rtu --config configs/default.yml
"""

from __future__ import annotations

import argparse
import sys


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(
        prog="sim-rtu",
        description="Simulated RTU devices for VOLTTRON platform driver testing",
    )
    parser.add_argument(
        "--config",
        type=str,
        default="configs/default.yml",
        help="Path to the YAML configuration file (default: configs/default.yml)",
    )
    parser.add_argument(
        "--log-level",
        type=str,
        default="INFO",
        choices=["DEBUG", "INFO", "WARNING", "ERROR"],
        help="Logging level (default: INFO)",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> None:
    """Main entry point for the simulator."""
    raise NotImplementedError(
        "TODO: load config, initialize devices, start simulation loop"
    )


if __name__ == "__main__":
    main()
