.PHONY: run build install-air dev clean

BINARY_NAME=bin/ncore-tmdb
CMD_PATH=./cmd/run
export PATH := $(HOME)/go/bin:$(PATH)

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY_NAME) $(CMD_PATH)

dev:
	air

docker:
	docker build -t ncore-tmdb .

clean:
	rm -rf bin/
	rm -rf tmp/
