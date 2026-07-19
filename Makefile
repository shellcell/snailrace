.PHONY: build test clean

GO := env -u GOROOT go
CGO_ENABLED ?= $(if $(filter Darwin,$(shell uname -s)),1,0)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || printf dev)
LDFLAGS := -s -w -X github.com/shellcell/snailrace/internal/app.Version=$(VERSION)

build:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o snailrace ./cmd/snailrace

test:
	$(GO) test ./...

clean:
	rm -f snailrace
