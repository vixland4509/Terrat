BINARY=terrat
PREFIX?=$(HOME)/go/bin
DESKTOP_DIR=$(HOME)/.local/share/applications

.PHONY: all build run clean install site-build site-dev

all: build

build:
	GOTOOLCHAIN=local go build -ldflags="-s -w" -o $(BINARY) main.go

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

site-build:
	cd site && npm run build

site-dev:
	cd site && npm run dev
