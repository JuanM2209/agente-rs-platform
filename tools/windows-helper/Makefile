.PHONY: build build-windows build-linux test tidy lint

# Default: build for the host OS.
build:
	go build -o nucleus-helper ./cmd

# Production Windows binary — stripped of debug info.
build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o nucleus-helper.exe ./cmd

# Linux binary (useful for server-side tooling or CI).
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o nucleus-helper ./cmd

# Run all tests.
test:
	go test -race -count=1 ./...

# Tidy dependencies.
tidy:
	go mod tidy

# Lint (requires golangci-lint in PATH).
lint:
	golangci-lint run ./...
