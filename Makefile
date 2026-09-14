.PHONY: all build test bench clean

BINARY_NAME=okf

all: build

build:
	go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/okf

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./...

clean:
	rm -f $(BINARY_NAME)
