BINARY=rcloneb
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"
DIST_DIR=dist

.PHONY: build clean install uninstall run tidy all release

build:
	go build $(LDFLAGS) -o $(BINARY) .

clean:
	rm -f $(BINARY)
	rm -rf $(DIST_DIR)

install: build
	cp $(BINARY) /usr/local/bin/

uninstall:
	rm -f /usr/local/bin/$(BINARY)

run: build
	./$(BINARY)

tidy:
	go mod tidy

# Cross-compilation targets
all: build-darwin build-linux build-windows

build-darwin: build-darwin-amd64 build-darwin-arm64

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-darwin-amd64 .

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-darwin-arm64 .

build-linux: build-linux-amd64 build-linux-arm64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-linux-amd64 .

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-linux-arm64 .

build-windows: build-windows-amd64

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-windows-amd64.exe .

release: clean all
	@echo "Built binaries in $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/
