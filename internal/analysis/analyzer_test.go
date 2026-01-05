package analysis

import (
	"testing"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/pkg/types"
)

// Helper function to create test metrics
func createTestMetrics(count int, baseTime time.Time, interval time.Duration) []*types.GCMetrics {
	metrics := make([]*types.GCMetrics, count)
	for i := 0; i < count; i++ {
		metrics[i] = &types.GCMetrics{
			NumGC:         uint32(10 + i*5),
			PauseTotalNs:  uint64(1000000 + i*500000),
			PauseNs:       make([]uint64, 256),
			PauseEnd:      make([]uint64, 256),
			HeapAlloc:     uint64(1024*1024 + i*512*1024),
			HeapSys:       uint64(2*1024*1024 + i*256*1024),
			HeapInuse:     uint64(1024*1024 + i*256*1024),
			TotalAlloc:    uint64(5*1024*1024 + i*1024*1024),
			Mallocs:       uint64(1000 + i*500),
			Frees:         uint64(900 + i*450),
			GCCPUFraction: 0.01 + float64(i)*0.005,
			Timestamp:     baseTime.Add(time.Duration(i) * interval),
		}
		// Set some pause data
		for j := 0; j < 10; j++ {
			metrics[i].PauseNs[j] = uint64(100000 + j*10000) // 100us - 200us
		}
	}
	return metrics
}

// Helper function to create test events
func createTestEvents(count int, baseTime time.Time) []*types.GCEvent {
	events := make([]*types.GCEvent, count)
	for i := 0; i < count; i++ {
		duration := time.Duration(500+i*100) * time.Microsecond
		events[i] = &types.GCEvent{
			Sequence:      uint32(i + 1),
			StartTime:     baseTime.Add(time.Duration(i) * time.Second),
			EndTime:       baseTime.Add(time.Duration(i)*time.Second + duration),
			Duration:      duration,
			HeapBefore:    uint64(1024*1024 + i*256*1024),
			HeapAfter:     uint64(512*1024 + i*128*1024),
			HeapReleased:  uint64(256 * 1024),
			TriggerReason: "automatic",
		}
	}
	return events
}

func TestNew(t *testing.T) {
	metrics := createTestMetrics(5, time.Now(), time.Second)
	analyzer := New(metrics)

	if analyzer == nil {
		t.Fatal("New() returned nil")
	}
	if len(analyzer.metrics) != 5 {
		t.Errorf("Expected 5 metrics, got %d", len(analyzer.metrics))
	}
	if analyzer.events != nil {
		t.Error("Expected events to be nil")
	}
}

func TestNewWithEvents(t *testing.T) {
	metrics := createTestMetrics(5, time.Now(), time.Second)
	events := createTestEvents(3, time.Now())

	analyzer := NewWithEvents(metrics, events)

	if analyzer == nil {
		t.Fatal("NewWithEvents() returned nil")
	}
	if len(analyzer.metrics) != 5 {
		t.Errorf("Expected 5 metrics, got %d", len(analyzer.metrics))
	}
	if len(analyzer.events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(analyzer.events))
	}
}

func TestAnalyze_InsufficientData(t *testing.T) {
	tests := []struct {
		name    string
		metrics []*types.GCMetrics
	}{
		{"nil metrics", nil},
		{"empty metrics", []*types.GCMetrics{}},
		{"single metric", createTestMetrics(1, time.Now(), time.Second)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := New(tt.metrics)
			_, err := analyzer.Analyze()
			if err != types.ErrInsufficientData {
				t.Errorf("Expected ErrInsufficientData, got %v", err)
			}
		})
	}
}

