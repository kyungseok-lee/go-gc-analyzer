# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Unit tests for internal packages
- GitHub Actions CI workflow
- golangci-lint configuration
- This CHANGELOG file

## [0.1.0] - 2026-01-06

### Added
- **Core Features**
  - GC metrics collection (`NewGCMetrics`, `NewGCMetricsPooled`, `NewGCMetricsLite`)
  - Comprehensive GC analysis with pause time percentiles (P95/P99)
  - Memory trend analysis and leak detection heuristics
  - Real-time monitoring with configurable alerts

- **Reporting**
  - Text report format with detailed GC statistics
  - JSON report format with configurable options
  - Summary report for quick overview
  - Table report for metric history
  - Prometheus/Grafana metrics format
  - Health check scoring system (0-100)

- **API**
  - `CollectOnce()` - Single snapshot collection
  - `CollectForDuration()` - Duration-based collection
  - `Analyze()` / `AnalyzeWithEvents()` - GC performance analysis
  - `GenerateTextReport()` / `GenerateJSONReport()` - Report generation
  - `GenerateHealthCheck()` - Health status evaluation
  - `NewMonitor()` - Continuous monitoring with callbacks

- **Examples**
  - Basic usage example (`examples/basic/`)
  - Advanced usage with workload patterns (`examples/advanced/`)
  - Continuous monitoring example (`examples/monitoring/`)

- **Developer Tools**
  - Comprehensive Makefile with bench, profile, lint targets
  - Benchmark suite for performance validation
  - Documentation in English and Korean

### Performance
- 44% faster text report generation using `strings.Builder`
- 55% reduction in memory allocations for analysis operations
- 71% fewer allocations in report generation
- `sync.Pool` optimization for pause time slices
- Pre-allocated slice capacities for recommendations and issues

### Technical Details
- Pure Go implementation with zero external dependencies
- Go 1.23+ required
- Thread-safe collector with `sync.RWMutex`
- Graceful shutdown with `sync.WaitGroup`

---

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 0.1.0 | 2026-01-06 | Initial release |

[Unreleased]: https://github.com/kyungseok-lee/go-gc-analyzer/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/kyungseok-lee/go-gc-analyzer/releases/tag/v0.1.0

