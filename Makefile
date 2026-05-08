VERSION ?= 0.1.0
LDFLAGS := -s -w -X github.com/andrey1555251-ux/mindforge/internal/version.Version=$(VERSION)

.PHONY: all build windows test lint vet clean run

all: test build

build:
	go build -ldflags "$(LDFLAGS)" -o dist/mindforge .

windows:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		go build -ldflags "$(LDFLAGS)" -o dist/mindforge.exe .

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf dist

run: build
	./dist/mindforge welcome
