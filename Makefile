.DEFAULT_GOAL := test

export GOPROXY := off
export GOSUMDB := off
export GOTOOLCHAIN := local

FUZZ_TIME ?= 5s
FUZZ_WORKERS ?= 2

.PHONY: fmt
fmt: ## Format Go files
	gofmt -w .

.PHONY: vet
vet: ## Run static checks
	go vet ./...

.PHONY: test
test: ## Run unit tests
	go test ./...

.PHONY: race
race: ## Run tests with the race detector
	go test -race ./...

.PHONY: fuzz
fuzz: ## Run all fuzz targets
	go test -run '^$$' -fuzz '^FuzzRoundTrip$$' -fuzztime $(FUZZ_TIME) -parallel $(FUZZ_WORKERS)
	go test -run '^$$' -fuzz '^FuzzDecode$$' -fuzztime $(FUZZ_TIME) -parallel $(FUZZ_WORKERS)
	go test -run '^$$' -fuzz '^FuzzPayload$$' -fuzztime $(FUZZ_TIME) -parallel $(FUZZ_WORKERS)
	go test -run '^$$' -fuzz '^FuzzBase62$$' -fuzztime $(FUZZ_TIME) -parallel $(FUZZ_WORKERS)
	go test -run '^$$' -fuzz '^FuzzChecksum$$' -fuzztime $(FUZZ_TIME) -parallel $(FUZZ_WORKERS)

.PHONY: bench
bench: ## Run benchmarks with allocation results
	go test -run '^$$' -bench . -benchmem ./...

.PHONY: cover
cover: ## Run race tests and show coverage
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: help
help: ## Show commands and fuzz options
	@awk 'BEGIN { FS = ":.*## "; print "Usage: make <command>\n" } /^[a-zA-Z_]+:.*## / { printf "  %-8s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@printf '\nFuzz options:\n  FUZZ_TIME     Duration per target (default: %s)\n  FUZZ_WORKERS  Workers per target (default: %s)\n\nExample: make fuzz FUZZ_TIME=5m\n' '$(FUZZ_TIME)' '$(FUZZ_WORKERS)'
