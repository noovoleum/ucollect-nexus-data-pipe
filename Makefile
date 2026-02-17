.PHONY: build clean run test help

# Binary name
BINARY_NAME=data-pipe

# Build directory
BUILD_DIR=.

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v ./cmd/data-pipe

clean: ## Remove build artifacts
	$(GOCLEAN)
	rm -f $(BUILD_DIR)/$(BINARY_NAME)

run: ## Run the application
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v ./cmd/data-pipe
	./$(BINARY_NAME) -config examples/config.json

test: ## Run tests
	$(GOTEST) -v ./...

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

install: ## Install the binary
	$(GOCMD) install ./cmd/data-pipe

all: clean build ## Clean and build
