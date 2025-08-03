package tests

import (
	"testing"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/analyzer"
)

func TestAnalyzer_Analyze(t *testing.T) {
	// Create test metrics data
	now := time.Now()
	metrics := []*analyzer.GCMetrics{
		{
			NumGC:         10,
			PauseTotalNs:  1000000,         // 1ms total
			HeapAlloc:     1024 * 1024,     // 1MB
			TotalAlloc:    5 * 1024 * 1024, // 5MB
			Mallocs:       1000,
			Frees:         900,
			GCCPUFraction: 0.01, // 1%
			Timestamp:     now,
		},
		{
			NumGC:         15,
			PauseTotalNs:  1500000,          // 1.5ms total
			HeapAlloc:     2 * 1024 * 1024,  // 2MB
			TotalAlloc:    10 * 1024 * 1024, // 10MB
			Mallocs:       2000,
			Frees:         1800,
			GCCPUFraction: 0.015, // 1.5%
			Timestamp:     now.Add(10 * time.Second),
		},
	}

	analyzer := analyzer.NewAnalyzer(metrics)
	analysis, err := analyzer.Analyze()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if analysis == nil {
		t.Fatal("Expected analysis result, got nil")
	}

	// Test basic analysis properties
	if analysis.Period != 10*time.Second {
		t.Errorf("Expected period of 10s, got %v", analysis.Period)
	}

	// Test GC frequency calculation
	expectedFreq := float64(5) / 10.0 // 5 GCs in 10 seconds
	if analysis.GCFrequency != expectedFreq {
		t.Errorf("Expected GC frequency of %f, got %f", expectedFreq, analysis.GCFrequency)
	}

	// Test allocation rate calculation
	allocDiff := float64(5 * 1024 * 1024) // 5MB difference
	expectedAllocRate := allocDiff / 10.0 // over 10 seconds
	if analysis.AllocRate != expectedAllocRate {
		t.Errorf("Expected alloc rate of %f, got %f", expectedAllocRate, analysis.AllocRate)
	}
}

func TestAnalyzer_InsufficientData(t *testing.T) {
	// Test with insufficient data
	metrics := []*analyzer.GCMetrics{
		{
			NumGC:     10,
			Timestamp: time.Now(),
		},
	}

	analyzer := analyzer.NewAnalyzer(metrics)
	_, err := analyzer.Analyze()

	if err == nil {
		t.Error("Expected error for insufficient data, got nil")
	}
}

func TestAnalyzer_GetPauseTimeDistribution(t *testing.T) {
	events := []*analyzer.GCEvent{
		{Duration: 500 * time.Microsecond}, // 0-1ms
		{Duration: 2 * time.Millisecond},   // 1-5ms
		{Duration: 7 * time.Millisecond},   // 5-10ms
		{Duration: 25 * time.Millisecond},  // 10-50ms
		{Duration: 75 * time.Millisecond},  // 50-100ms
		{Duration: 150 * time.Millisecond}, // 100ms+
	}

	analyzer := analyzer.NewAnalyzerWithEvents(nil, events)
	distribution := analyzer.GetPauseTimeDistribution()

	expected := map[string]int{
		"0-1ms":    1,
		"1-5ms":    1,
		"5-10ms":   1,
		"10-50ms":  1,
		"50-100ms": 1,
		"100ms+":   1,
	}

	for bucket, expectedCount := range expected {
		if distribution[bucket] != expectedCount {
			t.Errorf("Expected %d events in bucket %s, got %d",
				expectedCount, bucket, distribution[bucket])
		}
	}
}

func TestAnalyzer_GetMemoryTrend(t *testing.T) {
	now := time.Now()
	metrics := []*analyzer.GCMetrics{
		{
			HeapAlloc: 1024,
			HeapSys:   2048,
			HeapInuse: 1500,
			Timestamp: now,
		},
		{
			HeapAlloc: 2048,
			HeapSys:   3072,
			HeapInuse: 2500,
			Timestamp: now.Add(time.Second),
		},
	}

	analyzer := analyzer.NewAnalyzer(metrics)
	trend := analyzer.GetMemoryTrend()

	if len(trend) != 2 {
		t.Errorf("Expected 2 memory points, got %d", len(trend))
	}

	if trend[0].HeapAlloc != 1024 {
		t.Errorf("Expected first point HeapAlloc=1024, got %d", trend[0].HeapAlloc)
	}

	if trend[1].HeapAlloc != 2048 {
		t.Errorf("Expected second point HeapAlloc=2048, got %d", trend[1].HeapAlloc)
	}
}

func TestGCMetrics_NewGCMetrics(t *testing.T) {
	metrics := analyzer.NewGCMetrics()

	if metrics == nil {
		t.Fatal("Expected metrics, got nil")
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	// Basic sanity checks
	if metrics.NumGC < 0 {
		t.Error("NumGC should not be negative")
	}
}

func TestGCMetrics_ToBytes(t *testing.T) {
	metrics := &analyzer.GCMetrics{}

	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}

	for _, test := range tests {
		result := metrics.ToBytes(test.input)
		if result != test.expected {
			t.Errorf("ToBytes(%d) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestGCMetrics_ToDuration(t *testing.T) {
	metrics := &analyzer.GCMetrics{}

	tests := []struct {
		input    uint64
		expected time.Duration
	}{
		{1000000, time.Millisecond},
		{1000000000, time.Second},
		{500000, 500 * time.Microsecond},
	}

	for _, test := range tests {
		result := metrics.ToDuration(test.input)
		if result != test.expected {
			t.Errorf("ToDuration(%d) = %v, expected %v", test.input, result, test.expected)
		}
	}
}
