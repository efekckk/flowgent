# Flowgent Makefile — the canonical commands for working on this repo.
#
# First time:
#   make env            # writes .env with a generated FLOWGENT_CRED_KEY
#   make demo           # docker compose up — full stack at http://localhost:8080
#
# Day-to-day:
#   make dev            # postgres in docker, server hot-reloads on host
#   make test           # backend + web suites
#   make build          # production-grade binary + embedded SPA
#   make clean          # drop generated artifacts

SHELL          := /bin/bash
GO             ?= go
NPM            ?= npm
WEB_DIR        := web
DIST_DIR       := internal/webfs/dist
BIN            := flowgent

DATABASE_URL_DEV ?= postgres://flowgent:flowgent@localhost:5432/flowgent?sslmode=disable

.PHONY: help env demo demo-down dev web-build go-build build test web-test go-test \
        compose-config docker-build fmt vet clean

help:
	@echo "Flowgent — common targets"
	@echo "  make env          generate .env from .env.example with a fresh credential key"
	@echo "  make demo         docker compose up (postgres + api), http://localhost:8080"
	@echo "  make demo-down    docker compose down (keeps volumes)"
	@echo "  make dev          postgres in docker, go run on host"
	@echo "  make build        web build + go build → ./$(BIN)"
	@echo "  make test         go test ./... -race && npm --prefix $(WEB_DIR) test"
	@echo "  make clean        drop binary + web dist + embed dist"

env: .env

.env:
	@if [ ! -f .env.example ]; then \
		echo "error: .env.example missing"; exit 1; \
	fi
	@cp .env.example .env
	@KEY="$$(openssl rand -base64 32)"; \
	sed -i.bak "s|^FLOWGENT_CRED_KEY=.*|FLOWGENT_CRED_KEY=$$KEY|" .env && rm -f .env.bak
	@echo ".env written with a fresh FLOWGENT_CRED_KEY. Edit it to add provider keys."

demo: env
	docker compose up --build

demo-down:
	docker compose down

dev: env
	docker compose up -d postgres
	@echo "postgres is up on :5432"
	@set -a; source .env; set +a; \
	DATABASE_URL="$(DATABASE_URL_DEV)" $(GO) run ./cmd/flowgent

web-build:
	cd $(WEB_DIR) && $(NPM) install --no-audit --no-fund
	cd $(WEB_DIR) && $(NPM) run build
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	cp -r $(WEB_DIR)/dist/. $(DIST_DIR)/

go-build: web-build
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags='-s -w' -o $(BIN) ./cmd/flowgent

build: go-build

test: go-test web-test

go-test:
	$(GO) test ./... -race

web-test:
	cd $(WEB_DIR) && $(NPM) install --no-audit --no-fund
	cd $(WEB_DIR) && $(NPM) test

compose-config:
	docker compose config

docker-build:
	docker build -t flowgent .

fmt:
	$(GO) fmt ./...
	cd $(WEB_DIR) && $(NPM) run lint --if-present

vet:
	$(GO) vet ./...

clean:
	rm -f $(BIN)
	rm -rf $(WEB_DIR)/dist
	rm -rf $(WEB_DIR)/node_modules
	# Keep the placeholder index.html the webfs package needs.
	@if [ -d $(DIST_DIR) ]; then \
		find $(DIST_DIR) -mindepth 1 ! -name 'index.html' -exec rm -rf {} +; \
	fi
