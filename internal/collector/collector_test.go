package collector

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/pkg/types"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		wantInterval   time.Duration
		wantMaxSamples int
	}{
		{
			name:           "nil config",
			config:         nil,
			wantInterval:   types.DefaultCollectionInterval,
			wantMaxSamples: types.DefaultMaxSamples,
		},
		{
			name:           "empty config",
			config:         &Config{},
			wantInterval:   types.DefaultCollectionInterval,
			wantMaxSamples: types.DefaultMaxSamples,
		},
		{
			name: "custom config",
			config: &Config{
				Interval:   500 * time.Millisecond,
				MaxSamples: 100,
			},
			wantInterval:   500 * time.Millisecond,
			wantMaxSamples: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.config)
			if c == nil {
				t.Fatal("New() returned nil")
			}
			if c.interval != tt.wantInterval {
				t.Errorf("interval = %v, want %v", c.interval, tt.wantInterval)
			}
			if c.maxSamples != tt.wantMaxSamples {
				t.Errorf("maxSamples = %d, want %d", c.maxSamples, tt.wantMaxSamples)
			}
		})
	}
}

func TestCollector_StartStop(t *testing.T) {
	c := New(&Config{
		Interval:   100 * time.Millisecond,
		MaxSamples: 10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start collector
	err := c.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	if !c.IsRunning() {
		t.Error("IsRunning() should be true after Start()")
	}

	// Wait for some metrics to be collected
	time.Sleep(300 * time.Millisecond)

	// Stop collector
	c.Stop()

	if c.IsRunning() {
		t.Error("IsRunning() should be false after Stop()")
	}

	// Verify metrics were collected
	metrics := c.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Should have collected some metrics")
	}
}

func TestCollector_DoubleStart(t *testing.T) {
	c := New(&Config{
		Interval:   100 * time.Millisecond,
		MaxSamples: 10,
	})

	ctx := context.Background()

	// First start
	err := c.Start(ctx)
	if err != nil {
		t.Fatalf("First Start() error: %v", err)
	}
	defer c.Stop()

	// Second start should fail
	err = c.Start(ctx)
	if err != types.ErrCollectorAlreadyRunning {
		t.Errorf("Second Start() should return ErrCollectorAlreadyRunning, got %v", err)
	}
}

func TestCollector_DoubleStop(t *testing.T) {
	c := New(&Config{
		Interval:   100 * time.Millisecond,
		MaxSamples: 10,
	})

	ctx := context.Background()
	_ = c.Start(ctx)

	// First stop
	c.Stop()

	// Second stop should not panic
	c.Stop()
}

func TestCollector_ContextCancellation(t *testing.T) {
	c := New(&Config{
		Interval:   100 * time.Millisecond,
		MaxSamples: 10,
	})

	ctx, cancel := context.WithCancel(context.Background())

	err := c.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for collector to stop
	time.Sleep(200 * time.Millisecond)

	if c.IsRunning() {
		t.Error("Collector should stop when context is canceled")
	}
}

func TestCollector_GetMetrics_Empty(t *testing.T) {
	c := New(nil)
	metrics := c.GetMetrics()

	if metrics != nil {
		t.Errorf("GetMetrics() should return nil for empty collector, got %v", metrics)
	}
}

func TestCollector_GetEvents_Empty(t *testing.T) {
	c := New(nil)
	events := c.GetEvents()

	if events != nil {
		t.Errorf("GetEvents() should return nil for empty collector, got %v", events)
	}
}

func TestCollector_GetLatestMetrics_Empty(t *testing.T) {
	c := New(nil)
	metrics := c.GetLatestMetrics()

	if metrics != nil {
		t.Errorf("GetLatestMetrics() should return nil for empty collector, got %v", metrics)
	}
}

func TestCollector_Clear(t *testing.T) {
	c := New(&Config{
		Interval:   50 * time.Millisecond,
		MaxSamples: 100,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = c.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	c.Stop()

	// Verify we have data
	if c.MetricCount() == 0 {
		t.Fatal("Should have collected metrics")
	}

	// Clear data
	c.Clear()

	if c.MetricCount() != 0 {
		t.Errorf("MetricCount() after Clear() = %d, want 0", c.MetricCount())
	}
	if c.EventCount() != 0 {
		t.Errorf("EventCount() after Clear() = %d, want 0", c.EventCount())
	}
}

func TestCollector_MaxSamples(t *testing.T) {
	c := New(&Config{
		Interval:   10 * time.Millisecond,
		MaxSamples: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = c.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	c.Stop()

	// Should not exceed maxSamples
	if c.MetricCount() > 5 {
		t.Errorf("MetricCount() = %d, should not exceed maxSamples (5)", c.MetricCount())
	}
}

func TestCollector_Callbacks(t *testing.T) {
	var metricCallbackCount int
	var mu sync.Mutex

	c := New(&Config{
		Interval:   50 * time.Millisecond,
		MaxSamples: 100,
		OnMetricCollected: func(m *types.GCMetrics) {
			mu.Lock()
			metricCallbackCount++
			mu.Unlock()
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = c.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	c.Stop()

	mu.Lock()
	count := metricCallbackCount
	mu.Unlock()

	if count == 0 {
		t.Error("OnMetricCollected callback should have been called")
	}
}

func TestCollector_LiteMetrics(t *testing.T) {
	c := New(&Config{
		Interval:       50 * time.Millisecond,
		MaxSamples:     10,
		UseLiteMetrics: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	_ = c.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	c.Stop()

	metrics := c.GetMetrics()
	if len(metrics) == 0 {
		t.Fatal("Should have collected metrics")
	}

	// Lite metrics should have nil PauseNs
	for _, m := range metrics {
		if m.PauseNs != nil {
			t.Error("Lite metrics should have nil PauseNs")
			break
		}
	}
}

func TestCollectOnce(t *testing.T) {
	metrics := CollectOnce()

	if metrics == nil {
		t.Fatal("CollectOnce() returned nil")
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	// Basic sanity checks
	if metrics.HeapSys == 0 {
		t.Error("HeapSys should not be zero")
	}
}

func TestCollectOnceLite(t *testing.T) {
	metrics := CollectOnceLite()

	if metrics == nil {
		t.Fatal("CollectOnceLite() returned nil")
	}

	if metrics.PauseNs != nil {
		t.Error("Lite metrics should have nil PauseNs")
	}
}

func TestCollectForDuration(t *testing.T) {
	ctx := context.Background()
	metrics, err := CollectForDuration(ctx, 200*time.Millisecond, 50*time.Millisecond)

	if err != nil {
		t.Fatalf("CollectForDuration() error: %v", err)
	}

	if len(metrics) == 0 {
		t.Error("Should have collected metrics")
	}

	// Verify timestamps are increasing
	for i := 1; i < len(metrics); i++ {
		if !metrics[i].Timestamp.After(metrics[i-1].Timestamp) {
			t.Error("Timestamps should be increasing")
		}
	}
}

func TestCollectForDuration_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := CollectForDuration(ctx, time.Second, 100*time.Millisecond)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestCollectForDuration_ZeroInterval(t *testing.T) {
	ctx := context.Background()
	// Use longer duration to ensure at least one collection with default interval (1s)
	metrics, err := CollectForDuration(ctx, 1500*time.Millisecond, 0)

	if err != nil {
		t.Fatalf("CollectForDuration() error: %v", err)
	}

	// Should use default interval (1 second), so ~1 sample expected
	if len(metrics) == 0 {
		t.Error("Should have collected metrics with default interval")
	}
}

func TestGuessTriggerReason(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *types.GCMetrics
		expected string
	}{
		{
			name: "heap_size trigger",
			metrics: &types.GCMetrics{
				HeapAlloc: 100 * 1024 * 1024,
				NextGC:    50 * 1024 * 1024,
			},
			expected: "heap_size",
		},
		{
			name: "automatic trigger",
			metrics: &types.GCMetrics{
				HeapAlloc: 50 * 1024 * 1024,
				NextGC:    100 * 1024 * 1024,
				LastGC:    time.Now(),
			},
			expected: "automatic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := guessTriggerReason(tt.metrics)
			if result != tt.expected {
				t.Errorf("guessTriggerReason() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// Concurrency test
func TestCollector_ConcurrentAccess(t *testing.T) {
	c := New(&Config{
		Interval:   10 * time.Millisecond,
		MaxSamples: 100,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_ = c.Start(ctx)

	var wg sync.WaitGroup
	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = c.GetMetrics()
				_ = c.GetEvents()
				_ = c.GetLatestMetrics()
				_ = c.MetricCount()
				_ = c.EventCount()
				_ = c.IsRunning()
			}
		}()
	}

	wg.Wait()
	c.Stop()
}

// Benchmark tests
func BenchmarkCollectOnce(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CollectOnce()
	}
}

func BenchmarkCollectOnceLite(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CollectOnceLite()
	}
}

func BenchmarkCollector_GetMetrics(b *testing.B) {
	c := New(&Config{
		Interval:   10 * time.Millisecond,
		MaxSamples: 1000,
	})

	ctx := context.Background()
	_ = c.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.GetMetrics()
	}

	b.StopTimer()
	c.Stop()
}
