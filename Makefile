.PHONY: help dev prod down status update test lint format

tunnel = $(if $(filter 1,$(TUNNEL)),--tunnel)

help: ## Show this list of shortcuts
	@awk 'BEGIN {FS = ":.*## "} /^## / {print substr($$0, 4)} /^[a-z-]+:.*## / {printf "make %-8s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## ----------------------------------------

dev: ## Check prerequisites and start the local stack
	@scripts/deploy.sh

prod: ## Same as dev, plus the Cloudflare tunnel
	@scripts/deploy.sh --tunnel

down: ## Stop the stack; VOLUME=1 also removes its volumes
	@scripts/down.sh $(if $(filter 1,$(VOLUME)),--volumes)

## ----------------------------------------

status: ## Check that the containers run and answer; TUNNEL=1 includes the Cloudflare tunnel
	@scripts/status.sh $(tunnel)

update: ## Pull the latest commit and deploy it; TUNNEL=1 includes the Cloudflare tunnel
	@scripts/update.sh $(tunnel)

## ----------------------------------------

test: ## Run all tests, the stack tests need Docker
	go test ./...

lint: ## Check gofmt formatting and run go vet
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "gofmt -w needed:"; echo "$$unformatted"; exit 1; fi
	go vet ./...

format: ## Format the Go code with gofmt
	gofmt -w .
