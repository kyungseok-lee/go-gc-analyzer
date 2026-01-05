package reporting

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/pkg/types"
)

// Helper function to create test analysis
func createTestAnalysis() *types.GCAnalysis {
	return &types.GCAnalysis{
		Period:           10 * time.Second,
		StartTime:        time.Now().Add(-10 * time.Second),
		EndTime:          time.Now(),
		GCFrequency:      2.5,
		AvgGCInterval:    400 * time.Millisecond,
		AvgPauseTime:     500 * time.Microsecond,
		MinPauseTime:     100 * time.Microsecond,
		MaxPauseTime:     2 * time.Millisecond,
		P95PauseTime:     1 * time.Millisecond,
		P99PauseTime:     1500 * time.Microsecond,
		AvgHeapSize:      10 * 1024 * 1024,
		MinHeapSize:      5 * 1024 * 1024,
		MaxHeapSize:      15 * 1024 * 1024,
		HeapGrowthRate:   1024 * 1024,
		AllocRate:        5 * 1024 * 1024,
		AllocCount:       10000,
		FreeCount:        9500,
		GCOverhead:       2.5,
		MemoryEfficiency: 75.0,
		Recommendations:  []string{"Consider increasing GOGC", "Review allocation patterns"},
	}
}

// Helper function to create test metrics
func createTestMetrics(count int) []*types.GCMetrics {
	metrics := make([]*types.GCMetrics, count)
	baseTime := time.Now()

	for i := 0; i < count; i++ {
		metrics[i] = &types.GCMetrics{
			NumGC:         uint32(10 + i),
			PauseTotalNs:  uint64(1000000 * (i + 1)),
			HeapAlloc:     uint64(1024*1024 + i*100*1024),
			HeapSys:       uint64(2 * 1024 * 1024),
			HeapInuse:     uint64(1024*1024 + i*50*1024),
			HeapObjects:   uint64(1000 + i*100),
			TotalAlloc:    uint64(5*1024*1024 + i*500*1024),
			Sys:           uint64(10 * 1024 * 1024),
			GCCPUFraction: 0.02,
			Timestamp:     baseTime.Add(time.Duration(i) * time.Second),
		}
	}
	return metrics
}

// Helper function to create test events
func createTestEvents(count int) []*types.GCEvent {
	events := make([]*types.GCEvent, count)
	baseTime := time.Now()

	for i := 0; i < count; i++ {
		events[i] = &types.GCEvent{
			Sequence:      uint32(i + 1),
			StartTime:     baseTime.Add(time.Duration(i) * time.Second),
			EndTime:       baseTime.Add(time.Duration(i)*time.Second + 500*time.Microsecond),
			Duration:      500 * time.Microsecond,
			HeapBefore:    uint64(2 * 1024 * 1024),
			HeapAfter:     uint64(1 * 1024 * 1024),
			HeapReleased:  uint64(512 * 1024),
			TriggerReason: "automatic",
		}
	}
	return events
}

func TestNew(t *testing.T) {
	analysis := createTestAnalysis()
	metrics := createTestMetrics(5)
	events := createTestEvents(3)

	reporter := New(analysis, metrics, events)

	if reporter == nil {
		t.Fatal("New() returned nil")
	}
	if reporter.analysis != analysis {
		t.Error("Analysis not set correctly")
	}
}

func TestGenerateTextReport(t *testing.T) {
	analysis := createTestAnalysis()
	reporter := New(analysis, nil, nil)

	var buf bytes.Buffer
	err := reporter.GenerateTextReport(&buf)

	if err != nil {
		t.Fatalf("GenerateTextReport() error: %v", err)
	}

	output := buf.String()

	// Check for expected sections
	expectedSections := []string{
		"Go GC Analysis Report",
		"Analysis Period",
		"GC Frequency",
		"GC Pause Times",
		"Memory Usage",
		"Allocation Statistics",
		"Efficiency Metrics",
		"Recommendations",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Text report should contain '%s'", section)
		}
	}
}

