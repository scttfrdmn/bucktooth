.PHONY: build run test clean install deps fmt lint help

# Build variables
BINARY_NAME=bucktooth
BUILD_DIR=bin
MAIN_PATH=./cmd/bucktooth
GO=go

# Default target
.DEFAULT_GOAL := help

## help: Display this help message
help:
	@echo "BuckTooth - AI Assistant Gateway"
	@echo ""
	@echo "Available targets:"
	@grep -E '^## [a-zA-Z_-]+:' $(MAKEFILE_LIST) | sed 's/^## /  /' | column -t -s ':'

## build: Build the gateway binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

## run: Run the gateway
run:
	$(GO) run $(MAIN_PATH)

## run-debug: Run with debug logging
run-debug:
	$(GO) run $(MAIN_PATH) start --log-level debug

## test: Run all tests
test:
	$(GO) test -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GO) test -cover -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-race: Run tests with race detector
test-race:
	$(GO) test -race ./...

## bench: Run benchmarks
bench:
	$(GO) test -bench=. -benchmem -benchtime=5s ./bench/...

## deps: Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

## fmt: Format code
fmt:
	$(GO) fmt ./...

## lint: Run linters
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

## vet: Run go vet
vet:
	$(GO) vet ./...

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	$(GO) clean

## install: Install the binary
install:
	$(GO) install $(MAIN_PATH)

## dev: Run in development mode with auto-reload (requires air)
dev:
	@which air > /dev/null || (echo "air not installed. Run: go install github.com/cosmtrek/air@latest" && exit 1)
	air

## docker-build: Build Docker image
docker-build:
	docker build -t $(BINARY_NAME):latest -f Dockerfile .

## docker-run: Run Docker container
docker-run:
	docker run -p 8080:8080 -p 18789:18789 --env-file .env $(BINARY_NAME):latest

## docker-compose-up: Start all services with docker-compose
docker-compose-up:
	docker-compose up -d

## docker-compose-down: Stop all services and remove volumes
docker-compose-down:
	docker-compose down -v

## test-harness: Spin up harness container, run integration tests, tear down
test-harness:
	docker-compose -f docker-compose.harness.yml up -d --wait
	GATEWAY_URL=http://localhost:8080 go test -v -tags=integration ./harness/...
	docker-compose -f docker-compose.harness.yml down

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"

## ci: Run checks suitable for CI (no integration tests, no lint requirement)
ci: fmt vet test
	@echo "CI checks passed!"

## release-tag: Create and push a version tag (usage: make release-tag VERSION=0.13.0)
release-tag:
	@test -n "$(VERSION)" || (echo "VERSION is required: make release-tag VERSION=0.13.0" && exit 1)
	git tag v$(VERSION)
	git push origin v$(VERSION)