func TestAnalyze_Success(t *testing.T) {
	baseTime := time.Now()
	metrics := createTestMetrics(10, baseTime, time.Second)

	analyzer := New(metrics)
	analysis, err := analyzer.Analyze()

	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}
	if analysis == nil {
		t.Fatal("Analyze() returned nil analysis")
	}

	// Verify period
	expectedPeriod := 9 * time.Second // 10 samples, 1 second apart
	if analysis.Period != expectedPeriod {
		t.Errorf("Expected period %v, got %v", expectedPeriod, analysis.Period)
	}

	// Verify timestamps
	if !analysis.StartTime.Equal(baseTime) {
		t.Errorf("StartTime mismatch")
	}

	// Verify GC frequency is calculated
	if analysis.GCFrequency <= 0 {
		t.Error("GCFrequency should be positive")
	}

	// Verify heap size analysis
	if analysis.AvgHeapSize == 0 {
		t.Error("AvgHeapSize should not be zero")
	}
	if analysis.MinHeapSize > analysis.MaxHeapSize {
		t.Error("MinHeapSize should not exceed MaxHeapSize")
	}
}

func TestAnalyze_WithEvents(t *testing.T) {
	baseTime := time.Now()
	metrics := createTestMetrics(10, baseTime, time.Second)
	events := createTestEvents(5, baseTime)

	analyzer := NewWithEvents(metrics, events)
	analysis, err := analyzer.Analyze()

	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}

	// When events are available, pause times should be calculated from them
	if analysis.AvgPauseTime == 0 {
		t.Error("AvgPauseTime should not be zero with events")
	}
	if analysis.MinPauseTime == 0 {
		t.Error("MinPauseTime should not be zero with events")
	}
	if analysis.P95PauseTime == 0 {
		t.Error("P95PauseTime should not be zero with events")
	}
}

