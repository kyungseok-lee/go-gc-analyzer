package tests

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/pkg/gcanalyzer"
	"github.com/kyungseok-lee/go-gc-analyzer/pkg/types"
)

// =============================================================================
// Metric Collection Benchmarks
// =============================================================================

// BenchmarkCollectOnce_Benchmark measures the performance of single metric collection
func BenchmarkCollectOnce_Benchmark(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := gcanalyzer.CollectOnce()
		if metrics == nil {
			b.Fatal("Expected metrics")
		}
	}
}

// BenchmarkNewGCMetrics measures the performance of GCMetrics creation
func BenchmarkNewGCMetrics(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := gcanalyzer.CollectOnce()
		if metrics == nil {
			b.Fatal("Expected metrics")
		}
	}
}

// BenchmarkNewGCMetricsLite measures the performance of lightweight metrics creation
func BenchmarkNewGCMetricsLite(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := types.NewGCMetricsLite()
		if metrics == nil {
			b.Fatal("Expected metrics")
		}
	}
}

// BenchmarkNewGCMetricsPooled measures the performance of pooled metrics creation
func BenchmarkNewGCMetricsPooled(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := types.NewGCMetricsPooled()
		if metrics == nil {
			b.Fatal("Expected metrics")
		}
		metrics.Release() // Return to pool
	}
}

// =============================================================================
// Analyzer Benchmarks
// =============================================================================

// BenchmarkAnalyzer_Analyze_Benchmark measures the performance of GC analysis
func BenchmarkAnalyzer_Analyze_Benchmark(b *testing.B) {
	// Create test dataset
	metrics := generateTestMetrics(100)
	events := generateTestEvents(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analysis, err := gcanalyzer.AnalyzeWithEvents(metrics, events)
		if err != nil {
			b.Fatal(err)
		}
		if analysis == nil {
			b.Fatal("Expected analysis")
		}
	}
}

// BenchmarkAnalyzer_AnalyzeSmallDataset measures analysis performance with small dataset
func BenchmarkAnalyzer_AnalyzeSmallDataset(b *testing.B) {
	metrics := generateTestMetrics(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analysis, err := gcanalyzer.Analyze(metrics)
		if err != nil {
			b.Fatal(err)
		}
		if analysis == nil {
			b.Fatal("Expected analysis")
		}
	}
}

// BenchmarkAnalyzer_AnalyzeMediumDataset measures analysis performance with medium dataset
func BenchmarkAnalyzer_AnalyzeMediumDataset(b *testing.B) {
	metrics := generateTestMetrics(100)
	events := generateTestEvents(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analysis, err := gcanalyzer.AnalyzeWithEvents(metrics, events)
		if err != nil {
			b.Fatal(err)
		}
		if analysis == nil {
			b.Fatal("Expected analysis")
		}
	}
}

// BenchmarkAnalyzer_AnalyzeLargeDataset measures analysis performance with large dataset
func BenchmarkAnalyzer_AnalyzeLargeDataset(b *testing.B) {
	metrics := generateTestMetrics(1000)
	events := generateTestEvents(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analysis, err := gcanalyzer.AnalyzeWithEvents(metrics, events)
		if err != nil {
			b.Fatal(err)
		}
		if analysis == nil {
			b.Fatal("Expected analysis")
		}
	}
}

// BenchmarkAnalyzer_GetMemoryTrend measures memory trend calculation performance
func BenchmarkAnalyzer_GetMemoryTrend(b *testing.B) {
	metrics := generateTestMetrics(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trend := gcanalyzer.GetMemoryTrend(metrics)
		if len(trend) == 0 {
			b.Fatal("Expected memory trend data")
		}
	}
}

// BenchmarkAnalyzer_GetPauseTimeDistribution measures pause time distribution calculation
func BenchmarkAnalyzer_GetPauseTimeDistribution(b *testing.B) {
	events := generateTestEvents(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		distribution := gcanalyzer.GetPauseTimeDistribution(events)
		if len(distribution) == 0 {
			b.Fatal("Expected distribution data")
		}
	}
}

// =============================================================================
// Reporter Benchmarks
// =============================================================================

