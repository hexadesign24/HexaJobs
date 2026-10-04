VERSION ?= v1.1.1
DEBVER = $(shell echo $(VERSION) | sed 's/^v//')
MODULE := hexajobs.dev/hexajobs-cli
LDFLAGS := -s -w -X $(MODULE)/internal/ui/views.AppVersion=$(VERSION)
BIN_DIR := bin
BINARY_NAME := hexajobs
DIST_DIR := dist
DEB_DIR := dist/deb/hexajobs-cli_$(VERSION)_amd64

.PHONY: build build-linux-amd64 build-linux-arm64 build-windows-amd64 build-darwin-arm64 build-all checksum checksum-windows deb clean test vet

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/hexajobs ./cmd/hexajobs

build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/hexajobs-linux-amd64 ./cmd/hexajobs

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/hexajobs-linux-arm64 ./cmd/hexajobs

build-windows-amd64:
	@mkdir -p $(DIST_DIR)
	@echo "==> Building Windows AMD64 binary ($(VERSION))..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME).exe ./cmd/hexajobs
	@echo "==> Generating SHA-256 checksum..."
	@cd $(DIST_DIR) && sha256sum $(BINARY_NAME).exe > $(BINARY_NAME).exe.sha256
	@echo "==> Selesai: $(DIST_DIR)/$(BINARY_NAME).exe dan $(DIST_DIR)/$(BINARY_NAME).exe.sha256"

build-darwin-arm64:
	@mkdir -p $(DIST_DIR)
	@echo "==> Building macOS ARM64 binary ($(VERSION))..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/hexajobs
	@echo "==> Generating SHA-256 checksum..."
	@cd $(DIST_DIR) && sha256sum $(BINARY_NAME)-darwin-arm64 > $(BINARY_NAME)-darwin-arm64.sha256
	@echo "==> Selesai: $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 dan $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64.sha256"

build-all: build-linux-amd64 build-windows-amd64 build-darwin-arm64
	@echo "==> Semua biner rilis selesai."

checksum: build-linux-amd64
	sha256sum $(BIN_DIR)/hexajobs-linux-amd64 > $(BIN_DIR)/hexajobs-linux-amd64.sha256
	@if [ -f $(BIN_DIR)/hexajobs-linux-arm64 ]; then sha256sum $(BIN_DIR)/hexajobs-linux-arm64 > $(BIN_DIR)/hexajobs-linux-arm64.sha256; fi

checksum-windows: build-windows-amd64

deb: build-linux-amd64
	mkdir -p $(DEB_DIR)/DEBIAN $(DEB_DIR)/usr/bin $(DEB_DIR)/usr/share/doc/hexajobs-cli
	cp $(BIN_DIR)/hexajobs-linux-amd64 $(DEB_DIR)/usr/bin/hexajobs
	cp README.md $(DEB_DIR)/usr/share/doc/hexajobs-cli/README
	sed "s/^Version:.*/Version: $(DEBVER)/" packaging/deb/control > $(DEB_DIR)/DEBIAN/control
	dpkg-deb --build $(DEB_DIR) dist/hexajobs-cli_$(VERSION)_amd64.deb

clean:
	rm -rf $(DIST_DIR)
	@echo "==> Direktori $(DIST_DIR) dibersihkan."

test:
	go test ./... -count=1

vet:
	go vet ./...
