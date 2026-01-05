package types

import (
	"testing"
	"time"
)

func TestNewGCMetrics(t *testing.T) {
	metrics := NewGCMetrics()

	if metrics == nil {
		t.Fatal("NewGCMetrics() returned nil")
	}

	// Check timestamp is set
	if metrics.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	// Check PauseNs is allocated (standard metrics include pause data)
	if metrics.PauseNs == nil {
		t.Error("PauseNs should not be nil for standard metrics")
	}

	// Check basic sanity
	if metrics.HeapSys == 0 {
		t.Error("HeapSys should not be zero")
	}

	// Should not be pooled
	if metrics.pooled {
		t.Error("Standard metrics should not be pooled")
	}
}

func TestNewGCMetricsPooled(t *testing.T) {
	metrics := NewGCMetricsPooled()

	if metrics == nil {
		t.Fatal("NewGCMetricsPooled() returned nil")
	}

	// Check pause data is present
	if metrics.PauseNs == nil {
		t.Error("PauseNs should not be nil for pooled metrics")
	}

	// Should be marked as pooled
	if !metrics.pooled {
		t.Error("Pooled metrics should have pooled=true")
	}

	// Release should work without panic
	metrics.Release()

	// After release, slices should be nil
	if metrics.PauseNs != nil {
		t.Error("PauseNs should be nil after Release()")
	}
	if metrics.PauseEnd != nil {
		t.Error("PauseEnd should be nil after Release()")
	}

	// Pooled flag should be false
	if metrics.pooled {
		t.Error("pooled should be false after Release()")
	}
}

