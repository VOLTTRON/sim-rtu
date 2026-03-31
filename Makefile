BINARY := sim-rtu
MODULE := github.com/VOLTTRON/sim-rtu

.PHONY: build run test lint clean

build:
	go build -o bin/$(BINARY) ./cmd/sim-rtu

run: build
	./bin/$(BINARY) --config configs/default.yml

test:
	go test ./... -race -cover -coverprofile=coverage.out

coverage: test
	go tool cover -func=coverage.out

clean:
	rm -rf bin/ coverage.out
