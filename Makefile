.PHONY: build test lint fmt vet tidy clean help

BINARY ?= 42
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/joaodiniz/42cli/cmd.Version=$(VERSION) \
	-X github.com/joaodiniz/42cli/cmd.Commit=$(COMMIT) \
	-X github.com/joaodiniz/42cli/cmd.BuildDate=$(DATE)

help: ## Mostra este help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

build: ## Compila o binário ./42
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test: ## Roda os testes
	go test ./...

lint: ## Roda golangci-lint
	golangci-lint run ./...

fmt: ## Formata o código
	gofmt -w .
	goimports -w . 2>/dev/null || true

vet: ## Roda go vet
	go vet ./...

tidy: ## Sincroniza go.mod / go.sum
	go mod tidy

clean: ## Remove artefatos de build
	rm -f $(BINARY) $(BINARY).exe coverage.out
