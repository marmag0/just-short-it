.PHONY: help lint test

help: ## Show this list of shortcuts
	@awk 'BEGIN {FS = ":.*## "} /^[a-z-]+:.*## / {printf "make %-8s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

lint: ## Check gofmt formatting and run go vet
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "gofmt -w needed:"; echo "$$unformatted"; exit 1; fi
	go vet ./...

test: ## Run all tests, the stack tests need Docker
	go test ./...
