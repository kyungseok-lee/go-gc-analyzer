package tests

import (
	"context"
	"testing"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/analyzer"
)

func TestCollector_StartStop(t *testing.T) {
	collector := analyzer.NewCollector(nil)

	if collector.IsRunning() {
		t.Error("Collector should not be running initially")
	}

	ctx := context.Background()
	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	if !collector.IsRunning() {
		t.Error("Collector should be running after start")
	}

	// Try to start again - should return error
	err = collector.Start(ctx)
	if err != analyzer.ErrCollectorAlreadyRunning {
		t.Errorf("Expected ErrCollectorAlreadyRunning, got %v", err)
	}

	collector.Stop()

	if collector.IsRunning() {
		t.Error("Collector should not be running after stop")
	}

	// Stopping again should be safe
	collector.Stop()
}

func TestCollector_Collection(t *testing.T) {
	var collectedMetrics []*analyzer.GCMetrics
	var collectedEvents []*analyzer.GCEvent

	config := &analyzer.CollectorConfig{
		Interval:   50 * time.Millisecond,
		MaxSamples: 10,
		OnMetricCollected: func(m *analyzer.GCMetrics) {
			collectedMetrics = append(collectedMetrics, m)
		},
		OnGCEvent: func(e *analyzer.GCEvent) {
			collectedEvents = append(collectedEvents, e)
		},
	}

	collector := analyzer.NewCollector(config)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Wait for collection to happen
	<-ctx.Done()
	collector.Stop()

	// Should have collected some metrics
	metrics := collector.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Expected to collect some metrics")
	}

	// Callback should have been called
	if len(collectedMetrics) == 0 {
		t.Error("Expected callback to be called")
	}

	// Check that metrics have timestamps
	for _, m := range metrics {
		if m.Timestamp.IsZero() {
			t.Error("Expected non-zero timestamp in metrics")
		}
	}
}

func TestCollector_MaxSamples(t *testing.T) {
	config := &analyzer.CollectorConfig{
		Interval:   10 * time.Millisecond,
		MaxSamples: 3,
	}

	collector := analyzer.NewCollector(config)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Wait for collection
	<-ctx.Done()
	collector.Stop()

	metrics := collector.GetMetrics()

	// Should not exceed max samples
	if len(metrics) > 3 {
		t.Errorf("Expected at most 3 samples, got %d", len(metrics))
	}
}

func TestCollector_GetLatestMetrics(t *testing.T) {
	collector := analyzer.NewCollector(&analyzer.CollectorConfig{
		Interval: 50 * time.Millisecond,
	})

	// Should return nil when no metrics collected
	latest := collector.GetLatestMetrics()
	if latest != nil {
		t.Error("Expected nil for latest metrics when none collected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Wait for at least one collection
	<-ctx.Done()
	collector.Stop()

	latest = collector.GetLatestMetrics()
	if latest == nil {
		t.Error("Expected latest metrics after collection")
	}

	// Latest should be the most recent
	allMetrics := collector.GetMetrics()
	if len(allMetrics) > 0 {
		expectedLatest := allMetrics[len(allMetrics)-1]
		if latest.Timestamp != expectedLatest.Timestamp {
			t.Error("Latest metrics doesn't match expected latest")
		}
	}
}

func TestCollector_Clear(t *testing.T) {
	collector := analyzer.NewCollector(&analyzer.CollectorConfig{
		Interval: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	<-ctx.Done()
	collector.Stop()

	// Should have some data
	if len(collector.GetMetrics()) == 0 {
		t.Error("Expected some metrics before clear")
	}

	collector.Clear()

	// Should be empty after clear
	if len(collector.GetMetrics()) != 0 {
		t.Error("Expected no metrics after clear")
	}

	if len(collector.GetEvents()) != 0 {
		t.Error("Expected no events after clear")
	}

	if collector.GetLatestMetrics() != nil {
		t.Error("Expected no latest metrics after clear")
	}
}

func TestCollector_ContextCancellation(t *testing.T) {
	collector := analyzer.NewCollector(&analyzer.CollectorConfig{
		Interval: 10 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())

	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	if !collector.IsRunning() {
		t.Error("Collector should be running")
	}

	// Cancel context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Wait for context cancellation to take effect
	time.Sleep(100 * time.Millisecond)

	// Collector should still report as running (it doesn't auto-stop on context cancel)
	// This is by design - Stop() must be called explicitly
	if !collector.IsRunning() {
		t.Error("Collector should still be running until Stop() is called")
	}

	collector.Stop()
}

func TestCollectOnce(t *testing.T) {
	metrics := analyzer.CollectOnce()

	if metrics == nil {
		t.Fatal("Expected metrics, got nil")
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestCollectForDuration(t *testing.T) {
	ctx := context.Background()
	duration := 100 * time.Millisecond
	interval := 25 * time.Millisecond

	metrics, err := analyzer.CollectForDuration(ctx, duration, interval)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(metrics) == 0 {
		t.Error("Expected some metrics")
	}

	// Should have approximately duration/interval samples (±1 for timing)
	expectedSamples := int(duration / interval)
	if len(metrics) < expectedSamples-1 || len(metrics) > expectedSamples+2 {
		t.Errorf("Expected approximately %d samples, got %d", expectedSamples, len(metrics))
	}

	// Check that metrics are ordered by time
	for i := 1; i < len(metrics); i++ {
		if metrics[i].Timestamp.Before(metrics[i-1].Timestamp) {
			t.Error("Metrics should be ordered by timestamp")
		}
	}
}

func TestCollectForDuration_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	duration := 500 * time.Millisecond
	interval := 50 * time.Millisecond

	// Cancel context after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	metrics, err := analyzer.CollectForDuration(ctx, duration, interval)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	// Should still return some metrics collected before cancellation (optional)
	// Note: This might be empty if cancellation happens very quickly
	if len(metrics) == 0 {
		t.Log("No metrics collected before cancellation (timing dependent)")
	} else {
		t.Logf("Collected %d metrics before cancellation", len(metrics))
	}
}

func BenchmarkCollector_Start(b *testing.B) {
	for i := 0; i < b.N; i++ {
		collector := analyzer.NewCollector(&analyzer.CollectorConfig{
			Interval: time.Second, // Long interval to avoid actual collection
		})

		ctx := context.Background()
		err := collector.Start(ctx)
		if err != nil {
			b.Fatal(err)
		}

		collector.Stop()
	}
}

func BenchmarkCollectOnce_Collector(b *testing.B) {
	for i := 0; i < b.N; i++ {
		metrics := analyzer.CollectOnce()
		if metrics == nil {
			b.Fatal("Expected metrics")
		}
	}
}
