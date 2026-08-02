.PHONY: all build frontend webapp widget run dev docker clean install-air

BINARY_NAME := bin/ncore-tmdb
CMD_PATH    := ./cmd/run

# Prefer a known-good toolchain (avoids broken local Go installs).
export GOTOOLCHAIN := go1.25.5
export PATH := $(HOME)/.bun/bin:$(HOME)/go/bin:$(PATH)
export CGO_ENABLED := 0

# Default target: install frontends with bun, build them, embed into Go binary
all: build

build: frontend
	@mkdir -p bin
	go build -ldflags="-s -w" -o $(BINARY_NAME) $(CMD_PATH)
	@echo "Built $(BINARY_NAME)"

frontend: webapp widget

webapp:
	cd webapp && bun install --frozen-lockfile || bun install
	cd webapp && bun run build

widget:
	cd widget && bun install --frozen-lockfile || bun install
	cd widget && bun run build

run: build
	./$(BINARY_NAME)

dev: frontend
	air

docker:
	docker build -t ncore-tmdb .

clean:
	rm -rf bin/ tmp/ air/
	rm -rf webapp/node_modules widget/node_modules
	rm -rf webapp/dist widget/dist
	rm -rf internal/static/webapp/* internal/static/widget/*
	@printf '%s\n' '<!doctype html><html><body>Build with: make</body></html>' > internal/static/webapp/index.html
	@printf '%s\n' '<!-- build with: make -->' > internal/static/widget/snippet.html

install-air:
	go install github.com/air-verse/air@latest
