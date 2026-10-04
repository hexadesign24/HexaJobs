VERSION ?= v1.0.0
BINARY_NAME := hexajobs
DIST_DIR := dist
CMD_PATH := ./cmd/hexajobs
GOFLAGS_BUILD := -trimpath
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: help build-darwin-arm64 build-linux-amd64 build-windows-amd64 build-all clean test vet

help:
	@echo "HexaJobs CLI build targets:"
	@echo "  make build-windows-amd64  - Build Windows AMD64 (.exe + .sha256)"
	@echo "  make build-linux-amd64    - Build Linux AMD64"
	@echo "  make build-darwin-arm64   - Build macOS ARM64"
	@echo "  make build-all            - Build semua platform rilis"
	@echo "  make test                 - go test ./... -cover"
	@echo "  make vet                  - go vet ./..."
	@echo "  make clean                - Hapus direktori $(DIST_DIR)"
	@echo ""
	@echo "Variabel: VERSION=$(VERSION) BINARY_NAME=$(BINARY_NAME) DIST_DIR=$(DIST_DIR)"

.PHONY: build-darwin-arm64
build-darwin-arm64:
	@mkdir -p $(DIST_DIR)
	@echo "==> Building macOS ARM64 binary ($(VERSION))..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	@echo "==> Generating SHA-256 checksum..."
	@cd $(DIST_DIR) && sha256sum $(BINARY_NAME)-darwin-arm64 > $(BINARY_NAME)-darwin-arm64.sha256
	@echo "==> Selesai: $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 dan $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64.sha256"

.PHONY: build-linux-amd64
build-linux-amd64:
	@mkdir -p $(DIST_DIR)
	@echo "==> Building Linux AMD64 binary ($(VERSION))..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	@echo "==> Generating SHA-256 checksum..."
	@cd $(DIST_DIR) && sha256sum $(BINARY_NAME)-linux-amd64 > $(BINARY_NAME)-linux-amd64.sha256
	@echo "==> Selesai: $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 dan $(DIST_DIR)/$(BINARY_NAME)-linux-amd64.sha256"

.PHONY: build-windows-amd64
build-windows-amd64:
	@mkdir -p $(DIST_DIR)
	@echo "==> Building Windows AMD64 binary ($(VERSION))..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(DIST_DIR)/$(BINARY_NAME).exe ./cmd/hexajobs
	@echo "==> Generating SHA-256 checksum..."
	@cd $(DIST_DIR) && sha256sum $(BINARY_NAME).exe > $(BINARY_NAME).exe.sha256
	@echo "==> Selesai: $(DIST_DIR)/$(BINARY_NAME).exe dan $(DIST_DIR)/$(BINARY_NAME).exe.sha256"

build-all: build-darwin-arm64 build-linux-amd64 build-windows-amd64
	@echo "==> Semua biner rilis selesai di $(DIST_DIR)/"

clean:
	@rm -rf $(DIST_DIR)
	@echo "==> Direktori $(DIST_DIR) dibersihkan."

test:
	go test ./... -cover

vet:
	go vet ./...