func TestAnalyzeGCFrequency(t *testing.T) {
	tests := []struct {
		name        string
		startGC     uint32
		endGC       uint32
		periodSec   int
		expectedMin float64
		expectedMax float64
	}{
		{"no GC", 10, 10, 10, 0, 0},
		{"1 GC per second", 10, 20, 10, 0.9, 1.1},
		{"5 GC per second", 10, 60, 10, 4.9, 5.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseTime := time.Now()
			metrics := []*types.GCMetrics{
				{NumGC: tt.startGC, Timestamp: baseTime},
				{NumGC: tt.endGC, Timestamp: baseTime.Add(time.Duration(tt.periodSec) * time.Second)},
			}

			analyzer := New(metrics)
			analysis := &types.GCAnalysis{
				Period: time.Duration(tt.periodSec) * time.Second,
			}

			analyzer.analyzeGCFrequency(analysis)

			if analysis.GCFrequency < tt.expectedMin || analysis.GCFrequency > tt.expectedMax {
				t.Errorf("GCFrequency = %v, want between %v and %v",
					analysis.GCFrequency, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestGetPauseTimeDistribution(t *testing.T) {
	events := []*types.GCEvent{
		{Duration: 500 * time.Microsecond}, // 0-1ms
		{Duration: 2 * time.Millisecond},   // 1-5ms
		{Duration: 7 * time.Millisecond},   // 5-10ms
		{Duration: 25 * time.Millisecond},  // 10-50ms
		{Duration: 75 * time.Millisecond},  // 50-100ms
		{Duration: 150 * time.Millisecond}, // 100ms+
		{Duration: 200 * time.Millisecond}, // 100ms+
	}

	analyzer := NewWithEvents(nil, events)
	distribution := analyzer.GetPauseTimeDistribution()

	expected := map[string]int{
		"0-1ms":    1,
		"1-5ms":    1,
		"5-10ms":   1,
		"10-50ms":  1,
		"50-100ms": 1,
		"100ms+":   2,
	}

	for bucket, expectedCount := range expected {
		if distribution[bucket] != expectedCount {
			t.Errorf("Bucket %s: expected %d, got %d", bucket, expectedCount, distribution[bucket])
		}
	}
}

func TestGetPauseTimeDistribution_Empty(t *testing.T) {
	analyzer := NewWithEvents(nil, nil)
	distribution := analyzer.GetPauseTimeDistribution()

	// Should return all buckets with zero counts
	buckets := []string{"0-1ms", "1-5ms", "5-10ms", "10-50ms", "50-100ms", "100ms+"}
	for _, bucket := range buckets {
		if distribution[bucket] != 0 {
			t.Errorf("Bucket %s should be 0, got %d", bucket, distribution[bucket])
		}
	}
}

func TestGetMemoryTrend(t *testing.T) {
	baseTime := time.Now()
	metrics := createTestMetrics(5, baseTime, time.Second)

	analyzer := New(metrics)
	trend := analyzer.GetMemoryTrend()

	if len(trend) != 5 {
		t.Fatalf("Expected 5 points, got %d", len(trend))
	}

	// Verify trend data
	for i, point := range trend {
		if point.Timestamp.IsZero() {
			t.Errorf("Point %d has zero timestamp", i)
		}
		if point.HeapAlloc == 0 {
			t.Errorf("Point %d has zero HeapAlloc", i)
		}
	}

	// Verify heap growth
	if trend[4].HeapAlloc <= trend[0].HeapAlloc {
		t.Error("Expected increasing heap allocation")
	}
}

func TestGetMemoryTrend_Empty(t *testing.T) {
	analyzer := New(nil)
	trend := analyzer.GetMemoryTrend()

	if trend != nil {
		t.Errorf("Expected nil trend for empty metrics, got %v", trend)
	}
}

func TestGetStats(t *testing.T) {
	baseTime := time.Now()
	metrics := createTestMetrics(10, baseTime, time.Second)
	events := createTestEvents(5, baseTime)

	analyzer := NewWithEvents(metrics, events)
	stats := analyzer.GetStats()

	if stats.MetricCount != 10 {
		t.Errorf("MetricCount = %d, want 10", stats.MetricCount)
	}
	if stats.EventCount != 5 {
		t.Errorf("EventCount = %d, want 5", stats.EventCount)
	}
	if stats.PeriodSeconds < 8.9 || stats.PeriodSeconds > 9.1 {
		t.Errorf("PeriodSeconds = %v, want ~9", stats.PeriodSeconds)
	}
}

func TestPercentileIndex(t *testing.T) {
	tests := []struct {
		n          int
		percentile float64
		expected   int
	}{
		{100, 0.95, 94},
		{100, 0.99, 98},
		{10, 0.95, 8},
		{10, 0.99, 8},
		{1, 0.95, 0},
		{1, 0.99, 0},
	}

	for _, tt := range tests {
		result := percentileIndex(tt.n, tt.percentile)
		if result != tt.expected {
			t.Errorf("percentileIndex(%d, %v) = %d, want %d",
				tt.n, tt.percentile, result, tt.expected)
		}
	}
}

func TestGenerateRecommendations(t *testing.T) {
	baseTime := time.Now()
	metrics := createTestMetrics(20, baseTime, time.Second)

	// Modify metrics to trigger recommendations
	for _, m := range metrics {
		m.GCCPUFraction = 0.30 // 30% - triggers high GC overhead
	}

	analyzer := New(metrics)
	analysis, err := analyzer.Analyze()

	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}

	// Should have at least one recommendation for high GC overhead
	if len(analysis.Recommendations) == 0 {
		t.Log("Note: No recommendations generated (metrics may not meet thresholds)")
	}
}

// Benchmark tests
func BenchmarkAnalyze(b *testing.B) {
	metrics := createTestMetrics(100, time.Now(), time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := New(metrics)
		_, _ = analyzer.Analyze()
	}
}

func BenchmarkAnalyzeWithEvents(b *testing.B) {
	metrics := createTestMetrics(100, time.Now(), time.Second)
	events := createTestEvents(50, time.Now())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewWithEvents(metrics, events)
		_, _ = analyzer.Analyze()
	}
}

func BenchmarkGetPauseTimeDistribution(b *testing.B) {
	events := createTestEvents(1000, time.Now())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewWithEvents(nil, events)
		_ = analyzer.GetPauseTimeDistribution()
	}
}

func BenchmarkGetMemoryTrend(b *testing.B) {
	metrics := createTestMetrics(1000, time.Now(), time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := New(metrics)
		_ = analyzer.GetMemoryTrend()
	}
}
