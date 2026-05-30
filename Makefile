ifeq ($(OS),Windows_NT)
    # Detect whether the shell is cmd.exe or sh/bash by checking how it expands %OS%
    ifeq ($(shell echo %OS%),Windows_NT)
        BINARY_NAME=skillzeug.exe
        INSTALL_DIR=$(USERPROFILE)\.local\bin
        MKDIR=if not exist "$(INSTALL_DIR)" mkdir "$(INSTALL_DIR)"
        CP=copy /Y $(BINARY_NAME) "$(INSTALL_DIR)\$(BINARY_NAME)"
        RM=if exist $(BINARY_NAME) del /Q $(BINARY_NAME)
        RUN_CMD=.\$(BINARY_NAME)
    else
        BINARY_NAME=skillzeug.exe
        INSTALL_DIR=$(HOME)/.local/bin
        MKDIR=mkdir -p "$(INSTALL_DIR)"
        CP=cp $(BINARY_NAME) "$(INSTALL_DIR)/$(BINARY_NAME)"
        RM=rm -f $(BINARY_NAME)
        RUN_CMD=./$(BINARY_NAME)
    endif
else
    BINARY_NAME=skillzeug
    INSTALL_DIR=$(HOME)/bin
    MKDIR=mkdir -p "$(INSTALL_DIR)"
    CP=cp $(BINARY_NAME) "$(INSTALL_DIR)/$(BINARY_NAME)"
    RM=rm -f $(BINARY_NAME)
    RUN_CMD=./$(BINARY_NAME)
endif

.PHONY: all build clean run test help install

all: build

build:
	go build -o $(BINARY_NAME) main.go

install: build
	$(MKDIR)
	$(CP)
	@echo "Installed $(BINARY_NAME) to $(INSTALL_DIR)"

run: build
	$(RUN_CMD)

test:
	go test ./...

clean:
	go clean
	$(RM)

help:
	@echo "Usage:"
	@echo "  make build   - Build the binary"
	@echo "  make install - Build and install the binary"
	@echo "  make run     - Build and run the binary"
	@echo "  make test    - Run tests"
	@echo "  make clean   - Remove binary and clean build cache"
