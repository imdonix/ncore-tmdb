.PHONY: run build install-air dev clean

BINARY_NAME=bin/media-manager
CMD_PATH=./cmd/run
export PATH := $(HOME)/go/bin:$(PATH)

run:
	go run $(CMD_PATH)

build:
	mkdir -p bin
	go build -o $(BINARY_NAME) $(CMD_PATH)

dev:
	air

clean:
	rm -rf bin/
	rm -rf tmp/
