.PHONY: build run install clean

BINARY=cipher
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/cipher

install: build
	install -m 755 $(BUILD_DIR)/$(BINARY) /home/cipher/bin/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)
