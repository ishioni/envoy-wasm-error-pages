.PHONY: help build build-docker clean version dev up down logs restart test-errors test-headers

# Version defaults to git SHA (determined on host), but can be overridden
# This is calculated here and passed to Docker, avoiding the need for .git in the image
VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")

# Docker image name
IMAGE_NAME ?= envoy-wasm-error-pages
IMAGE_TAG ?= $(VERSION)

# Build flags
GOOS := wasip1
GOARCH := wasm
BUILDMODE := c-shared
LDFLAGS := -X main.version=$(VERSION)

# Output files
WASM_OUTPUT := main.wasm
DOCKER_WASM_OUTPUT := plugin.wasm

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the WASM plugin locally
	@echo "Building WASM plugin (version: $(VERSION))..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -buildmode=$(BUILDMODE) -ldflags "$(LDFLAGS)" -o $(WASM_OUTPUT) main.go
	@echo "Build complete: $(WASM_OUTPUT)"

build-docker: ## Build Docker image with the WASM plugin (auto-passes VERSION)
	@echo "Building Docker image (version: $(VERSION))..."
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@if [ "$(IMAGE_TAG)" != "latest" ]; then \
		docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_NAME):latest; \
	fi
	@echo "Docker image built: $(IMAGE_NAME):$(IMAGE_TAG)"

build-docker-version: ## Build Docker image with custom version (use VERSION=x.y.z)
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "$$(git rev-parse --short HEAD 2>/dev/null || echo 'dev')" ]; then \
		echo "Error: Please specify VERSION (e.g., make build-docker-version VERSION=1.0.0)"; \
		exit 1; \
	fi
	@$(MAKE) build-docker VERSION=$(VERSION) IMAGE_TAG=$(VERSION)

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	rm -f $(WASM_OUTPUT)
	@echo "Clean complete"

version: ## Show current version
	@echo "$(VERSION)"

test: ## Run tests
	go test -v ./...

fmt: ## Format Go code
	go fmt ./...

lint: ## Run linter (requires golangci-lint)
	golangci-lint run

dev: up ## Start local development environment (alias for 'up')

up: ## Start docker-compose with Envoy, WASM plugin, and debug backend
	@echo "Starting development environment (version: $(VERSION))..."
	VERSION=$(VERSION) docker-compose up --build

down: ## Stop and remove docker-compose containers
	@echo "Stopping development environment..."
	VERSION=$(VERSION) docker-compose down -v

logs: ## Follow logs from all services
	VERSION=$(VERSION) docker-compose logs -f

restart: ## Restart the development environment
	@$(MAKE) down
	@$(MAKE) up

test-errors: ## Test error pages (requires running environment)
	@echo "Testing error pages..."
	@echo "\n==> Testing 200 (should pass through):"
	@curl -s http://localhost:10000/200 | head -n 5 || true
	@echo "\n==> Testing 400 (should show custom 4xx page):"
	@curl -s http://localhost:10000/400 | grep -o '<h1>.*</h1>' || true
	@echo "\n==> Testing 404 (should show custom 4xx page):"
	@curl -s http://localhost:10000/404 | grep -o '<h1>.*</h1>' || true
	@echo "\n==> Testing 500 (should show custom 5xx page):"
	@curl -s http://localhost:10000/500 | grep -o '<h1>.*</h1>' || true
	@echo "\n==> Testing 503 (should show custom 5xx page):"
	@curl -s http://localhost:10000/503 | grep -o '<h1>.*</h1>' || true
	@echo "\nDone! Visit http://localhost:10000/500 in your browser to see the full page."

test-headers: ## Test X-App-Id header injection (requires running environment)
	@echo "Testing X-App-Id header injection..."
	@echo "\n==> Testing exact match (tr.movishell.pl):"
	@curl -s -H "Host: tr.movishell.pl" http://localhost:10000/200 | grep "X-App-Id:" || echo "  No header found"
	@echo "\n==> Testing exact match (pl.movishell.pl):"
	@curl -s -H "Host: pl.movishell.pl" http://localhost:10000/200 | grep "X-App-Id:" || echo "  No header found"
	@echo "\n==> Testing wildcard match (abc.test.movishell.pl):"
	@curl -s -H "Host: abc.test.movishell.pl" http://localhost:10000/200 | grep "X-App-Id:" || echo "  No header found"
	@echo "\n==> Testing wildcard match (random.movishell.pl):"
	@curl -s -H "Host: random.movishell.pl" http://localhost:10000/200 | grep "X-App-Id:" || echo "  No header found"
	@echo "\n==> Testing no match (example.com):"
	@curl -s -H "Host: example.com" http://localhost:10000/200 | grep "X-App-Id:" || echo "  No header found (expected)"
	@echo "\nDone! All X-App-Id header tests complete."

.DEFAULT_GOAL := help
