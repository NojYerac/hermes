BINARY      = hermes-bridge
CMD         = ./cmd/hermes-bridge
BUILD_DIR   = build
LDFLAGS     = -ldflags="-s -w"

.PHONY: build run test lint clean

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD)

run:
	go run $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
