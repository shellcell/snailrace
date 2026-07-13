.PHONY: build test clean

GO := env -u GOROOT go
CGO_ENABLED ?= $(if $(filter Darwin,$(shell uname -s)),1,0)

build:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -trimpath -ldflags="-s -w" -o snailrace ./cmd/snailrace

test:
	$(GO) test ./...

clean:
	rm -f snailrace