func TestGenerateTextReport_NilAnalysis(t *testing.T) {
	reporter := New(nil, nil, nil)

	var buf bytes.Buffer
	err := reporter.GenerateTextReport(&buf)

	if err != ErrNoAnalysisData {
		t.Errorf("Expected ErrNoAnalysisData, got %v", err)
	}
}

func TestGenerateJSONReport(t *testing.T) {
	analysis := createTestAnalysis()
	metrics := createTestMetrics(3)
	events := createTestEvents(2)

	reporter := New(analysis, metrics, events)

	var buf bytes.Buffer
	err := reporter.GenerateJSONReport(&buf, true)

	if err != nil {
		t.Fatalf("GenerateJSONReport() error: %v", err)
	}

	// Verify valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("Invalid JSON output: %v", err)
	}

	// Check for expected fields
	if _, ok := result["analysis"]; !ok {
		t.Error("JSON should contain 'analysis' field")
	}
	if _, ok := result["metrics"]; !ok {
		t.Error("JSON should contain 'metrics' field")
	}
	if _, ok := result["events"]; !ok {
		t.Error("JSON should contain 'events' field")
	}
}

func TestGenerateJSONReport_NoIndent(t *testing.T) {
	analysis := createTestAnalysis()
	reporter := New(analysis, nil, nil)

	var buf bytes.Buffer
	err := reporter.GenerateJSONReport(&buf, false)

	if err != nil {
		t.Fatalf("GenerateJSONReport() error: %v", err)
	}

	// Without indent, should be single line (except for trailing newline)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("Non-indented JSON should be single line, got %d lines", len(lines))
	}
}

