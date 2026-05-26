WEB_DIR := apps/web
BACKEND_DIR := apps/backend
WEB_DIST := $(WEB_DIR)/dist
BACKEND_STATIC_DIST := $(BACKEND_DIR)/servers/echo/staticweb/dist

.PHONY: web-install web-build web-copy-assets single-binary docker-release

web-install:
	cd $(WEB_DIR) && npm install

web-build:
	cd $(WEB_DIR) && npm run build

web-copy-assets: web-build
	mkdir -p $(BACKEND_STATIC_DIST)
	find $(BACKEND_STATIC_DIST) -mindepth 1 -maxdepth 1 ! -name placeholder.txt -exec rm -rf {} +
	cp -R $(WEB_DIST)/. $(BACKEND_STATIC_DIST)/

single-binary: web-copy-assets
	cd $(BACKEND_DIR) && GO111MODULE=on CGO_ENABLED=1 go build -o datalk-api ./cmd/api

docker-release:
	docker build -t datalk:latest .
