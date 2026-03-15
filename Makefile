BINARY := hass
CMD := ./cmd/hass

# Detect architecture for macOS compatibility
ARCH := $(shell uname -m)
ifeq ($(ARCH),arm64)
	GOARCH := arm64
else
	GOARCH := amd64
endif

.PHONY: build install clean test

build:
	GOARCH=$(GOARCH) go build -o $(BINARY) $(CMD)

install:
	GOARCH=$(GOARCH) go install $(CMD)

clean:
	rm -f $(BINARY)

test:
	go test ./...

.DEFAULT_GOAL := build
