.PHONY: gen test run tidy clean help

# Build the generator from the sibling module and put it on PATH for go:generate.
BIN := $(CURDIR)/bin
export PATH := $(BIN):$(PATH)

gen: ## Build the generator and run go:generate to (re)create *_match.go
	cd ../cmd/sumx && go build -o $(BIN)/sumx .
	go generate ./...

test: ## Run the generated-Match tests
	go test -race ./...

run: ## Run the example program
	go run .

tidy: ## Tidy modules
	go mod tidy

clean: ## Remove build artifacts
	rm -rf bin

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'
