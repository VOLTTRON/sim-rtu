FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /sim-rtu ./cmd/sim-rtu

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /sim-rtu /usr/local/bin/sim-rtu
COPY configs/ configs/
EXPOSE 47808/udp 8080
ENTRYPOINT ["sim-rtu"]
CMD ["--config", "configs/default.yml"]
