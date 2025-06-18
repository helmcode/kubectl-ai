BINARY_NAME=kubectl-ai
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=${VERSION}"

.PHONY: all
all: test build

.PHONY: build
build:
	go build ${LDFLAGS} -o ${BINARY_NAME} .

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	go clean
	rm -f ${BINARY_NAME}
	rm -rf dist/

.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: install
install: build
	mkdir -p ${HOME}/.local/bin
	cp ${BINARY_NAME} ${HOME}/.local/bin/
	@echo "Installed to ${HOME}/.local/bin/${BINARY_NAME}"
	@echo "Make sure ${HOME}/.local/bin is in your PATH"

.PHONY: install-global
install-global: build
	sudo cp ${BINARY_NAME} /usr/local/bin/
	@echo "Installed to /usr/local/bin/${BINARY_NAME}"

.PHONY: run
run: build
	./${BINARY_NAME}

# Development helpers
.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run

# Cross compilation
.PHONY: build-all
build-all:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-windows-amd64.exe .

.PHONY: release-dry-run
release-dry-run:
	goreleaser release --snapshot --clean

# Quick test during development
.PHONY: test-local
test-local: build
	./${BINARY_NAME} debug "test problem" -r deployment/nginx -n default
