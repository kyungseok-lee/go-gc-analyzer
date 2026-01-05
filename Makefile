# Go GC Analyzer - Build & Development Makefile
# ==============================================================================

.PHONY: all build test bench bench-compare profile clean lint fmt help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOVET=$(GOCMD) vet
GOFMT=gofmt
GOMOD=$(GOCMD) mod
BENCHSTAT=~/go/bin/benchstat

# Directories
BENCHMARK_DIR=benchmarks
PROFILE_DIR=profiles

# Default target
all: lint test

# ==============================================================================
# Build & Run
# ==============================================================================

build: ## Build all packages
	$(GOBUILD) ./...

run-basic: ## Run basic example
	$(GOCMD) run ./examples/basic/main.go

run-advanced: ## Run advanced example
	$(GOCMD) run ./examples/advanced/main.go

run-monitoring: ## Run monitoring example
	$(GOCMD) run ./examples/monitoring/main.go

# ==============================================================================
# Testing
# ==============================================================================

test: ## Run all tests
	$(GOTEST) -v ./...

test-race: ## Run tests with race detector
	$(GOTEST) -v -race ./...

test-cover: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ==============================================================================
# Benchmarking
# ==============================================================================

bench: ## Run benchmarks
	@mkdir -p $(BENCHMARK_DIR)
	$(GOTEST) -bench=. -benchmem -count=6 ./tests/... 2>&1 | tee $(BENCHMARK_DIR)/bench_$$(date +%Y%m%d_%H%M%S).txt

bench-short: ## Run quick benchmarks (count=1)
	$(GOTEST) -bench=. -benchmem -count=1 ./tests/...

bench-save: ## Run benchmarks and save as baseline
	@mkdir -p $(BENCHMARK_DIR)
	$(GOTEST) -bench=. -benchmem -count=6 ./tests/... > $(BENCHMARK_DIR)/baseline.txt
	@echo "Baseline saved: $(BENCHMARK_DIR)/baseline.txt"

bench-compare: ## Compare current benchmarks to baseline
	@mkdir -p $(BENCHMARK_DIR)
	$(GOTEST) -bench=. -benchmem -count=6 ./tests/... > $(BENCHMARK_DIR)/current.txt
	$(BENCHSTAT) $(BENCHMARK_DIR)/baseline.txt $(BENCHMARK_DIR)/current.txt

bench-cpu: ## Run benchmarks with CPU profile
	@mkdir -p $(PROFILE_DIR)
	$(GOTEST) -bench=BenchmarkRealWorldScenario -benchmem -cpuprofile=$(PROFILE_DIR)/cpu.prof ./tests/...
	@echo "CPU profile: $(PROFILE_DIR)/cpu.prof"
	@echo "View with: go tool pprof -http=:8080 $(PROFILE_DIR)/cpu.prof"

bench-mem: ## Run benchmarks with memory profile
	@mkdir -p $(PROFILE_DIR)
	$(GOTEST) -bench=BenchmarkMemoryUsage -benchmem -memprofile=$(PROFILE_DIR)/mem.prof ./tests/...
	@echo "Memory profile: $(PROFILE_DIR)/mem.prof"
	@echo "View with: go tool pprof -http=:8080 $(PROFILE_DIR)/mem.prof"

# ==============================================================================
# Profiling
# ==============================================================================

profile-cpu: ## Generate CPU profile from advanced example
	@mkdir -p $(PROFILE_DIR)
	$(GOCMD) run ./examples/advanced/main.go -cpuprofile=$(PROFILE_DIR)/app_cpu.prof 2>/dev/null || true
	@if [ -f $(PROFILE_DIR)/app_cpu.prof ]; then \
		echo "View with: go tool pprof -http=:8080 $(PROFILE_DIR)/app_cpu.prof"; \
	fi

profile-mem: ## Generate memory profile
	@mkdir -p $(PROFILE_DIR)
	$(GOCMD) run ./examples/advanced/main.go -memprofile=$(PROFILE_DIR)/app_mem.prof 2>/dev/null || true
	@if [ -f $(PROFILE_DIR)/app_mem.prof ]; then \
		echo "View with: go tool pprof -http=:8080 $(PROFILE_DIR)/app_mem.prof"; \
	fi

pprof-cpu: bench-cpu ## Run CPU profile and open in browser
	$(GOCMD) tool pprof -http=:8080 $(PROFILE_DIR)/cpu.prof

pprof-mem: bench-mem ## Run memory profile and open in browser
	$(GOCMD) tool pprof -http=:8080 $(PROFILE_DIR)/mem.prof

# ==============================================================================
# GC Trace & Debug
# ==============================================================================

gctrace: ## Run with GC tracing enabled
	GODEBUG=gctrace=1 $(GOCMD) run ./examples/advanced/main.go 2>&1 | head -50

schedtrace: ## Run with scheduler tracing
	GODEBUG=schedtrace=1000 $(GOCMD) run ./examples/advanced/main.go 2>&1 | head -50

gcpacertrace: ## Run with GC pacer tracing
	GODEBUG=gcpacertrace=1 $(GOCMD) run ./examples/advanced/main.go 2>&1 | head -50

# ==============================================================================
# Code Quality
# ==============================================================================

lint: ## Run linters
	$(GOVET) ./...
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt: ## Format code
	$(GOFMT) -s -w .

fmt-check: ## Check formatting
	@if [ "$$($(GOFMT) -s -l . | wc -l)" -gt 0 ]; then \
		echo "Code is not formatted:"; \
		$(GOFMT) -s -l .; \
		exit 1; \
	fi

# ==============================================================================
# Dependencies
# ==============================================================================

deps: ## Download dependencies
	$(GOMOD) download

deps-update: ## Update dependencies
	$(GOMOD) tidy

deps-tools: ## Install development tools
	$(GOCMD) install golang.org/x/perf/cmd/benchstat@latest
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# ==============================================================================
# Cleanup
# ==============================================================================

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -rf $(BENCHMARK_DIR)/*.txt
	rm -rf $(PROFILE_DIR)/*.prof
	rm -f coverage.out coverage.html

clean-all: clean ## Clean everything including profiles
	rm -rf $(BENCHMARK_DIR)
	rm -rf $(PROFILE_DIR)

# ==============================================================================
# Help
# ==============================================================================

help: ## Show this help
	@echo "Go GC Analyzer - Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make bench           # Run benchmarks"
	@echo "  make bench-compare   # Compare to baseline"
	@echo "  make pprof-cpu       # CPU profile in browser"
	@echo "  make gctrace         # Run with GC tracing"

