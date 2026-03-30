"""FastAPI application for simulator inspection and control.

Provides REST endpoints for:
- Reading/writing point values
- Querying simulation state
- Adjusting simulation parameters at runtime
"""

from __future__ import annotations

from fastapi import FastAPI

from sim_rtu.api.routes import create_router


def create_app() -> FastAPI:
    """Create and configure the FastAPI application.

    Returns:
        Configured FastAPI application instance.
    """
    app = FastAPI(
        title="sim-rtu",
        description="Simulated RTU device API",
        version="0.1.0",
    )
    app.include_router(create_router())
    return app


async def start_api(host: str = "0.0.0.0", port: int = 8080) -> None:
    """Start the API server.

    Args:
        host: Bind address.
        port: Listen port.
    """
    raise NotImplementedError("TODO: implement uvicorn startup")
