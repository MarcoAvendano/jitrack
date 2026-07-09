BINARY := sr-cli
INSTALL_DIR := /opt/homebrew/bin

.PHONY: build test install clean

build:
	go build -o $(BINARY) .

test:
	go test ./...

install: build
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

clean:
	rm -f $(BINARY)
