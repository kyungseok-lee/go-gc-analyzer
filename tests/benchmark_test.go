package tests

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/pkg/gcanalyzer"
)

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
		metrics := gcanalyzer.NewGCMetrics()
		if metrics == nil {
			b.Fatal("Expected metrics")
		}
	}
}

// BenchmarkAnalyzer_Analyze_Benchmark measures the performance of GC analysis
func BenchmarkAnalyzer_Analyze_Benchmark(b *testing.B) {
	// Create test dataset
	metrics := generateTestMetrics(100)
	events := generateTestEvents(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gcAnalyzer := gcanalyzer.NewAnalyzerWithEvents(metrics, events)
		analysis, err := gcAnalyzer.Analyze()
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
		gcAnalyzer := gcanalyzer.NewAnalyzer(metrics)
		analysis, err := gcAnalyzer.Analyze()
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
		gcAnalyzer := gcanalyzer.NewAnalyzerWithEvents(metrics, events)
		analysis, err := gcAnalyzer.Analyze()
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
	gcAnalyzer := gcanalyzer.NewAnalyzer(metrics)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trend := gcAnalyzer.GetMemoryTrend()
		if len(trend) == 0 {
			b.Fatal("Expected memory trend data")
		}
	}
}

// BenchmarkAnalyzer_GetPauseTimeDistribution measures pause time distribution calculation
func BenchmarkAnalyzer_GetPauseTimeDistribution(b *testing.B) {
	events := generateTestEvents(100)
	gcAnalyzer := gcanalyzer.NewAnalyzerWithEvents(nil, events)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		distribution := gcAnalyzer.GetPauseTimeDistribution()
		if len(distribution) == 0 {
			b.Fatal("Expected distribution data")
		}
	}
}

// BenchmarkReporter_GenerateTextReport measures text report generation performance
func BenchmarkReporter_GenerateTextReport(b *testing.B) {
	metrics := generateTestMetrics(50)
	events := generateTestEvents(25)

	gcAnalyzer := gcanalyzer.NewAnalyzerWithEvents(metrics, events)
	analysis, _ := gcAnalyzer.Analyze()

	reporter := gcanalyzer.NewReporter(analysis, metrics, events)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		err := reporter.GenerateTextReport(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReporter_GenerateJSONReport measures JSON report generation performance
func BenchmarkReporter_GenerateJSONReport(b *testing.B) {
	metrics := generateTestMetrics(50)
	events := generateTestEvents(25)

	gcAnalyzer := gcanalyzer.NewAnalyzerWithEvents(metrics, events)
	analysis, _ := gcAnalyzer.Analyze()

	reporter := gcanalyzer.NewReporter(analysis, metrics, events)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		err := reporter.GenerateJSONReport(&buf, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReporter_GenerateTableReport measures table report generation performance
func BenchmarkReporter_GenerateTableReport(b *testing.B) {
	metrics := generateTestMetrics(50)

	gcAnalyzer := gcanalyzer.NewAnalyzer(metrics)
	analysis, _ := gcAnalyzer.Analyze()

	reporter := gcanalyzer.NewReporter(analysis, metrics, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		err := reporter.GenerateTableReport(&buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReporter_GenerateHealthCheck measures health check generation performance
func BenchmarkReporter_GenerateHealthCheck(b *testing.B) {
	metrics := generateTestMetrics(50)

	gcAnalyzer := gcanalyzer.NewAnalyzer(metrics)
	analysis, _ := gcAnalyzer.Analyze()

	reporter := gcanalyzer.NewReporter(analysis, metrics, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		healthCheck := reporter.GenerateHealthCheck()
		if healthCheck == nil {
			b.Fatal("Expected health check")
		}
	}
}

// BenchmarkCollector_StartStop measures collector start/stop overhead
func BenchmarkCollector_StartStop(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector := gcanalyzer.NewCollector(&gcanalyzer.CollectorConfig{
			Interval: time.Hour, // Very long interval to avoid actual collection
		})

		ctx := context.Background()
		err := collector.Start(ctx)
		if err != nil {
			b.Fatal(err)
		}

		collector.Stop()
	}
}

// BenchmarkCollector_Collection measures the overhead of the collection process
func BenchmarkCollector_Collection(b *testing.B) {
	var collectedCount int

	config := &gcanalyzer.CollectorConfig{
		Interval:   time.Millisecond, // Very fast collection
		MaxSamples: 1000,
		OnMetricCollected: func(m *gcanalyzer.GCMetrics) {
			collectedCount++
		},
	}

	collector := gcanalyzer.NewCollector(config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	b.ResetTimer()

	err := collector.Start(ctx)
	if err != nil {
		b.Fatal(err)
	}

	<-ctx.Done()
	collector.Stop()

	b.StopTimer()
	b.Logf("Collected %d metrics during benchmark", collectedCount)
}

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
		gcAnalyzer := gcanalyzer.NewAnalyzer(metrics)
		analysis, err := gcAnalyzer.Analyze()
		if err != nil {
			b.Fatal(err)
		}

		// Generate a report
		reporter := gcanalyzer.NewReporter(analysis, metrics, nil)
		var buf strings.Builder
		err = reporter.GenerateSummaryReport(&buf)
		if err != nil {
			b.Fatal(err)
		}

		// Generate health check
		healthCheck := reporter.GenerateHealthCheck()
		if healthCheck == nil {
			b.Fatal("Expected health check")
		}
	}
}

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
		gcAnalyzer := gcanalyzer.NewAnalyzerWithEvents(metrics, events)
		analysis, err := gcAnalyzer.Analyze()
		if err != nil {
			b.Fatal(err)
		}

		reporter := gcanalyzer.NewReporter(analysis, metrics, events)
		var buf strings.Builder
		reporter.GenerateTextReport(&buf)

		// Force cleanup
		runtime.KeepAlive(gcAnalyzer)
		runtime.KeepAlive(reporter)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.StopTimer()
	b.Logf("Memory used: %d bytes", m2.TotalAlloc-m1.TotalAlloc)
	b.Logf("Allocations: %d", m2.Mallocs-m1.Mallocs)
}

// Helper function to generate test metrics
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

// Helper function to generate test events
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
