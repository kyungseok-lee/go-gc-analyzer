# Go GC Analyzer

A comprehensive Go Garbage Collection performance analyzer for monitoring, analyzing, and optimizing GC behavior.

## Overview

This library provides tools to:

- **Monitor** GC metrics in real-time with configurable callbacks
- **Analyze** GC performance patterns and identify bottlenecks
- **Report** findings in multiple formats (text, JSON, Prometheus)
- **Recommend** optimizations based on collected data

## Features

- 📊 Real-time GC metrics collection
- 📈 Comprehensive GC analysis with pause time percentiles (P95/P99)
- 🔔 Alert callbacks for performance issues
- 📝 Multiple report formats (text, JSON, summary, Prometheus)
- 🏥 Health check scoring system
- 🔄 Memory trend analysis
- 💡 Automatic optimization recommendations

## Installation

```bash
go get github.com/kyungseok-lee/go-gc-analyzer
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"
    
    "github.com/kyungseok-lee/go-gc-analyzer/pkg/gcanalyzer"
)

func main() {
    ctx := context.Background()

    // Collect metrics for 10 seconds
    metrics, err := gcanalyzer.CollectForDuration(ctx, 10*time.Second, time.Second)
    if err != nil {
        panic(err)
    }
    
    // Analyze collected data
    analysis, err := gcanalyzer.Analyze(metrics)
    if err != nil {
        panic(err)
    }
    
    // Generate report
    gcanalyzer.GenerateSummaryReport(analysis, os.Stdout)

    // Get health status
    health := gcanalyzer.GenerateHealthCheck(analysis)
    fmt.Printf("GC Health Score: %d/100 (%s)\n", health.Score, health.Status)
}
```

### Continuous Monitoring with Alerts

```go
monitor := gcanalyzer.NewMonitor(&gcanalyzer.MonitorConfig{
        Interval:   time.Second,
    MaxSamples: 1000,
    OnAlert: func(alert *gcanalyzer.Alert) {
        log.Printf("[%s] %s: %s", alert.Severity, alert.Type, alert.Message)
    },
        OnMetric: func(m *gcanalyzer.GCMetrics) {
        log.Printf("Heap: %s, GC#: %d", 
            types.FormatBytes(m.HeapAlloc), m.NumGC)
    },
})

ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
defer cancel()

monitor.Start(ctx)
defer monitor.Stop()

// Your application logic here...
```

## API Reference

### Core Functions

| Function | Description |
|----------|-------------|
| `CollectOnce()` | Collect a single GC metrics snapshot |
| `CollectForDuration(ctx, duration, interval)` | Collect metrics over a time period |
| `Analyze(metrics)` | Perform comprehensive GC analysis |
| `AnalyzeWithEvents(metrics, events)` | Analyze with detailed event data |
| `GenerateTextReport(analysis, w)` | Generate detailed text report |
| `GenerateJSONReport(analysis, w, indent)` | Generate JSON report |
| `GenerateSummaryReport(analysis, w)` | Generate concise summary |
| `GenerateHealthCheck(analysis)` | Generate health check status |

### Metrics Types

```go
// GCMetrics - comprehensive GC statistics
type GCMetrics struct {
    NumGC         uint32    // Total GC cycles
    PauseTotalNs  uint64    // Total pause time
    HeapAlloc     uint64    // Allocated heap bytes
    HeapSys       uint64    // System heap bytes
    GCCPUFraction float64   // GC CPU usage fraction
    // ... more fields
}

// GCAnalysis - analyzed results
type GCAnalysis struct {
    GCFrequency   float64       // GCs per second
    AvgPauseTime  time.Duration // Average pause time
    P95PauseTime  time.Duration // 95th percentile pause
    P99PauseTime  time.Duration // 99th percentile pause
    AllocRate     float64       // Bytes allocated per second
    GCOverhead    float64       // GC CPU percentage
    Recommendations []string    // Optimization suggestions
}
```

---

## Performance Benchmarking & Profiling Guide

### Running Benchmarks

```bash
# Run all benchmarks
make bench

# Quick benchmark (single run)
make bench-short

# Save baseline for comparison
make bench-save

# Compare current vs baseline
make bench-compare
```

### Manual Benchmark Commands

```bash
# Full benchmark with 6 iterations
go test -bench=. -benchmem -count=6 ./tests/... | tee bench.txt

# Compare two benchmark files
benchstat baseline.txt current.txt
```

### CPU Profiling

```bash
# Generate CPU profile
make bench-cpu

# View in browser
make pprof-cpu
# or manually:
go tool pprof -http=:8080 profiles/cpu.prof
```

### Memory Profiling

```bash
# Generate memory profile
make bench-mem

# View in browser
make pprof-mem
# or manually:
go tool pprof -http=:8080 profiles/mem.prof
```

### GC Tracing

```bash
# Enable GC tracing
GODEBUG=gctrace=1 go run ./examples/advanced/main.go

# Scheduler tracing
GODEBUG=schedtrace=1000 go run ./examples/advanced/main.go

# GC pacer tracing
GODEBUG=gcpacertrace=1 go run ./examples/advanced/main.go
```

