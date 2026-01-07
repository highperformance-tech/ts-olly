# Variables
STATICCHECK_VERSION = 2025.1.1

# Helpers
## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^//'

.PHONY: confirm
confirm:
	@echo 'Are you sure? [y/N]' && read ans && [ $${ans:-N} = y ]

# Development
## deps: install required development tools
.PHONY: deps
deps:
	@echo 'Installing development tools...'
	go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)

## run: run the ts-olly application
.PHONY: run
	@go run ./cmd/ts-olly -node node1 -logsdir ./cmd/ts-olly/testdata

## audit: tidy dependencies, then format, vet, and test all code
.PHONY: audit
audit: deps
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

# Build
## build: build the ts-olly application for the current platform
.PHONY: build
build: bin
	@echo 'Building ts-olly for this platform...'
	CGO_ENABLED=0 go build -o ./bin/ts-olly ./cmd/ts-olly

## build/all: build the ts-olly application for all relevant platforms
.PHONY: build/all
build/all: build/linux build/darwin

## build/linux: build the ts-olly application for linux_amd64
.PHONY: build/linux
build/linux: bin
	@echo 'Building ts-olly for linux...'
	GOOS=linux GOARCH=amd64 go build -o=./bin/linux_amd64/ts-olly ./cmd/ts-olly

## build/darwin: build the ts-olly application for darwin_amd64
.PHONY: build/darwin
build/darwin: bin
	@echo 'Building ts-olly for darwin...'
	GOOS=darwin GOARCH=amd64 go build -o=./bin/darwin_amd64/ts-olly ./cmd/ts-olly

# Containerize
## container: build a docker image for the ts-olly application using goreleaser
.PHONY: container
container:
	@echo 'Building docker image for ts-olly...'
	goreleaser release --snapshot --clean --skip=publish

bin:
	mkdir -p bin

# Release
## release-local: test goreleaser configuration locally (no publish)
.PHONY: release-local
release-local:
	@echo 'Testing GoReleaser configuration...'
	goreleaser check
	goreleaser release --snapshot --clean