VERSION ?= 0.1.0
BINARY_NAME = och-helper
WINDOWS_BINARY = $(BINARY_NAME).exe
TARGET_TRIPLE := $(shell rustc -vV 2>/dev/null | grep host | cut -d' ' -f2)
TELEGRAM_BOT_TOKEN ?=
TELEGRAM_CHAT_ID ?=
LDFLAGS = -X main.version=$(VERSION)
ifneq ($(TELEGRAM_BOT_TOKEN),)
LDFLAGS += -X github.com/tonypk/openclaw-helper/internal/report.telegramBotToken=$(TELEGRAM_BOT_TOKEN)
endif
ifneq ($(TELEGRAM_CHAT_ID),)
LDFLAGS += -X github.com/tonypk/openclaw-helper/internal/report.telegramChatID=$(TELEGRAM_CHAT_ID)
endif

.PHONY: build build-windows build-dev build-sidecar test clean lint check run dev frontend-dev frontend-build

## Build for current platform (development)
build-dev:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/helper

## Build Go helper as Tauri sidecar for current platform
build-sidecar:
	go build -ldflags "$(LDFLAGS)" -o frontend/src-tauri/binaries/$(BINARY_NAME)-$(TARGET_TRIPLE) ./cmd/helper

## Cross-compile for Windows amd64
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o frontend/src-tauri/binaries/$(BINARY_NAME)-x86_64-pc-windows-msvc.exe ./cmd/helper

## Run Go tests
test:
	go test -v ./...

## Run Go tests with race detector
test-race:
	go test -v -race ./...

## Run tests with coverage
test-cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(WINDOWS_BINARY) coverage.out
	rm -f frontend/src-tauri/binaries/$(BINARY_NAME)-*

## Run go vet
lint:
	go vet ./...

## Run the helper in CLI check mode
check: build-dev
	./$(BINARY_NAME) --check

## Run the helper IPC server
run: build-dev
	./$(BINARY_NAME)

## Install frontend dependencies
frontend-install:
	cd frontend && npm install

## Build frontend only
frontend-build:
	cd frontend && npm run build

## Run frontend dev server only (for UI work without Tauri)
frontend-dev:
	cd frontend && npm run dev

## Full development: build sidecar + run tauri dev
dev: build-sidecar
	cd frontend && npx @tauri-apps/cli@2 dev

## Build complete Tauri application
build: build-sidecar frontend-build
	cd frontend && npx @tauri-apps/cli@2 build

## Run all checks before commit
ci: lint test frontend-build
	@echo "All checks passed!"
