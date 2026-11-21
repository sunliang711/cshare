.PHONY: build run dev dev-web web web-deps clean test vet lint

APP_NAME := crossshare-server
BUILD_DIR := ./build
WEB_DIR := ./web
VERSION ?= dev

export PATH := $(PATH):/usr/local/go/bin:/opt/homebrew/bin
NPM ?= $(or $(shell PATH="$(PATH)" command -v npm),npm)

web-deps:
	@if [ ! -d "$(WEB_DIR)/node_modules" ]; then \
		cd "$(WEB_DIR)" && "$(NPM)" ci; \
	fi

web: web-deps
	cd "$(WEB_DIR)" && "$(NPM)" run build

build: web
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

run: build
	$(BUILD_DIR)/$(APP_NAME)

dev: web
	go run ./cmd/server

dev-web: web-deps
	cd "$(WEB_DIR)" && "$(NPM)" run dev

clean:
	rm -rf $(BUILD_DIR) $(WEB_DIR)/dist

test: web
	go test ./... -race -count=1

vet: web
	go vet ./...

lint: web
	golangci-lint run ./...
