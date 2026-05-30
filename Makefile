# Lintasan Go — build orchestration
#
# The dashboard UI (SvelteKit) is compiled to a static SPA and embedded into the
# Go binary via go:embed, so `lintasan start` serves the full app from a single
# executable with no separate Node process.
#
# Common targets:
#   make build        → build frontend + embed + compile single binary (./lintasan)
#   make frontend     → build only the SvelteKit static output into internal/web/dist
#   make backend      → compile the Go binary (assumes dist/ already built)
#   make run          → build then start the server
#   make test         → run the Go test suite (excludes the experimental provider pkg)
#   make clean        → remove build artifacts
#   make release      → cross-compile linux/amd64 + darwin/arm64 binaries into dist-bin/

BINARY      := lintasan
PKG         := ./cmd/lintasan
DIST        := internal/web/dist
FRONTEND    := frontend
VERSION     := $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X github.com/sanhaji182/lintasan-go/internal/version.Version=$(VERSION)

.PHONY: build frontend backend run test clean release deps

## Full build: frontend → embed → single binary
build: frontend backend
	@echo "✓ Built ./$(BINARY) ($(VERSION)) with embedded dashboard"

## Compile the SvelteKit dashboard into the embedded dist directory
frontend:
	@echo "→ Building frontend (SvelteKit static SPA)…"
	cd $(FRONTEND) && npm install && npm run build
	@echo "→ Syncing build output into $(DIST)…"
	rm -rf $(DIST)
	mkdir -p $(DIST)
	cp -r $(FRONTEND)/build/* $(DIST)/
	@# keep the placeholder so the dir is never empty
	touch $(DIST)/.gitkeep

## Compile the Go binary (CGO required for go-sqlite3)
backend:
	@echo "→ Compiling Go binary…"
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o $(BINARY) $(PKG)

## Build and start
run: build
	./$(BINARY) start

## Run tests (skip the untracked experimental provider package)
test:
	go test $$(go list ./... | grep -v '/internal/provider')

## Remove build artifacts (keeps the .gitkeep placeholder)
clean:
	rm -f $(BINARY)
	rm -rf $(FRONTEND)/build $(FRONTEND)/.svelte-kit
	find $(DIST) -mindepth 1 ! -name '.gitkeep' -delete
	rm -rf dist-bin

## Cross-compile release binaries (frontend embedded). CGO needs a cross toolchain
## for non-native targets; the native target always works.
release: frontend
	@echo "→ Building release binaries into dist-bin/…"
	mkdir -p dist-bin
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o dist-bin/$(BINARY)-linux-amd64 $(PKG)
	@echo "✓ dist-bin/$(BINARY)-linux-amd64"

## Install frontend deps only
deps:
	cd $(FRONTEND) && npm install
