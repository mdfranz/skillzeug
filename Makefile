BINARY_NAME=skillzeug
INSTALL_DIR=$(HOME)/bin

.PHONY: all build clean run test help install

all: build

build:
	go build -o $(BINARY_NAME) main.go

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(INSTALL_DIR)"

run: build
	./$(BINARY_NAME)

test:
	go test ./...

clean:
	go clean
	rm -f $(BINARY_NAME)

help:
	@echo "Usage:"
	@echo "  make build   - Build the binary"
	@echo "  make install - Build and install the binary to ~/bin"
	@echo "  make run     - Build and run the binary"
	@echo "  make test    - Run tests"
	@echo "  make clean   - Remove binary and clean build cache"
