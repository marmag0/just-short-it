.PHONY: help dev prod down status update test lint format

help: ## Show this list of shortcuts
	@awk 'BEGIN {FS = ":.*## "} /^## / {print substr($$0, 4)} /^[a-z-]+:.*## / {printf "make %-8s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## ----------------------------------------

dev: ## Check prerequisites and start the local stack
	@scripts/deploy.sh

prod: ## Same as dev, plus the Cloudflare tunnel
	@scripts/deploy.sh --tunnel

down: ## Stop the stack, volumes are kept
	docker compose --profile tunnel down

## ----------------------------------------

status: ## Check that the containers run and answer; ARGS=--tunnel includes the Cloudflare tunnel
	@scripts/status.sh $(ARGS)

update: ## Pull the latest commit and deploy it; ARGS=--tunnel includes the Cloudflare tunnel
	@scripts/update.sh $(ARGS)

## ----------------------------------------

test: ## Run all tests, the stack tests need Docker
	go test ./...

lint: ## Check gofmt formatting and run go vet
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "gofmt -w needed:"; echo "$$unformatted"; exit 1; fi
	go vet ./...

format: ## Format the Go code with gofmt
	gofmt -w .
