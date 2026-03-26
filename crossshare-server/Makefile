.PHONY: build run dev clean test lint

APP_NAME := crossshare-server
BUILD_DIR := ./build
VERSION ?= dev

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

run: build
	$(BUILD_DIR)/$(APP_NAME)

dev:
	go run ./cmd/server

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...
