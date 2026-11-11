.PHONY: build build-web build-server dev-web dev-server clean

WEB_DIR := web
SERVER_DIR := server
EMBED_DIR := $(SERVER_DIR)/internal/web/dist

# Full production build: static frontend embedded into a single Go binary.
build: build-web build-server

build-web:
	cd $(WEB_DIR) && pnpm install --frozen-lockfile && pnpm build
	rm -rf $(EMBED_DIR)
	mkdir -p $(EMBED_DIR)
	cp -r $(WEB_DIR)/build/. $(EMBED_DIR)/

build-server:
	cd $(SERVER_DIR) && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../easyplan ./cmd/easyplan

# Frontend dev server on :5173 (proxies /api to the backend on :9092).
dev-web:
	cd $(WEB_DIR) && pnpm dev

# Backend with a local data dir.
dev-server:
	cd $(SERVER_DIR) && EASYPLAN_DATA_DIR=../.dev-data go run ./cmd/easyplan

clean:
	rm -rf $(WEB_DIR)/build $(WEB_DIR)/.svelte-kit easyplan .dev-data