// BenchmarkReporter_GenerateTextReport measures text report generation performance
func BenchmarkReporter_GenerateTextReport(b *testing.B) {
	metrics := generateTestMetrics(50)
	events := generateTestEvents(25)

	analysis, _ := gcanalyzer.AnalyzeWithEvents(metrics, events)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		err := gcanalyzer.GenerateTextReport(analysis, metrics, events, &buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReporter_GenerateJSONReport measures JSON report generation performance
func BenchmarkReporter_GenerateJSONReport(b *testing.B) {
	metrics := generateTestMetrics(50)
	events := generateTestEvents(25)

	analysis, _ := gcanalyzer.AnalyzeWithEvents(metrics, events)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		err := gcanalyzer.GenerateJSONReport(analysis, metrics, events, &buf, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReporter_GenerateSummaryReport measures summary report generation performance
func BenchmarkReporter_GenerateSummaryReport(b *testing.B) {
	metrics := generateTestMetrics(50)
	analysis, _ := gcanalyzer.Analyze(metrics)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		err := gcanalyzer.GenerateSummaryReport(analysis, &buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReporter_GenerateHealthCheck measures health check generation performance
func BenchmarkReporter_GenerateHealthCheck(b *testing.B) {
	metrics := generateTestMetrics(50)

	analysis, _ := gcanalyzer.Analyze(metrics)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		healthCheck := gcanalyzer.GenerateHealthCheck(analysis)
		if healthCheck == nil {
			b.Fatal("Expected health check")
		}
	}
}

// =============================================================================
// Real-World Scenario Benchmarks
// =============================================================================

// BenchmarkRealWorldScenario simulates a real-world monitoring scenario
func BenchmarkRealWorldScenario(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate collecting metrics for 1 second
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

		metrics, err := gcanalyzer.CollectForDuration(ctx, 100*time.Millisecond, 10*time.Millisecond)
		cancel()

		if err != nil {
			b.Fatal(err)
		}

		if len(metrics) == 0 {
			continue // Skip if no metrics collected
		}

		// Analyze the metrics
		analysis, err := gcanalyzer.Analyze(metrics)
		if err != nil {
			b.Fatal(err)
		}

		// Generate a report
		var buf strings.Builder
		err = gcanalyzer.GenerateSummaryReport(analysis, &buf)
		if err != nil {
			b.Fatal(err)
		}

		// Generate health check
		healthCheck := gcanalyzer.GenerateHealthCheck(analysis)
		if healthCheck == nil {
			b.Fatal("Expected health check")
		}
	}
}

// BenchmarkHighFrequencyCollection simulates high-frequency metrics collection
func BenchmarkHighFrequencyCollection(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Collect 10 metrics rapidly
		for j := 0; j < 10; j++ {
			metrics := types.NewGCMetricsLite()
			if metrics == nil {
				b.Fatal("Expected metrics")
			}
		}
	}
}

// BenchmarkConcurrentCollection measures concurrent collection performance
func BenchmarkConcurrentCollection(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			metrics := gcanalyzer.CollectOnce()
			if metrics == nil {
				b.Fatal("Expected metrics")
			}
		}
	})
}

// =============================================================================
// Memory Benchmarks
// =============================================================================

// BenchmarkMemoryUsage measures memory usage of the analyzer
func BenchmarkMemoryUsage(b *testing.B) {
	// Generate a large dataset
	metrics := generateTestMetrics(1000)
	events := generateTestEvents(500)

	b.ResetTimer()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for i := 0; i < b.N; i++ {
		analysis, err := gcanalyzer.AnalyzeWithEvents(metrics, events)
		if err != nil {
			b.Fatal(err)
		}

		var buf strings.Builder
		gcanalyzer.GenerateTextReport(analysis, metrics, events, &buf)

		// Force cleanup
		runtime.KeepAlive(analysis)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.StopTimer()
	b.Logf("Memory used: %d bytes", m2.TotalAlloc-m1.TotalAlloc)
	b.Logf("Allocations: %d", m2.Mallocs-m1.Mallocs)
}

// BenchmarkMemoryAllocationPattern measures allocation patterns under load
func BenchmarkMemoryAllocationPattern(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate typical usage pattern
		metrics := generateTestMetrics(100)

		analysis, err := gcanalyzer.Analyze(metrics)
		if err != nil {
			b.Fatal(err)
		}

		_ = gcanalyzer.GenerateHealthCheck(analysis)
	}
}

// =============================================================================
// Scale Benchmarks
// =============================================================================

// BenchmarkScaleAnalysis measures how analysis scales with data size
func BenchmarkScaleAnalysis(b *testing.B) {
	sizes := []int{10, 50, 100, 500, 1000}

	for _, size := range sizes {
		b.Run(strings.Replace(string(rune(size)), "", "", -1), func(b *testing.B) {
			metrics := generateTestMetrics(size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := gcanalyzer.Analyze(metrics)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// generateTestMetrics creates test metrics data
func generateTestMetrics(count int) []*gcanalyzer.GCMetrics {
	metrics := make([]*gcanalyzer.GCMetrics, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		metrics[i] = &gcanalyzer.GCMetrics{
			NumGC:         uint32(i * 10),
			PauseTotalNs:  uint64(i * 100000),
			HeapAlloc:     uint64(1024*1024 + i*1024), // Growing heap
			TotalAlloc:    uint64(i * 5 * 1024 * 1024),
			Sys:           uint64(2*1024*1024 + i*2048),
			Mallocs:       uint64(i * 1000),
			Frees:         uint64(i * 900),
			HeapSys:       uint64(2*1024*1024 + i*1024),
			HeapInuse:     uint64(1024*1024 + i*512),
			HeapObjects:   uint64(100 + i*10),
			GCCPUFraction: float64(i) * 0.001,
			Timestamp:     now.Add(time.Duration(i) * time.Second),
		}

		// Add pause data
		metrics[i].PauseNs = make([]uint64, 256)
		metrics[i].PauseEnd = make([]uint64, 256)
		for j := 0; j < 10; j++ { // Add some pause data
			metrics[i].PauseNs[j] = uint64(100000 + j*10000) // 100μs + j*10μs
			metrics[i].PauseEnd[j] = uint64(now.Add(time.Duration(i)*time.Second + time.Duration(j)*time.Millisecond).UnixNano())
		}
	}

	return metrics
}

// generateTestEvents creates test GC events
func generateTestEvents(count int) []*gcanalyzer.GCEvent {
	events := make([]*gcanalyzer.GCEvent, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		duration := time.Duration(100000+i*10000) * time.Nanosecond // Varying pause times
		startTime := now.Add(time.Duration(i) * time.Second)

		events[i] = &gcanalyzer.GCEvent{
			Sequence:      uint32(i + 1),
			StartTime:     startTime,
			EndTime:       startTime.Add(duration),
			Duration:      duration,
			HeapBefore:    uint64(2*1024*1024 + i*1024),
			HeapAfter:     uint64(1024*1024 + i*512),
			HeapReleased:  uint64(1024*1024 + i*512),
			TriggerReason: []string{"automatic", "heap_size", "forced", "periodic"}[i%4],
		}
	}

	return events
}
