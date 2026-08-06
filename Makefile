# open-faith-map — task runner for developers.
#
# Run `make` (or `make help`) to see every target. These are a friendly front door: they delegate
# to the tools that actually own each job — the bundled gödel wrapper (./godelw) for
# build/test/format/lint, and docker compose for the local stack — so there is no second source of
# truth to drift. Mirrors go-oikumenea's own Makefile shape (docs/architecture/decisions.md#d-stack).

-include .env
export

GODEL   ?= ./godelw
COMPOSE ?= docker compose

.DEFAULT_GOAL := help

# ---------------------------------------------------------------------------------------------------
##@ General
# ---------------------------------------------------------------------------------------------------

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

# ---------------------------------------------------------------------------------------------------
##@ Build
# ---------------------------------------------------------------------------------------------------

.PHONY: build
build: ## Build the openfaithmap-api binary (-> out/build/)
	$(GODEL) build openfaithmap-api

.PHONY: install
install: ## Install openfaithmap-api into $GOBIN (on your PATH)
	go install ./cmd/openfaithmap-api

.PHONY: run
run: ## Run openfaithmap-api locally (serve, foreground)
	go run ./cmd/openfaithmap-api serve

.PHONY: clean
clean: ## Remove build/dist outputs
	$(GODEL) clean || true
	rm -rf out

# ---------------------------------------------------------------------------------------------------
##@ Verify
# ---------------------------------------------------------------------------------------------------

.PHONY: format
format: ## Format Go source (gödel)
	$(GODEL) format

.PHONY: lint
lint: ## Lint Go source (golangci-lint via gödel)
	$(GODEL) verify --apply=false --skip-test

.PHONY: test
test: ## Run Go tests
	$(GODEL) test

.PHONY: verify
verify: ## Format + lint + test — the same gate CI runs
	$(GODEL) verify

# ---------------------------------------------------------------------------------------------------
##@ Docker
# ---------------------------------------------------------------------------------------------------

.PHONY: docker-build
docker-build: ## Build the openfaithmap-api image
	docker build -t openfaithmap-api:local .

.PHONY: up
up: ## Start the local stack (openfaithmap-postgres + openfaithmap-api)
	$(COMPOSE) up --build

.PHONY: down
down: ## Stop the local stack
	$(COMPOSE) down
