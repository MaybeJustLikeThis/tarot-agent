.PHONY: build test run clean lint

BINARY_NAME := tarot-agent
BUILD_DIR := ./bin

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/tarot-agent

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html
