VERSION := $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)

install:
	go mod download

build:
	CGO_ENABLED=0 go build -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT}" -o datamatic ./datamatic.go

test:
	go test -coverprofile=coverage.out -v ./...

lint:
	golangci-lint run --verbose

lint-fix:
	golangci-lint run --verbose --fix

coverage: test
	go tool cover -html=coverage.out