func TestNewGCMetricsLite(t *testing.T) {
	metrics := NewGCMetricsLite()

	if metrics == nil {
		t.Fatal("NewGCMetricsLite() returned nil")
	}

	// Lite metrics should NOT have pause data
	if metrics.PauseNs != nil {
		t.Error("Lite metrics should have nil PauseNs")
	}
	if metrics.PauseEnd != nil {
		t.Error("Lite metrics should have nil PauseEnd")
	}

	// Should not be pooled
	if metrics.pooled {
		t.Error("Lite metrics should not be pooled")
	}

	// Other fields should still be populated
	if metrics.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestGCMetrics_Release_NotPooled(t *testing.T) {
	// Standard metrics (not pooled)
	metrics := NewGCMetrics()

	// Release should be no-op for non-pooled metrics
	metrics.Release()

	// Slices should still be present
	if metrics.PauseNs == nil {
		t.Error("PauseNs should not be nil for non-pooled metrics after Release()")
	}
}

func TestGCMetrics_Release_Multiple(t *testing.T) {
	metrics := NewGCMetricsPooled()

	// First release
	metrics.Release()

	// Second release should not panic
	metrics.Release()
}

func TestGCMetrics_Clone(t *testing.T) {
	original := NewGCMetrics()
	original.NumGC = 42
	original.HeapAlloc = 1024 * 1024

	clone := original.Clone()

	if clone == nil {
		t.Fatal("Clone() returned nil")
	}

	// Check values are copied
	if clone.NumGC != original.NumGC {
		t.Error("NumGC not cloned correctly")
	}
	if clone.HeapAlloc != original.HeapAlloc {
		t.Error("HeapAlloc not cloned correctly")
	}

	// Check slices are deep copied
	if clone.PauseNs == nil {
		t.Error("PauseNs should be cloned")
	}

	// Modify clone should not affect original
	clone.NumGC = 100
	if original.NumGC == 100 {
		t.Error("Clone should be independent from original")
	}

	// Clone should not be pooled
	if clone.pooled {
		t.Error("Clone should not be pooled")
	}
}

func TestGCMetrics_Clone_Nil(t *testing.T) {
	var metrics *GCMetrics
	clone := metrics.Clone()

	if clone != nil {
		t.Error("Clone of nil should be nil")
	}
}

func TestGCMetrics_Clone_LiteMetrics(t *testing.T) {
	original := NewGCMetricsLite()
	clone := original.Clone()

	if clone == nil {
		t.Fatal("Clone() returned nil")
	}

	// Lite metrics have nil slices, clone should too
	if clone.PauseNs != nil {
		t.Error("Clone of lite metrics should have nil PauseNs")
	}
}

func TestGCMetrics_ToBytes(t *testing.T) {
	metrics := &GCMetrics{}

	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024, "1.5 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}

	for _, tt := range tests {
		result := metrics.ToBytes(tt.input)
		if result != tt.expected {
			t.Errorf("ToBytes(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestGCMetrics_ToDuration(t *testing.T) {
	metrics := &GCMetrics{}

	tests := []struct {
		input    uint64
		expected time.Duration
	}{
		{0, 0},
		{1000, time.Microsecond},
		{1000000, time.Millisecond},
		{1000000000, time.Second},
		{500000, 500 * time.Microsecond},
	}

	for _, tt := range tests {
		result := metrics.ToDuration(tt.input)
		if result != tt.expected {
			t.Errorf("ToDuration(%d) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KB"},
		{2048, "2.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{1125899906842624, "1.0 PB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatBytesRate(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "0 B/s"},
		{-1, "0 B/s"},
		{1024, "1.0 KB/s"},
		{1048576, "1.0 MB/s"},
	}

	for _, tt := range tests {
		result := FormatBytesRate(tt.input)
		if result != tt.expected {
			t.Errorf("FormatBytesRate(%v) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestHealthCheckStatus(t *testing.T) {
	status := &HealthCheckStatus{
		Status:      "healthy",
		Score:       85,
		Issues:      []string{"Minor issue"},
		Summary:     "Test summary",
		LastUpdated: time.Now(),
	}

	if status.Status != "healthy" {
		t.Error("Status not set correctly")
	}
	if status.Score != 85 {
		t.Error("Score not set correctly")
	}
	if len(status.Issues) != 1 {
		t.Error("Issues not set correctly")
	}
}

func TestGCEvent(t *testing.T) {
	event := &GCEvent{
		Sequence:      1,
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(time.Millisecond),
		Duration:      time.Millisecond,
		HeapBefore:    1024 * 1024,
		HeapAfter:     512 * 1024,
		HeapReleased:  256 * 1024,
		TriggerReason: "automatic",
	}

	if event.Sequence != 1 {
		t.Error("Sequence not set correctly")
	}
	if event.Duration != time.Millisecond {
		t.Error("Duration not set correctly")
	}
}

func TestMemoryPoint(t *testing.T) {
	point := MemoryPoint{
		Timestamp: time.Now(),
		HeapAlloc: 1024,
		HeapSys:   2048,
		HeapInuse: 1500,
	}

	if point.HeapAlloc != 1024 {
		t.Error("HeapAlloc not set correctly")
	}
}

func TestGCAnalysis(t *testing.T) {
	analysis := &GCAnalysis{
		Period:          time.Minute,
		StartTime:       time.Now().Add(-time.Minute),
		EndTime:         time.Now(),
		GCFrequency:     2.5,
		AvgGCInterval:   24 * time.Second,
		AvgPauseTime:    500 * time.Microsecond,
		Recommendations: []string{"Test recommendation"},
	}

	if analysis.Period != time.Minute {
		t.Error("Period not set correctly")
	}
	if len(analysis.Recommendations) != 1 {
		t.Error("Recommendations not set correctly")
	}
}

func TestConstants(t *testing.T) {
	// Verify constants are defined with expected values
	if ThresholdGCFrequencyHigh <= 0 {
		t.Error("ThresholdGCFrequencyHigh should be positive")
	}
	if ThresholdAvgPauseLong <= 0 {
		t.Error("ThresholdAvgPauseLong should be positive")
	}
	if DefaultCollectionInterval <= 0 {
		t.Error("DefaultCollectionInterval should be positive")
	}
	if DefaultMaxSamples <= 0 {
		t.Error("DefaultMaxSamples should be positive")
	}
}

func TestErrors(t *testing.T) {
	// Verify errors are defined
	if ErrCollectorAlreadyRunning == nil {
		t.Error("ErrCollectorAlreadyRunning should not be nil")
	}
	if ErrCollectorNotRunning == nil {
		t.Error("ErrCollectorNotRunning should not be nil")
	}
	if ErrInsufficientData == nil {
		t.Error("ErrInsufficientData should not be nil")
	}
}

// Benchmark tests
func BenchmarkNewGCMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewGCMetrics()
	}
}

func BenchmarkNewGCMetricsPooled(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := NewGCMetricsPooled()
		m.Release()
	}
}

func BenchmarkNewGCMetricsLite(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewGCMetricsLite()
	}
}

func BenchmarkGCMetrics_Clone(b *testing.B) {
	original := NewGCMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = original.Clone()
	}
}

func BenchmarkFormatBytes(b *testing.B) {
	sizes := []uint64{0, 1024, 1048576, 1073741824}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, size := range sizes {
			_ = FormatBytes(size)
		}
	}
}