### Understanding gctrace Output

```
gc 1 @0.012s 2%: 0.015+0.89+0.003 ms clock, 0.12+0.45/0.67/0+0.024 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
```

| Field | Meaning |
|-------|---------|
| `gc 1` | GC cycle number |
| `@0.012s` | Time since program start |
| `2%` | CPU time spent in GC |
| `0.015+0.89+0.003 ms clock` | Wall clock: STW mark + concurrent + STW sweep |
| `4->4->0 MB` | Heap before -> after -> live |
| `5 MB goal` | Target heap size |
| `8 P` | Number of processors used |

---

## Performance Optimization Results

### Benchmark Comparison (Before → After)

| Benchmark | Time | Memory | Allocations |
|-----------|------|--------|-------------|
| GenerateTextReport | **-44%** | - | **-71%** (41→12) |
| MemoryUsage | **-20%** | - | **-67%** (45→15) |
| Analyzer_Analyze | **-5%** | **-55%** | **-33%** |
| AnalyzeSmallDataset | - | **-98%** | **-33%** |
| RealWorldScenario | - | -2% | **-17%** |
| **Overall (geomean)** | **-7.6%** | **-31%** | **-24%** |

### Key Optimizations Applied

1. **sync.Pool for Reusable Slices**
   - Duration slices in analyzer
   - strings.Builder in reporter
   - Reduces GC pressure significantly

2. **Pre-allocated Capacities**
   - Recommendations slice: `make([]string, 0, 8)`
   - Health check issues: `make([]string, 0, 6)`
   - Avoids slice growth reallocations

3. **strings.Builder over fmt.Sprintf**
   - Text report generation optimized
   - 71% fewer allocations

4. **Lightweight Metrics Collection**
   - `NewGCMetricsLite()` - skips pause data (~4KB savings)
   - `NewGCMetricsPooled()` - reuses pause slices

---

## Runtime Tuning Guide

### GOGC (GC Target Percentage)

```bash
# Default is 100 (GC when heap doubles)
GOGC=100 ./myapp

# More aggressive GC (lower latency, more CPU)
GOGC=50 ./myapp

# Less aggressive GC (lower CPU, higher memory)
GOGC=200 ./myapp
```

### GOMEMLIMIT (Memory Limit)

```bash
# Set soft memory limit (Go 1.19+)
GOMEMLIMIT=1GiB ./myapp
```

### Recommended Settings by Use Case

| Use Case | GOGC | GOMEMLIMIT | Notes |
|----------|------|------------|-------|
| Low latency | 50-100 | Auto | More frequent, shorter GC |
| High throughput | 200-400 | Set | Less GC overhead |
| Memory constrained | 50 | Set limit | Prevent OOM |
| Batch processing | 400+ | Set limit | Minimize GC interruptions |

---

## Project Structure

```
go-gc-analyzer/
├── pkg/
│   ├── gcanalyzer/    # Public API
│   │   └── api.go
│   └── types/         # Shared types
│       ├── metrics.go
│       ├── constants.go
│       ├── errors.go
│       └── format.go
├── internal/
│   ├── analysis/      # GC analysis logic
│   ├── collector/     # Metrics collection
│   └── reporting/     # Report generation
├── examples/
│   ├── basic/         # Simple usage example
│   ├── advanced/      # Advanced features
│   └── monitoring/    # Continuous monitoring
├── tests/
│   ├── analyzer_test.go
│   ├── benchmark_test.go
│   ├── collector_test.go
│   └── integration_test.go
├── benchmarks/        # Benchmark results
├── profiles/          # Profiling outputs
├── Makefile           # Build & dev commands
└── README.md
```

---

## Examples

### Run Examples

```bash
# Basic example
go run ./examples/basic/main.go

# Advanced example
go run ./examples/advanced/main.go

# Monitoring example
go run ./examples/monitoring/main.go
```

---

## Development

### Prerequisites

- Go 1.23+
- Make (optional, for convenience commands)

### Setup

```bash
# Clone repository
git clone https://github.com/kyungseok-lee/go-gc-analyzer.git
cd go-gc-analyzer

# Install dev tools
make deps-tools

# Run tests
make test

# Run benchmarks
make bench
```

### Available Make Targets

```bash
make help        # Show all available commands
make test        # Run tests
make bench       # Run benchmarks
make lint        # Run linters
make fmt         # Format code
make clean       # Clean artifacts
```

---

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Run tests and benchmarks
4. Commit your changes with proper message format:
   - `perf:` for performance improvements
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation
   - `test:` for test additions
5. Push to the branch
6. Create a Pull Request with benchmark comparison

---

## License

MIT License - see LICENSE file for details.

---

## References

- [Go GC Guide](https://tip.golang.org/doc/gc-guide)
- [runtime package documentation](https://pkg.go.dev/runtime)
- [pprof documentation](https://pkg.go.dev/runtime/pprof)
- [GODEBUG environment variable](https://pkg.go.dev/runtime#hdr-Environment_Variables)