func TestGenerateJSONReportWithOptions(t *testing.T) {
	analysis := createTestAnalysis()
	metrics := createTestMetrics(3)
	events := createTestEvents(2)

	reporter := New(analysis, metrics, events)

	tests := []struct {
		name          string
		opts          JSONReportOptions
		expectMetrics bool
		expectEvents  bool
	}{
		{
			name: "all included",
			opts: JSONReportOptions{
				Indent:         true,
				IncludeMetrics: true,
				IncludeEvents:  true,
			},
			expectMetrics: true,
			expectEvents:  true,
		},
		{
			name: "no metrics",
			opts: JSONReportOptions{
				IncludeMetrics: false,
				IncludeEvents:  true,
			},
			expectMetrics: false,
			expectEvents:  true,
		},
		{
			name: "no events",
			opts: JSONReportOptions{
				IncludeMetrics: true,
				IncludeEvents:  false,
			},
			expectMetrics: true,
			expectEvents:  false,
		},
		{
			name: "compact mode",
			opts: JSONReportOptions{
				IncludeMetrics:   true,
				CompactPauseData: true,
			},
			expectMetrics: true,
			expectEvents:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := reporter.GenerateJSONReportWithOptions(&buf, tt.opts)

			if err != nil {
				t.Fatalf("GenerateJSONReportWithOptions() error: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			hasMetrics := result["metrics"] != nil
			hasEvents := result["events"] != nil

			if hasMetrics != tt.expectMetrics {
				t.Errorf("metrics presence = %v, want %v", hasMetrics, tt.expectMetrics)
			}
			if hasEvents != tt.expectEvents {
				t.Errorf("events presence = %v, want %v", hasEvents, tt.expectEvents)
			}
		})
	}
}

func TestGenerateCompactJSONReport(t *testing.T) {
	analysis := createTestAnalysis()
	metrics := createTestMetrics(3)
	reporter := New(analysis, metrics, nil)

	var buf bytes.Buffer
	err := reporter.GenerateCompactJSONReport(&buf)

	if err != nil {
		t.Fatalf("GenerateCompactJSONReport() error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Compact should not include metrics or events
	if result["metrics"] != nil {
		t.Error("Compact report should not include metrics")
	}
}

func TestGenerateTableReport(t *testing.T) {
	metrics := createTestMetrics(5)
	reporter := New(nil, metrics, nil)

	var buf bytes.Buffer
	err := reporter.GenerateTableReport(&buf)

	if err != nil {
		t.Fatalf("GenerateTableReport() error: %v", err)
	}

	output := buf.String()

	// Check for header
	if !strings.Contains(output, "Timestamp") {
		t.Error("Table should contain Timestamp header")
	}
	if !strings.Contains(output, "Heap") {
		t.Error("Table should contain Heap header")
	}
}

func TestGenerateTableReport_NoMetrics(t *testing.T) {
	reporter := New(nil, nil, nil)

	var buf bytes.Buffer
	err := reporter.GenerateTableReport(&buf)

	if err != ErrNoMetricsData {
		t.Errorf("Expected ErrNoMetricsData, got %v", err)
	}
}

func TestGenerateSummaryReport(t *testing.T) {
	analysis := createTestAnalysis()
	reporter := New(analysis, nil, nil)

	var buf bytes.Buffer
	err := reporter.GenerateSummaryReport(&buf)

	if err != nil {
		t.Fatalf("GenerateSummaryReport() error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "GC Summary Report") {
		t.Error("Summary should contain title")
	}
	if !strings.Contains(output, "Period") {
		t.Error("Summary should contain period")
	}
}

func TestGenerateSummaryReport_NilAnalysis(t *testing.T) {
	reporter := New(nil, nil, nil)

	var buf bytes.Buffer
	err := reporter.GenerateSummaryReport(&buf)

	if err != ErrNoAnalysisData {
		t.Errorf("Expected ErrNoAnalysisData, got %v", err)
	}
}

func TestGenerateEventsReport(t *testing.T) {
	events := createTestEvents(5)
	reporter := New(nil, nil, events)

	var buf bytes.Buffer
	err := reporter.GenerateEventsReport(&buf)

	if err != nil {
		t.Fatalf("GenerateEventsReport() error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "GC Events Report") {
		t.Error("Events report should contain title")
	}
	if !strings.Contains(output, "Duration") {
		t.Error("Events report should contain Duration column")
	}
}

func TestGenerateEventsReport_NoEvents(t *testing.T) {
	reporter := New(nil, nil, nil)

	var buf bytes.Buffer
	err := reporter.GenerateEventsReport(&buf)

	if err != ErrNoEventsData {
		t.Errorf("Expected ErrNoEventsData, got %v", err)
	}
}

func TestGenerateGrafanaMetrics(t *testing.T) {
	analysis := createTestAnalysis()
	reporter := New(analysis, nil, nil)

	var buf bytes.Buffer
	err := reporter.GenerateGrafanaMetrics(&buf)

	if err != nil {
		t.Fatalf("GenerateGrafanaMetrics() error: %v", err)
	}

	output := buf.String()

	// Check Prometheus format
	expectedMetrics := []string{
		"gc_frequency_total",
		"gc_pause_time_avg_seconds",
		"gc_pause_time_p99_seconds",
		"heap_size_avg_bytes",
		"allocation_rate_bytes_per_second",
		"gc_overhead_percent",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(output, metric) {
			t.Errorf("Prometheus output should contain '%s'", metric)
		}
	}

	// Check for HELP and TYPE annotations
	if !strings.Contains(output, "# HELP") {
		t.Error("Should contain HELP annotations")
	}
	if !strings.Contains(output, "# TYPE") {
		t.Error("Should contain TYPE annotations")
	}
}

func TestGenerateGrafanaMetrics_NilAnalysis(t *testing.T) {
	reporter := New(nil, nil, nil)

	var buf bytes.Buffer
	err := reporter.GenerateGrafanaMetrics(&buf)

	if err != ErrNoAnalysisData {
		t.Errorf("Expected ErrNoAnalysisData, got %v", err)
	}
}

func TestGenerateHealthCheck(t *testing.T) {
	analysis := createTestAnalysis()
	reporter := New(analysis, nil, nil)

	health := reporter.GenerateHealthCheck()

	if health == nil {
		t.Fatal("GenerateHealthCheck() returned nil")
	}

	// Check valid status
	validStatuses := map[string]bool{"healthy": true, "warning": true, "critical": true}
	if !validStatuses[health.Status] {
		t.Errorf("Invalid status: %s", health.Status)
	}

	// Check score range
	if health.Score < 0 || health.Score > 100 {
		t.Errorf("Score should be 0-100, got %d", health.Score)
	}

	// Check timestamp
	if health.LastUpdated.IsZero() {
		t.Error("LastUpdated should not be zero")
	}
}

func TestGenerateHealthCheck_NilAnalysis(t *testing.T) {
	reporter := New(nil, nil, nil)

	health := reporter.GenerateHealthCheck()

	if health == nil {
		t.Fatal("GenerateHealthCheck() should not return nil")
	}

	if health.Status != "unknown" {
		t.Errorf("Status should be 'unknown' for nil analysis, got %s", health.Status)
	}

	if health.Score != 0 {
		t.Errorf("Score should be 0 for nil analysis, got %d", health.Score)
	}
}

func TestGenerateHealthCheck_Thresholds(t *testing.T) {
	tests := []struct {
		name           string
		analysis       *types.GCAnalysis
		expectedStatus string
		minScore       int
		maxScore       int
	}{
		{
			name: "healthy",
			analysis: &types.GCAnalysis{
				GCFrequency:      1.0,
				AvgPauseTime:     10 * time.Millisecond,
				P99PauseTime:     50 * time.Millisecond,
				GCOverhead:       5.0,
				MemoryEfficiency: 80.0,
				AllocRate:        10 * 1024 * 1024,
			},
			expectedStatus: "healthy",
			minScore:       80,
			maxScore:       100,
		},
		{
			name: "high gc frequency",
			analysis: &types.GCAnalysis{
				GCFrequency:      15.0, // Above threshold
				AvgPauseTime:     10 * time.Millisecond,
				P99PauseTime:     50 * time.Millisecond,
				GCOverhead:       5.0,
				MemoryEfficiency: 80.0,
				AllocRate:        10 * 1024 * 1024,
			},
			expectedStatus: "healthy",
			minScore:       80,
			maxScore:       90,
		},
		{
			name: "critical - multiple issues",
			analysis: &types.GCAnalysis{
				GCFrequency:      15.0,
				AvgPauseTime:     200 * time.Millisecond,
				P99PauseTime:     600 * time.Millisecond,
				GCOverhead:       30.0,
				MemoryEfficiency: 40.0,
				AllocRate:        200 * 1024 * 1024,
			},
			expectedStatus: "critical",
			minScore:       0,
			maxScore:       59,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := New(tt.analysis, nil, nil)
			health := reporter.GenerateHealthCheck()

			if health.Status != tt.expectedStatus {
				t.Errorf("Status = %s, want %s", health.Status, tt.expectedStatus)
			}

			if health.Score < tt.minScore || health.Score > tt.maxScore {
				t.Errorf("Score = %d, want between %d and %d",
					health.Score, tt.minScore, tt.maxScore)
			}
		})
	}
}

// Benchmark tests
func BenchmarkGenerateTextReport(b *testing.B) {
	analysis := createTestAnalysis()
	reporter := New(analysis, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = reporter.GenerateTextReport(&buf)
	}
}

func BenchmarkGenerateJSONReport(b *testing.B) {
	analysis := createTestAnalysis()
	metrics := createTestMetrics(100)
	events := createTestEvents(50)
	reporter := New(analysis, metrics, events)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = reporter.GenerateJSONReport(&buf, false)
	}
}

func BenchmarkGenerateHealthCheck(b *testing.B) {
	analysis := createTestAnalysis()
	reporter := New(analysis, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reporter.GenerateHealthCheck()
	}
}

func BenchmarkGenerateGrafanaMetrics(b *testing.B) {
	analysis := createTestAnalysis()
	reporter := New(analysis, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = reporter.GenerateGrafanaMetrics(&buf)
	}
}
