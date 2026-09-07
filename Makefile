BINARY=terrat
VERSION?=0.1.5
PREFIX?=$(HOME)/go/bin
DESKTOP_DIR=$(HOME)/.local/share/applications

.PHONY: all build run clean install release site-build site-dev

all: build

build:
	GOTOOLCHAIN=local go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BINARY) main.go

run: build
	./$(BINARY)

ICON_DIR=$(HOME)/.local/share/icons/hicolor/512x512/apps

install: build
	mkdir -p $(PREFIX)
	mkdir -p $(DESKTOP_DIR)
	mkdir -p $(ICON_DIR)
	install -m 755 $(BINARY) $(PREFIX)/$(BINARY)
	cp icon.png $(PREFIX)/icon.png
	cp icon.png $(ICON_DIR)/terraterminal.png
	cp icon.png $(ICON_DIR)/terrat.png
	cp terraterminal.desktop $(DESKTOP_DIR)/terraterminal.desktop
	cp terraterminal.desktop $(DESKTOP_DIR)/terrat.desktop
	sed -i 's|Exec=.*|Exec=$(PREFIX)/$(BINARY)|g' $(DESKTOP_DIR)/terraterminal.desktop
	sed -i 's|Icon=.*|Icon=$(ICON_DIR)/terraterminal.png|g' $(DESKTOP_DIR)/terraterminal.desktop
	sed -i 's|Exec=.*|Exec=$(PREFIX)/$(BINARY)|g' $(DESKTOP_DIR)/terrat.desktop
	sed -i 's|Icon=.*|Icon=$(ICON_DIR)/terrat.png|g' $(DESKTOP_DIR)/terrat.desktop
	@command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database $(DESKTOP_DIR) || true
	@command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -f -t $(HOME)/.local/share/icons/hicolor 2>/dev/null || true
	@echo "TerraTerminal installed to $(PREFIX)/$(BINARY) with circular icon!"

clean:
	rm -f $(BINARY)
	rm -rf dist release

release:
	@mkdir -p dist
	@echo "Building release binaries..."
	GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o dist/$(BINARY)-linux-amd64 main.go
	GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o dist/$(BINARY)-linux-arm64 main.go
	@echo "Packaging release tarballs..."
	@cd dist && tar -czf $(BINARY)-v$(VERSION)-linux-amd64.tar.gz $(BINARY)-linux-amd64
	@cd dist && tar -czf $(BINARY)-v$(VERSION)-linux-arm64.tar.gz $(BINARY)-linux-arm64
	@cd dist && sha256sum *.tar.gz > checksums.txt
	@echo "Release assets ready in dist/:"
	@ls -lh dist/

site-build:
	cd site && npm run build

site-dev:
	cd site && npm run dev
