.PHONY: help build install test vet fmt clean

# Default target
help:
	@echo "easyrice - Go CLI dotfile manager"
	@echo ""
	@echo "Available targets:"
	@echo "  make build    - Compile binary to ./easyrice"
	@echo "  make install  - Install binary to \$$(GOPATH)/bin/easyrice and create rice symlink"
	@echo "  make test     - Run tests with race detector"
	@echo "  make vet      - Run go vet"
	@echo "  make fmt      - Format code with gofmt"
	@echo "  make clean    - Remove local ./easyrice binary"
	@echo "  make help     - Show this message"

# GOPATH configuration with fallback
GOPATH   ?= $(shell go env GOPATH)
GOBIN    := $(GOPATH)/bin
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.Version=$(VERSION)

# Build local binary
build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o easyrice ./cli

# Install binary to GOPATH/bin and create symlink
install: build
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(GOBIN)/easyrice ./cli
	ln -sf $(GOBIN)/easyrice $(GOBIN)/rice

# Run tests with race detector
test:
	go test -race -count=1 ./...

# Run go vet
vet:
	go vet ./...

# Format code
fmt:
	gofmt -w .

# Clean local binary
clean:
	rm -f easyrice
