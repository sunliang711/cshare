BINARY := share
BUILD_DIR := ./build

.PHONY: build install clean

build:
	go build -o $(BUILD_DIR)/$(BINARY) .

install:
	go install .

clean:
	rm -f $(BINARY)
