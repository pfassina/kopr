.PHONY: build run test test-integration lint clean docker

BINARY := kopr
BUILD_DIR := bin
LDFLAGS := -s -w

build:
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/kopr

run: build
	$(BUILD_DIR)/$(BINARY) $(ARGS)

test:
	go test ./...

test-integration:
	go test -tags integration ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)

docker:
	docker build -t kopr .
