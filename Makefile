# Works with GNU make on Linux/macOS and via Git Bash on Windows.
BINARY  := tradermade
MODULE  := github.com/tradermade/tradermade-cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.1.0-dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%d)
LDFLAGS := -X $(MODULE)/cmd.Version=$(VERSION) -X $(MODULE)/cmd.Commit=$(COMMIT) -X $(MODULE)/cmd.Date=$(DATE)

ifeq ($(OS),Windows_NT)
	EXE := .exe
endif

.PHONY: build test vet fmt check install clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY)$(EXE) .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

# check = everything CI runs; keep them identical.
check: vet test
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then \
		echo "gofmt needed on:"; echo "$$unformatted"; exit 1; fi

install:
	go build -ldflags "$(LDFLAGS)" -o "$(shell go env GOPATH)/bin/$(BINARY)$(EXE)" .

clean:
	rm -f $(BINARY) $(BINARY).exe
