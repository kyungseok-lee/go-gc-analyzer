package analyzer

import (
	"context"
	"sync"
	"time"
)

// Collector is responsible for collecting GC metrics over time
type Collector struct {
	mu         sync.RWMutex
	isRunning  bool
	metrics    []*GCMetrics
	events     []*GCEvent
	interval   time.Duration
	maxSamples int
	stopCh     chan struct{}

	// Callbacks
	onMetricCollected func(*GCMetrics)
	onGCEvent         func(*GCEvent)
}

// CollectorConfig holds configuration for the collector
type CollectorConfig struct {
	// Collection interval (default: 1 second)
	Interval time.Duration

	// Maximum number of samples to keep in memory (default: 1000)
	MaxSamples int

	// Callback functions
	OnMetricCollected func(*GCMetrics)
	OnGCEvent         func(*GCEvent)
}

// NewCollector creates a new GC metrics collector
func NewCollector(config *CollectorConfig) *Collector {
	if config == nil {
		config = &CollectorConfig{}
	}

	if config.Interval == 0 {
		config.Interval = time.Second
	}

	if config.MaxSamples == 0 {
		config.MaxSamples = 1000
	}

	return &Collector{
		interval:          config.Interval,
		maxSamples:        config.MaxSamples,
		metrics:           make([]*GCMetrics, 0, config.MaxSamples),
		events:            make([]*GCEvent, 0, config.MaxSamples),
		stopCh:            make(chan struct{}),
		onMetricCollected: config.OnMetricCollected,
		onGCEvent:         config.OnGCEvent,
	}
}

// Start begins collecting GC metrics
func (c *Collector) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		return ErrCollectorAlreadyRunning
	}

	c.isRunning = true

	go c.collectLoop(ctx)

	return nil
}

// Stop stops collecting GC metrics
func (c *Collector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return
	}

	c.isRunning = false
	close(c.stopCh)
}

// IsRunning returns whether the collector is currently running
func (c *Collector) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

// GetMetrics returns a copy of all collected metrics
func (c *Collector) GetMetrics() []*GCMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*GCMetrics, len(c.metrics))
	copy(result, c.metrics)
	return result
}

// GetEvents returns a copy of all collected GC events
func (c *Collector) GetEvents() []*GCEvent {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*GCEvent, len(c.events))
	copy(result, c.events)
	return result
}

// GetLatestMetrics returns the most recent metrics sample
func (c *Collector) GetLatestMetrics() *GCMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.metrics) == 0 {
		return nil
	}

	return c.metrics[len(c.metrics)-1]
}

// Clear removes all collected metrics and events
func (c *Collector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = c.metrics[:0]
	c.events = c.events[:0]
}

// collectLoop runs the collection loop
func (c *Collector) collectLoop(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	var lastGCCount uint32

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			metrics := NewGCMetrics()

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
func (c *Collector) addMetrics(metrics *GCMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add to metrics slice
	c.metrics = append(c.metrics, metrics)

	// Keep only the last maxSamples samples
	if len(c.metrics) > c.maxSamples {
		c.metrics = c.metrics[len(c.metrics)-c.maxSamples:]
	}
}

// detectGCEvents detects and records GC events
func (c *Collector) detectGCEvents(lastGCCount uint32, current *GCMetrics) {
	newGCCount := current.NumGC - lastGCCount

	for i := uint32(0); i < newGCCount; i++ {
		// Get pause time for this GC
		pauseIndex := (current.NumGC - newGCCount + i) % uint32(len(current.PauseNs))
		pauseNs := current.PauseNs[pauseIndex]

		// Get pause end time
		endNs := current.PauseEnd[pauseIndex]
		endTime := time.Unix(0, int64(endNs))
		startTime := endTime.Add(-time.Duration(pauseNs))

		event := &GCEvent{
			Sequence:      current.NumGC - newGCCount + i + 1,
			StartTime:     startTime,
			EndTime:       endTime,
			Duration:      time.Duration(pauseNs),
			TriggerReason: c.guessTriggerReason(current),
		}

		c.addEvent(event)

		// Call callback if provided
		if c.onGCEvent != nil {
			c.onGCEvent(event)
		}
	}
}

// addEvent adds a GC event to the collection
func (c *Collector) addEvent(event *GCEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.events = append(c.events, event)

	// Keep only the last maxSamples events
	if len(c.events) > c.maxSamples {
		c.events = c.events[len(c.events)-c.maxSamples:]
	}
}

// guessTriggerReason attempts to guess the GC trigger reason
func (c *Collector) guessTriggerReason(metrics *GCMetrics) string {
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
func CollectOnce() *GCMetrics {
	return NewGCMetrics()
}

// CollectForDuration collects GC metrics for a specific duration
func CollectForDuration(ctx context.Context, duration time.Duration, interval time.Duration) ([]*GCMetrics, error) {
	if interval == 0 {
		interval = time.Second
	}

	collector := NewCollector(&CollectorConfig{
		Interval:   interval,
		MaxSamples: int(duration/interval) + 100, // Add some buffer
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
