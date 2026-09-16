.PHONY: all build test test-v link clean help

BINARY := bin/cpa-quotas$(if $(filter Windows_NT,$(OS)),.exe,)

ifeq ($(OS),Windows_NT)
    CLEAN_BIN := if exist bin rmdir /s /q bin
else
    CLEAN_BIN := rm -rf bin
endif

all: test build

build:
	go build -o $(BINARY) ./cmd/cpa-quotas

test:
	go test ./...

test-v:
	go test -v ./...

link:
	herdr plugin link .

clean:
	go clean
	$(CLEAN_BIN)

help:
	@echo Available targets:
	@echo   build   - Build cpa-quotas binary
	@echo   test    - Run unit tests
	@echo   test-v  - Run unit tests with verbose output
	@echo   link    - Link plugin into Herdr
	@echo   clean   - Remove build artifacts
