VERSION ?= v1.1
DEBVER = $(shell echo $(VERSION) | sed 's/^v//')
MODULE := hexajobs.dev/hexajobs-cli
LDFLAGS := -s -w -X $(MODULE)/internal/ui/views.AppVersion=$(VERSION)
BIN_DIR := bin
DEB_DIR := dist/deb/hexajobs-cli_$(VERSION)_amd64

.PHONY: build build-linux-amd64 build-linux-arm64 checksum deb clean test vet

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/hexajobs ./cmd/hexajobs

build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/hexajobs-linux-amd64 ./cmd/hexajobs

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/hexajobs-linux-arm64 ./cmd/hexajobs

checksum: build-linux-amd64
	sha256sum $(BIN_DIR)/hexajobs-linux-amd64 > $(BIN_DIR)/hexajobs-linux-amd64.sha256
	@if [ -f $(BIN_DIR)/hexajobs-linux-arm64 ]; then sha256sum $(BIN_DIR)/hexajobs-linux-arm64 > $(BIN_DIR)/hexajobs-linux-arm64.sha256; fi

deb: build-linux-amd64
	mkdir -p $(DEB_DIR)/DEBIAN $(DEB_DIR)/usr/bin $(DEB_DIR)/usr/share/doc/hexajobs-cli
	cp $(BIN_DIR)/hexajobs-linux-amd64 $(DEB_DIR)/usr/bin/hexajobs
	cp README.md $(DEB_DIR)/usr/share/doc/hexajobs-cli/README
	sed "s/^Version:.*/Version: $(DEBVER)/" packaging/deb/control > $(DEB_DIR)/DEBIAN/control
	dpkg-deb --build $(DEB_DIR) dist/hexajobs-cli_$(VERSION)_amd64.deb

test:
	go test ./... -count=1

vet:
	go vet ./...
