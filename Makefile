.PHONY: test vet build install tidy demo clean

PREFIX ?= $(shell go env GOPATH)/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -ldflags "$(LDFLAGS)" -o folio .

install:
	go install -ldflags "$(LDFLAGS)" .

tidy:
	go mod tidy

demo: build
	@tmpdir=$$(mktemp -d); \
	HOME=$$tmpdir ./folio ingest chat testdata/chat_whatsapp.txt; \
	HOME=$$tmpdir ./folio ingest chat testdata/chat_telegram.html; \
	HOME=$$tmpdir ./folio ingest letter testdata/letter.eml; \
	HOME=$$tmpdir ./folio stats; \
	HOME=$$tmpdir ./folio search boarding; \
	rm -rf $$tmpdir

clean:
	rm -f folio
