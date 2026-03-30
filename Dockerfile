FROM python:3.11-slim AS builder

WORKDIR /app
COPY pyproject.toml .
COPY src/ src/
RUN pip install --no-cache-dir .

FROM python:3.11-slim

WORKDIR /app
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin/sim-rtu /usr/local/bin/sim-rtu
COPY configs/ configs/

EXPOSE 47808/udp
EXPOSE 8080

ENTRYPOINT ["sim-rtu"]
CMD ["--config", "configs/default.yml"]
