PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
DESTDIR ?=
GO ?= go
BIN := bin/prettyx
GOFILES := $(shell git ls-files '*.go')

.PHONY: all build test fuzz bench install clean

all: $(BIN)

build: $(BIN)

$(BIN): $(GOFILES) go.mod go.sum
	@mkdir -p $(dir $@)
	$(GO) build -trimpath -ldflags '-s -w' -o $@ ./cmd/prettyx

install: $(BIN)
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 755 $(BIN) "$(DESTDIR)$(BINDIR)/prettyx"

test:
	go test -count=1 -race ./...

fuzz:
	@for pkg in $$(go list ./...); do \
		fuzzes=$$(go test -list Fuzz $$pkg 2>/dev/null | grep '^Fuzz' || true); \
		for fuzz in $$fuzzes; do \
			go test -run=^$$ -fuzz=^$$fuzz$$ -fuzztime=10s $$pkg || exit 1; \
		done; \
	done

bench:
	go test -run=^$$ -bench=. -benchmem ./...

clean:
	rm -f $(BIN)
	@rmdir bin 2>/dev/null || true
