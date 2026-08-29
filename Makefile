.PHONY: test vet build install tidy demo clean release-snapshot

PREFIX ?= $(shell go env GOPATH)/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.5.0)
LDFLAGS := -X main.version=$(VERSION)

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -ldflags "$(LDFLAGS)" -o folio .
	chmod +x install.sh

install:
	go install -ldflags "$(LDFLAGS)" .
	chmod +x install.sh

tidy:
	go mod tidy

demo: build
	@tmpdir=$$(mktemp -d); \
	HOME=$$tmpdir ./folio init; \
	HOME=$$tmpdir ./folio ingest chat testdata/chat_whatsapp.txt; \
	HOME=$$tmpdir ./folio ingest chat testdata/chat_telegram.html; \
	HOME=$$tmpdir ./folio ingest letter testdata/letter.eml; \
	HOME=$$tmpdir ./folio stats; \
	HOME=$$tmpdir ./folio search boarding; \
	rm -rf $$tmpdir

release-snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -f folio
	rm -rf dist
