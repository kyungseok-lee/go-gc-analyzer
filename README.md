# Go GC Analyzer

[![Go Report Card](https://goreportcard.com/badge/github.com/kyungseok-lee/go-gc-analyzer)](https://goreportcard.com/report/github.com/kyungseok-lee/go-gc-analyzer)
[![GoDoc](https://godoc.org/github.com/kyungseok-lee/go-gc-analyzer?status.svg)](https://godoc.org/github.com/kyungseok-lee/go-gc-analyzer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/dl/)

A comprehensive Go library for analyzing and monitoring garbage collection (GC) performance in Go applications. This library provides detailed insights into GC behavior, memory usage patterns, and performance metrics to help optimize your Go applications.

## 🚀 Features

- **Real-time GC Monitoring**: Continuous collection of GC metrics with configurable intervals and alerting
- **Comprehensive Analysis**: Detailed analysis of GC frequency, pause times, memory usage, and allocation patterns
- **Multiple Report Formats**: Generate reports in text, JSON, Prometheus, and summary formats
- **Health Monitoring**: Built-in health checks with configurable alert thresholds and scoring
- **Memory Trend Analysis**: Track memory usage patterns over time with detailed trend data
- **Pause Time Distribution**: Analyze GC pause time distributions and percentiles from GC events
- **Performance Recommendations**: Automated suggestions for GC performance optimization
- **Simple API**: Clean and intuitive API with single import path (`pkg/gcanalyzer`)
- **Modular Architecture**: Well-structured internal packages with clean separation of concerns
- **Zero Dependencies**: Pure Go implementation with no external dependencies
- **High Performance**: Optimized with minimal allocations, efficient sorting (using `slices` package), and graceful shutdown support
- **Thread-Safe**: All monitoring operations are safe for concurrent use

## 📦 Installation

```bash
go get github.com/kyungseok-lee/go-gc-analyzer
```

**Requirements**: Go 1.21 or later (uses `slices` package for optimized sorting)

## 🏃‍♂️ Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"
    
    "github.com/kyungseok-lee/go-gc-analyzer/pkg/gcanalyzer"
    "github.com/kyungseok-lee/go-gc-analyzer/pkg/types"
)

func main() {
    // Collect GC metrics for 10 seconds
    ctx := context.Background()
    metrics, err := gcanalyzer.CollectForDuration(ctx, 10*time.Second, time.Second)
    if err != nil {
        panic(err)
    }
    
    // Analyze the collected metrics
    analysis, err := gcanalyzer.Analyze(metrics)
    if err != nil {
        panic(err)
    }
    
    // Print analysis results
    fmt.Printf("GC Frequency: %.2f GCs/second\n", analysis.GCFrequency)
    fmt.Printf("Average Pause Time: %v\n", analysis.AvgPauseTime)
    fmt.Printf("Average Heap Size: %s\n", types.FormatBytes(analysis.AvgHeapSize))
    fmt.Printf("Allocation Rate: %s\n", types.FormatBytesRate(analysis.AllocRate))
    fmt.Printf("GC Overhead: %.2f%%\n", analysis.GCOverhead)
    
    // Generate a report
    gcanalyzer.GenerateSummaryReport(analysis, os.Stdout)
}
```

### Continuous Monitoring

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/kyungseok-lee/go-gc-analyzer/pkg/gcanalyzer"
)

func main() {
    config := &gcanalyzer.MonitorConfig{
        Interval:   time.Second,
        MaxSamples: 300, // Keep 5 minutes of data
        OnMetric: func(m *gcanalyzer.GCMetrics) {
            if m.GCCPUFraction > 0.1 {
                log.Printf("High GC CPU usage: %.2f%%", m.GCCPUFraction*100)
            }
        },
        OnGCEvent: func(e *gcanalyzer.GCEvent) {
            if e.Duration > 10*time.Millisecond {
                log.Printf("Long GC pause: %v", e.Duration)
            }
        },
    }
    
    monitor := gcanalyzer.NewMonitor(config)
    
    ctx := context.Background()
    err := monitor.Start(ctx)
    if err != nil {
        panic(err)
    }
    
    // Let it run for a while
    time.Sleep(1 * time.Minute)
    
    monitor.Stop()
    
    // Analyze collected data
    metrics := monitor.GetMetrics()
    if len(metrics) >= 2 {
        analysis, _ := gcanalyzer.Analyze(metrics)
        
        fmt.Printf("Analysis complete: %d recommendations\n", len(analysis.Recommendations))
        for _, rec := range analysis.Recommendations {
            fmt.Printf("- %s\n", rec)
        }
    }
}
```

## 📊 Monitoring Server

The library includes a ready-to-use monitoring example:

```bash
go run examples/monitoring/main.go
```

This starts a monitoring service with real-time alerting and periodic analysis.

## 📖 API Documentation

### Core Types

#### GCMetrics
Represents a snapshot of GC metrics at a specific point in time.

```go
type GCMetrics struct {
    NumGC          uint32        // Number of GCs
    PauseTotalNs   uint64        // Total pause time in nanoseconds
    HeapAlloc      uint64        // Current heap allocation
    TotalAlloc     uint64        // Total bytes allocated
    Sys            uint64        // Total bytes from OS
    GCCPUFraction  float64       // Fraction of CPU time in GC
    Timestamp      time.Time     // Collection timestamp
    // ... more fields
}
```

#### GCAnalysis
Contains comprehensive analysis results.

```go
type GCAnalysis struct {
    Period           time.Duration  // Analysis period
    GCFrequency      float64        // GCs per second
    AvgPauseTime     time.Duration  // Average pause time
    P95PauseTime     time.Duration  // 95th percentile pause time
    P99PauseTime     time.Duration  // 99th percentile pause time
    AvgHeapSize      uint64         // Average heap size
    AllocRate        float64        // Allocation rate (bytes/second)
    GCOverhead       float64        // GC CPU overhead percentage
    MemoryEfficiency float64        // Memory efficiency percentage
    Recommendations  []string       // Performance recommendations
    // ... more fields
}
```

### Main Functions

#### Collection Functions

```go
// Collect a single snapshot
func CollectOnce() *GCMetrics

// Collect for a specific duration
func CollectForDuration(ctx context.Context, duration, interval time.Duration) ([]*GCMetrics, error)
```

#### Analysis Functions

```go
// Perform analysis on metrics
func Analyze(metrics []*GCMetrics) (*GCAnalysis, error)

// Perform analysis with both metrics and events
func AnalyzeWithEvents(metrics []*GCMetrics, events []*GCEvent) (*GCAnalysis, error)

// Get memory trend data
func GetMemoryTrend(metrics []*GCMetrics) []MemoryPoint

// Get pause time distribution
func GetPauseTimeDistribution(events []*GCEvent) map[string]int
```

#### Reporting Functions

```go
// Generate various report formats
func GenerateTextReport(analysis *GCAnalysis, metrics []*GCMetrics, events []*GCEvent, w io.Writer) error
func GenerateJSONReport(analysis *GCAnalysis, metrics []*GCMetrics, events []*GCEvent, w io.Writer, indent bool) error
func GenerateSummaryReport(analysis *GCAnalysis, w io.Writer) error

// Generate health check
func GenerateHealthCheck(analysis *GCAnalysis) *HealthCheckStatus
```

#### Utility Functions (types package)

```go
// Format bytes into human-readable format (KB, MB, GB, etc.)
func FormatBytes(bytes uint64) string

// Format bytes per second into human-readable format
func FormatBytesRate(bytesPerSecond float64) string
```

## 🔧 Configuration

### Monitor Configuration

```go
type MonitorConfig struct {
    // Collection interval (default: 1 second)
    Interval time.Duration
    
    // Maximum samples to keep in memory (default: 1000)
    MaxSamples int
    
    // Alert callback function
    OnAlert func(*Alert)
    
    // Metric collection callback
    OnMetric func(*GCMetrics)
    
    // GC event callback
    OnGCEvent func(*GCEvent)
}
```

### Threshold Constants (types package)

The library provides configurable threshold constants for analysis and health checks:

```go
const (
    ThresholdGCFrequencyHigh     = 10.0                  // GCs per second
    ThresholdAvgPauseLong        = 100 * time.Millisecond
    ThresholdP99PauseVeryLong    = 500 * time.Millisecond
    ThresholdGCOverheadHigh      = 25.0                  // percentage
    ThresholdMemoryEfficiencyLow = 50.0                  // percentage
    ThresholdAllocationRateHigh  = 100 * 1024 * 1024     // 100 MB/s
)
```

## 📈 Understanding the Metrics

### GC Frequency
- **Low (< 1 GC/s)**: Excellent, minimal GC pressure
- **Medium (1-5 GC/s)**: Good, normal application behavior
- **High (> 5 GC/s)**: Consider optimization, reduce allocation rate

### Pause Times
- **Excellent (< 1ms)**: Low-latency applications
- **Good (1-10ms)**: Most applications
- **Needs attention (> 10ms)**: May impact responsiveness
- **Critical (> 100ms)**: Immediate optimization needed

### GC Overhead
- **Excellent (< 5%)**: Minimal GC impact
- **Good (5-15%)**: Acceptable for most applications
- **High (15-25%)**: Consider tuning
- **Critical (> 25%)**: Significant performance impact

### Memory Efficiency
- **Excellent (> 80%)**: Efficient memory usage
- **Good (60-80%)**: Normal usage
- **Poor (< 60%)**: Memory fragmentation or inefficient allocation patterns

## 🎯 Performance Optimization Tips

Based on the analysis results, here are common optimization strategies:

### High GC Frequency
- Reduce allocation rate by reusing objects
- Use object pools for frequently allocated objects
- Increase `GOGC` value to trigger GC less frequently
- Optimize data structures to reduce pointer indirection

### Long Pause Times
- Reduce heap size if possible
- Minimize large object allocations
- Use streaming processing instead of batching
- Consider concurrent GC tuning (Go 1.19+)

### High GC Overhead
- Profile allocation hotspots with `go tool pprof`
- Implement object pooling
- Use value types instead of pointer types where possible
- Optimize slice and map usage patterns

### Memory Leaks
- Check for goroutine leaks
- Ensure proper cleanup of resources
- Use weak references where appropriate
- Monitor memory growth trends over time

## 🏷️ Examples

The library includes comprehensive examples:

- **[Basic Usage](examples/basic/main.go)**: Simple collection and analysis
- **[Advanced Features](examples/advanced/main.go)**: Workload analysis, performance comparison
- **[Monitoring Service](examples/monitoring/main.go)**: Continuous monitoring with alerts

Run examples:

```bash
# Basic example
go run examples/basic/main.go

# Advanced features
go run examples/advanced/main.go

# Monitoring service
go run examples/monitoring/main.go
```

## 🧪 Testing

Run the complete test suite:

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run benchmarks
go test -bench=. ./tests

# Run with race detection
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📊 Benchmarks

The library is designed for minimal overhead (Apple M1 Pro):

```
BenchmarkCollectOnce-10                    42912     26028 ns/op    4336 B/op     3 allocs/op
BenchmarkAnalyzer_Analyze-10             2212604       546 ns/op     752 B/op     3 allocs/op
BenchmarkAnalyzer_GetMemoryTrend-10      1415532       828 ns/op    4864 B/op     1 allocs/op
BenchmarkReporter_GenerateTextReport-10   404581      3003 ns/op    1985 B/op    41 allocs/op
BenchmarkReporter_GenerateHealthCheck-10 12011064      112 ns/op     192 B/op     2 allocs/op
```

Performance characteristics:
- **CollectOnce**: ~26μs per collection (includes runtime.ReadMemStats)
- **Analysis**: ~546ns with optimized sorting using `slices.SortFunc`
- **Reporting**: Fast generation with reduced allocations
- **Health Check**: Sub-microsecond generation (~112ns)
- **Memory overhead**: Minimal, configurable retention with graceful cleanup

## 🔌 Integration

### Prometheus/Grafana

Export metrics in Prometheus format:

```go
reporter := reporting.New(analysis, metrics, nil)
err := reporter.GenerateGrafanaMetrics(w)
```

### JSON APIs

All data structures are JSON-serializable for easy integration:

```go
analysis, _ := gcanalyzer.Analyze(metrics)
data, _ := json.Marshal(analysis)
```

### Health Checks

Integrate with health check systems:

```go
healthCheck := gcanalyzer.GenerateHealthCheck(analysis)
if healthCheck.Status != "healthy" {
    // Alert or take action
}
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development Setup

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for your changes
5. Run the test suite (`go test ./...`)
6. Commit your changes (`git commit -am 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Guidelines

- Write clear, self-documenting code
- Add tests for new functionality
- Update documentation as needed
- Follow Go best practices and idioms
- Ensure backward compatibility when possible

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Go team for the excellent runtime metrics APIs
- The Go community for inspiration and feedback
- Contributors who help improve this library

## 📞 Support

- 📖 [Documentation](https://godoc.org/github.com/kyungseok-lee/go-gc-analyzer)
- 🐛 [Issue Tracker](https://github.com/kyungseok-lee/go-gc-analyzer/issues)
- 💬 [Discussions](https://github.com/kyungseok-lee/go-gc-analyzer/discussions)

---

**Made with ❤️ for the Go community**
