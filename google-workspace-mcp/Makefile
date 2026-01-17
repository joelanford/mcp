BINARY_NAME := google-workspace-mcp
BIN_DIR := bin

.PHONY: build clean

install:
	go install .

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) .

clean:
	rm -rf $(BIN_DIR)
