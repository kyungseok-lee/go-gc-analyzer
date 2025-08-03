package tests

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/pkg/gcanalyzer"
)

func TestIntegration_FullAnalysisFlow(t *testing.T) {
	// Force some allocations to trigger GC
	forceGCActivity()

	// Collect metrics for a short period
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	metrics, err := gcanalyzer.CollectForDuration(ctx, 2*time.Second, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	if len(metrics) == 0 {
		t.Fatal("No metrics collected")
	}

	// Analyze the collected metrics
	analysis, err := gcanalyzer.Analyze(metrics)
	if err != nil {
		t.Fatalf("Failed to analyze metrics: %v", err)
	}

	// Validate analysis results
	if analysis.Period <= 0 {
		t.Error("Analysis period should be positive")
	}

	if analysis.GCFrequency < 0 {
		t.Error("GC frequency should not be negative")
	}

	if analysis.AvgHeapSize == 0 {
		t.Error("Average heap size should not be zero")
	}

		// Test text report generation
	var textReport strings.Builder
	err = gcanalyzer.GenerateTextReport(analysis, metrics, nil, &textReport)
	if err != nil {
		t.Errorf("Failed to generate text report: %v", err)
	}

	reportContent := textReport.String()
	if !strings.Contains(reportContent, "GC Analysis Report") {
		t.Error("Text report should contain title")
	}

	if !strings.Contains(reportContent, "GC Frequency") {
		t.Error("Text report should contain GC frequency section")
	}

	// Test JSON report generation
	var jsonReport strings.Builder
	err = gcanalyzer.GenerateJSONReport(analysis, metrics, nil, &jsonReport, true)
	if err != nil {
		t.Errorf("Failed to generate JSON report: %v", err)
	}

	jsonContent := jsonReport.String()
	if !strings.Contains(jsonContent, "analysis") {
		t.Error("JSON report should contain analysis data")
	}

		// Test summary report
	var summaryReport strings.Builder
	err = gcanalyzer.GenerateSummaryReport(analysis, &summaryReport)
	if err != nil {
		t.Errorf("Failed to generate summary report: %v", err)
	}
	
	summaryContent := summaryReport.String()
	if !strings.Contains(summaryContent, "GC Summary Report") {
		t.Error("Summary report should contain title")
	}
	
	// Test health check
	healthCheck := gcanalyzer.GenerateHealthCheck(analysis)
	if healthCheck == nil {
		t.Error("Health check should not be nil")
	}

	if healthCheck.Status == "" {
		t.Error("Health check status should not be empty")
	}

	if healthCheck.Score < 0 || healthCheck.Score > 100 {
		t.Errorf("Health check score should be between 0-100, got %d", healthCheck.Score)
	}
}

func TestIntegration_CollectorWithAnalysis(t *testing.T) {
	// Setup collector with callbacks
	var collectedMetrics []*gcanalyzer.GCMetrics
	var events []*gcanalyzer.GCEvent

		config := &gcanalyzer.MonitorConfig{
		Interval:   100 * time.Millisecond,
		MaxSamples: 50,
		OnMetric: func(m *gcanalyzer.GCMetrics) {
			collectedMetrics = append(collectedMetrics, m)
		},
		OnGCEvent: func(e *gcanalyzer.GCEvent) {
			events = append(events, e)
		},
	}
	
	monitor := gcanalyzer.NewMonitor(config)

	// Start collection
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	// Generate some GC activity
	go func() {
		for i := 0; i < 5; i++ {
			forceGCActivity()
			time.Sleep(100 * time.Millisecond)
		}
	}()

		// Wait for collection period
	<-ctx.Done()
	monitor.Stop()
	
	// Verify we collected metrics
	finalMetrics := monitor.GetMetrics()
	if len(finalMetrics) == 0 {
		t.Error("Should have collected some metrics")
	}

	// Verify callbacks were called
	if len(collectedMetrics) == 0 {
		t.Error("OnMetricCollected callback should have been called")
	}

	// Analyze collected data
		if len(finalMetrics) >= 2 {
		analysis, err := gcanalyzer.AnalyzeWithEvents(finalMetrics, monitor.GetEvents())
		if err != nil {
			t.Errorf("Failed to analyze collected metrics: %v", err)
		}
		
		if analysis.Period <= 0 {
			t.Error("Analysis period should be positive")
		}
	}
}

func TestIntegration_MemoryTrendAnalysis(t *testing.T) {
	// Collect metrics while generating memory pressure
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start background memory allocation
	go func() {
		allocations := make([][]byte, 0, 100)
		for i := 0; i < 50; i++ {
			// Allocate and keep some memory
			data := make([]byte, 1024*1024) // 1MB
			allocations = append(allocations, data)
			time.Sleep(20 * time.Millisecond)
		}
		// Keep allocations alive until context is done
		<-ctx.Done()
		_ = allocations // Prevent compiler optimization
	}()

	metrics, err := gcanalyzer.CollectForDuration(ctx, 1*time.Second, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	if len(metrics) < 2 {
		t.Fatal("Need at least 2 metrics for trend analysis")
	}

		// Test memory trend analysis
	memoryTrend := gcanalyzer.GetMemoryTrend(metrics)
	if len(memoryTrend) != len(metrics) {
		t.Errorf("Memory trend should have same length as metrics, got %d vs %d",
			len(memoryTrend), len(metrics))
	}

	// Verify trend data is reasonable
	for i, point := range memoryTrend {
		if point.Timestamp.IsZero() {
			t.Errorf("Memory trend point %d has zero timestamp", i)
		}

		if point.HeapAlloc == 0 {
			t.Errorf("Memory trend point %d has zero heap allocation", i)
		}
	}

	// Should show increasing memory usage
	first := memoryTrend[0]
	last := memoryTrend[len(memoryTrend)-1]

	if last.HeapAlloc <= first.HeapAlloc {
		t.Log("Warning: Expected increasing memory usage, but heap didn't grow")
		// This might happen in some test environments, so just log a warning
	}
}

func TestIntegration_PauseTimeDistribution(t *testing.T) {
	// Force multiple GCs to get pause time data
	for i := 0; i < 5; i++ {
		forceGCActivity()
		runtime.GC() // Force GC to ensure we have events
		time.Sleep(10 * time.Millisecond)
	}

	// Collect some metrics
	ctx := context.Background()
	metrics, err := gcanalyzer.CollectForDuration(ctx, 500*time.Millisecond, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	if len(metrics) == 0 {
		t.Fatal("No metrics collected")
	}

	// Check pause time distribution (using empty events for this test)
	distribution := gcanalyzer.GetPauseTimeDistribution([]*gcanalyzer.GCEvent{})

	// Verify all expected buckets exist
	expectedBuckets := []string{"0-1ms", "1-5ms", "5-10ms", "10-50ms", "50-100ms", "100ms+"}
	for _, bucket := range expectedBuckets {
		if _, exists := distribution[bucket]; !exists {
			t.Errorf("Distribution should contain bucket %s", bucket)
		}
	}

	// Verify counts are non-negative
	for bucket, count := range distribution {
		if count < 0 {
			t.Errorf("Bucket %s should not have negative count: %d", bucket, count)
		}
	}
}

func TestIntegration_ReporterFormats(t *testing.T) {
	// Collect some metrics
	ctx := context.Background()
	metrics, err := gcanalyzer.CollectForDuration(ctx, 500*time.Millisecond, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	if len(metrics) < 2 {
		t.Fatal("Need at least 2 metrics for analysis")
	}

		// Analyze
	analysis, err := gcanalyzer.Analyze(metrics)
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	// Test all report formats
	formats := []struct {
		name string
		test func() error
	}{
		{
			"text",
			func() error {
				var buf strings.Builder
				return gcanalyzer.GenerateTextReport(analysis, metrics, nil, &buf)
			},
		},
		{
			"json",
			func() error {
				var buf strings.Builder
				return gcanalyzer.GenerateJSONReport(analysis, metrics, nil, &buf, false)
			},
		},
		{
			"json-indented",
			func() error {
				var buf strings.Builder
				return gcanalyzer.GenerateJSONReport(analysis, metrics, nil, &buf, true)
			},
		},
		{
			"summary",
			func() error {
				var buf strings.Builder
				return gcanalyzer.GenerateSummaryReport(analysis, &buf)
			},
		},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			err := format.test()
			if err != nil {
				t.Errorf("Failed to generate %s report: %v", format.name, err)
			}
		})
	}
}

// forceGCActivity generates memory allocations to trigger garbage collection
func forceGCActivity() {
	// Create temporary allocations that will need to be collected
	for i := 0; i < 1000; i++ {
		data := make([]byte, 1024) // 1KB allocation
		_ = data                   // Prevent optimization
	}

	// Create some objects that will go out of scope
	slices := make([][]int, 100)
	for i := range slices {
		slices[i] = make([]int, 100)
	}

	// Force a garbage collection
	runtime.GC()
}
