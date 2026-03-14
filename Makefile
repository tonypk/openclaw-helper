VERSION ?= 0.1.0
BINARY_NAME = och-helper
WINDOWS_BINARY = $(BINARY_NAME).exe

.PHONY: build build-windows build-dev test clean lint

## Build for current platform (development)
build-dev:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME) ./cmd/helper

## Cross-compile for Windows amd64
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(WINDOWS_BINARY) ./cmd/helper

## Run tests
test:
	go test -v -race ./...

## Run tests with coverage
test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(WINDOWS_BINARY) coverage.out

## Run go vet
lint:
	go vet ./...

## Run the helper in CLI check mode
check: build-dev
	./$(BINARY_NAME) --check

## Run the helper IPC server
run: build-dev
	./$(BINARY_NAME)
