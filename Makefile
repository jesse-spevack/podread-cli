BINARY   := podread
MODULE   := github.com/jspevack/podread-cli
DIST     := dist

VERSION  ?= dev
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE     := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS  := -s -w \
	-X '$(MODULE)/internal/api.Version=$(VERSION)' \
	-X '$(MODULE)/internal/api.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/api.Date=$(DATE)'

PLATFORMS := darwin/arm64 darwin/amd64 linux/arm64 linux/amd64

.PHONY: build build-all clean

## build: compile for the current platform
build:
	@mkdir -p $(DIST)
	go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY) .

## build-all: cross-compile for all supported platforms
build-all:
	@mkdir -p $(DIST)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output=$(DIST)/$(BINARY)-$${os}-$${arch}; \
		echo "Building $$output ..."; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$output . || exit 1; \
	done

## clean: remove build artifacts
clean:
	rm -rf $(DIST)
