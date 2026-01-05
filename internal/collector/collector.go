package collector

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/pkg/types"
)

// Collector is responsible for collecting GC metrics over time.
// It provides thread-safe metric collection with configurable intervals
// and supports callback functions for real-time monitoring.
type Collector struct {
	mu         sync.RWMutex
	running    atomic.Bool
	metrics    []*types.GCMetrics
	events     []*types.GCEvent
	interval   time.Duration
	maxSamples int
	stopCh     chan struct{}
	wg         sync.WaitGroup // Added for graceful shutdown

	// Callbacks
	onMetricCollected func(*types.GCMetrics)
	onGCEvent         func(*types.GCEvent)

	// useLiteMetrics controls whether to use lightweight metrics collection
	useLiteMetrics bool
}

// Config holds configuration for the collector
type Config struct {
	// Collection interval (default: 1 second)
	Interval time.Duration

	// Maximum number of samples to keep in memory (default: 1000)
	MaxSamples int

	// Callback functions
	OnMetricCollected func(*types.GCMetrics)
	OnGCEvent         func(*types.GCEvent)

	// UseLiteMetrics uses lightweight metrics without pause slice data (saves ~4KB per sample)
	UseLiteMetrics bool
}

// New creates a new GC metrics collector
func New(config *Config) *Collector {
	if config == nil {
		config = &Config{}
	}

	interval := config.Interval
	if interval == 0 {
		interval = types.DefaultCollectionInterval
	}

	maxSamples := config.MaxSamples
	if maxSamples == 0 {
		maxSamples = types.DefaultMaxSamples
	}

	return &Collector{
		interval:          interval,
		maxSamples:        maxSamples,
		metrics:           make([]*types.GCMetrics, 0, min(maxSamples, 256)), // Reasonable initial capacity
		events:            make([]*types.GCEvent, 0, min(maxSamples, 256)),
		stopCh:            make(chan struct{}),
		onMetricCollected: config.OnMetricCollected,
		onGCEvent:         config.OnGCEvent,
		useLiteMetrics:    config.UseLiteMetrics,
	}
}

// Start begins collecting GC metrics.
// Returns ErrCollectorAlreadyRunning if the collector is already running.
// The collector will stop when the context is cancelled or Stop() is called.
func (c *Collector) Start(ctx context.Context) error {
	if !c.running.CompareAndSwap(false, true) {
		return types.ErrCollectorAlreadyRunning
	}

	// Reset stop channel for potential restart
	c.mu.Lock()
	c.stopCh = make(chan struct{})
	c.mu.Unlock()

	c.wg.Add(1)
	go c.collectLoop(ctx)

	return nil
}

// Stop stops collecting GC metrics and waits for the collection loop to finish.
// It is safe to call Stop multiple times.
func (c *Collector) Stop() {
	if !c.running.CompareAndSwap(true, false) {
		return
	}

	c.mu.Lock()
	close(c.stopCh)
	c.mu.Unlock()

	// Wait for the collection loop to finish
	c.wg.Wait()
}

// IsRunning returns whether the collector is currently running
func (c *Collector) IsRunning() bool {
	return c.running.Load()
}

// GetMetrics returns a copy of all collected metrics
func (c *Collector) GetMetrics() []*types.GCMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.metrics) == 0 {
		return nil
	}

	result := make([]*types.GCMetrics, len(c.metrics))
	copy(result, c.metrics)
	return result
}

// GetEvents returns a copy of all collected GC events
func (c *Collector) GetEvents() []*types.GCEvent {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.events) == 0 {
		return nil
	}

	result := make([]*types.GCEvent, len(c.events))
	copy(result, c.events)
	return result
}

// GetLatestMetrics returns a copy of the most recent metrics sample
func (c *Collector) GetLatestMetrics() *types.GCMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.metrics) == 0 {
		return nil
	}

	// Return a deep copy to prevent race conditions
	latest := c.metrics[len(c.metrics)-1]
	return latest.Clone()
}

// Clear removes all collected metrics and events
func (c *Collector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear references for GC
	clear(c.metrics)
	clear(c.events)
	c.metrics = c.metrics[:0]
	c.events = c.events[:0]
}

// MetricCount returns the current number of collected metrics
func (c *Collector) MetricCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.metrics)
}

// EventCount returns the current number of collected events
func (c *Collector) EventCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.events)
}

// collectLoop runs the collection loop.
// It handles context cancellation and stop signals gracefully.
func (c *Collector) collectLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	var lastGCCount uint32

	for {
		select {
		case <-ctx.Done():
			c.running.Store(false)
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			var metrics *types.GCMetrics
			if c.useLiteMetrics {
				metrics = types.NewGCMetricsLite()
			} else {
				metrics = types.NewGCMetrics()
			}

			// Detect new GC events
			if lastGCCount > 0 && metrics.NumGC > lastGCCount {
				c.detectGCEvents(lastGCCount, metrics)
			}
			lastGCCount = metrics.NumGC

			c.addMetrics(metrics)

			// Call callback if provided
			if c.onMetricCollected != nil {
				c.onMetricCollected(metrics)
			}
		}
	}
}

// addMetrics adds a metrics sample to the collection
func (c *Collector) addMetrics(metrics *types.GCMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = append(c.metrics, metrics)

	// Keep only the last maxSamples samples using efficient trimming
	if len(c.metrics) > c.maxSamples {
		excess := len(c.metrics) - c.maxSamples
		// Zero out removed elements to allow GC
		for i := 0; i < excess; i++ {
			c.metrics[i] = nil
		}
		c.metrics = c.metrics[excess:]
	}
}

// detectGCEvents detects and records GC events
func (c *Collector) detectGCEvents(lastGCCount uint32, current *types.GCMetrics) {
	// Skip if no pause data available (lite mode)
	if len(current.PauseNs) == 0 {
		return
	}

	newGCCount := current.NumGC - lastGCCount
	pauseLen := uint32(len(current.PauseNs))

	for i := uint32(0); i < newGCCount; i++ {
		// Get pause time for this GC with wraparound handling
		pauseIndex := (current.NumGC - newGCCount + i) % pauseLen
		pauseNs := current.PauseNs[pauseIndex]

		// Get pause end time
		endNs := current.PauseEnd[pauseIndex]
		endTime := time.Unix(0, int64(endNs))
		startTime := endTime.Add(-time.Duration(pauseNs))

		event := &types.GCEvent{
			Sequence:      current.NumGC - newGCCount + i + 1,
			StartTime:     startTime,
			EndTime:       endTime,
			Duration:      time.Duration(pauseNs),
			TriggerReason: guessTriggerReason(current),
		}

		c.addEvent(event)

		// Call callback if provided
		if c.onGCEvent != nil {
			c.onGCEvent(event)
		}
	}
}

// addEvent adds a GC event to the collection
func (c *Collector) addEvent(event *types.GCEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.events = append(c.events, event)

	// Keep only the last maxSamples events
	if len(c.events) > c.maxSamples {
		excess := len(c.events) - c.maxSamples
		// Zero out removed elements to allow GC
		for i := 0; i < excess; i++ {
			c.events[i] = nil
		}
		c.events = c.events[excess:]
	}
}

// guessTriggerReason attempts to guess the GC trigger reason
func guessTriggerReason(metrics *types.GCMetrics) string {
	// This is a heuristic-based approach
	// In practice, Go doesn't expose the actual trigger reason

	if metrics.HeapAlloc >= metrics.NextGC {
		return "heap_size"
	}

	// Force GC detection (when NextGC is very low compared to heap)
	if metrics.NextGC < metrics.HeapAlloc/2 {
		return "forced"
	}

	// Periodic GC (when it's been a while since last GC)
	timeSinceLastGC := time.Since(metrics.LastGC)
	if timeSinceLastGC > 2*time.Minute {
		return "periodic"
	}

	return "automatic"
}

// CollectOnce collects a single GC metrics sample
func CollectOnce() *types.GCMetrics {
	return types.NewGCMetrics()
}

// CollectOnceLite collects a single lightweight GC metrics sample (no pause data)
func CollectOnceLite() *types.GCMetrics {
	return types.NewGCMetricsLite()
}

// CollectForDuration collects GC metrics for a specific duration
func CollectForDuration(ctx context.Context, duration time.Duration, interval time.Duration) ([]*types.GCMetrics, error) {
	if interval == 0 {
		interval = types.DefaultCollectionInterval
	}

	estimatedSamples := int(duration/interval) + 10 // Add buffer
	collector := New(&Config{
		Interval:   interval,
		MaxSamples: estimatedSamples,
	})

	if err := collector.Start(ctx); err != nil {
		return nil, err
	}

	// Wait for the duration
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		collector.Stop()
		return nil, ctx.Err()
	case <-timer.C:
		collector.Stop()
		return collector.GetMetrics(), nil
	}
}

// CollectForDurationLite collects lightweight GC metrics for a specific duration
func CollectForDurationLite(ctx context.Context, duration time.Duration, interval time.Duration) ([]*types.GCMetrics, error) {
	if interval == 0 {
		interval = types.DefaultCollectionInterval
	}

	estimatedSamples := int(duration/interval) + 10
	collector := New(&Config{
		Interval:       interval,
		MaxSamples:     estimatedSamples,
		UseLiteMetrics: true,
	})

	if err := collector.Start(ctx); err != nil {
		return nil, err
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		collector.Stop()
		return nil, ctx.Err()
	case <-timer.C:
		collector.Stop()
		return collector.GetMetrics(), nil
	}
}
