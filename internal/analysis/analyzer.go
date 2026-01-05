package analysis

import (
	"cmp"
	"slices"
	"sync"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/pkg/types"
)

// durationSlicePool provides reusable duration slices to reduce allocations
var durationSlicePool = sync.Pool{
	New: func() any {
		// Pre-allocate with reasonable capacity
		s := make([]time.Duration, 0, 256)
		return &s
	},
}

func getDurationSlice() *[]time.Duration {
	s, ok := durationSlicePool.Get().(*[]time.Duration)
	if !ok {
		// Fallback: create new slice if type assertion fails
		newSlice := make([]time.Duration, 0, 256)
		return &newSlice
	}
	*s = (*s)[:0]
	return s
}

func putDurationSlice(s *[]time.Duration) {
	if cap(*s) > 4096 { // Don't pool very large slices
		return
	}
	durationSlicePool.Put(s)
}

// Analyzer provides GC performance analysis capabilities.
// It analyzes GC metrics and events to provide insights into GC behavior
// and generates recommendations for performance optimization.
type Analyzer struct {
	metrics []*types.GCMetrics
	events  []*types.GCEvent
}

// New creates a new analyzer with the provided metrics.
// Returns an Analyzer that can perform comprehensive GC analysis.
func New(metrics []*types.GCMetrics) *Analyzer {
	return &Analyzer{
		metrics: metrics,
	}
}

// NewWithEvents creates a new analyzer with metrics and events.
// Events provide more detailed pause time information for analysis.
func NewWithEvents(metrics []*types.GCMetrics, events []*types.GCEvent) *Analyzer {
	return &Analyzer{
		metrics: metrics,
		events:  events,
	}
}

// Analyze performs comprehensive GC analysis
func (a *Analyzer) Analyze() (*types.GCAnalysis, error) {
	if len(a.metrics) < 2 {
		return nil, types.ErrInsufficientData
	}

	first := a.metrics[0]
	last := a.metrics[len(a.metrics)-1]

	analysis := &types.GCAnalysis{
		Period:    last.Timestamp.Sub(first.Timestamp),
		StartTime: first.Timestamp,
		EndTime:   last.Timestamp,
	}

	// Analyze GC frequency
	a.analyzeGCFrequency(analysis)

	// Analyze pause times
	a.analyzePauseTimes(analysis)

	// Analyze memory usage
	a.analyzeMemoryUsage(analysis)

	// Analyze allocation patterns
	a.analyzeAllocations(analysis)

	// Calculate efficiency metrics
	a.calculateEfficiencyMetrics(analysis)

	// Generate recommendations
	a.generateRecommendations(analysis)

	return analysis, nil
}

// analyzeGCFrequency analyzes GC frequency patterns
func (a *Analyzer) analyzeGCFrequency(analysis *types.GCAnalysis) {
	if len(a.metrics) < 2 {
		return
	}

	first := a.metrics[0]
	last := a.metrics[len(a.metrics)-1]

	gcCount := last.NumGC - first.NumGC
	periodSeconds := analysis.Period.Seconds()

	if periodSeconds > 0 {
		analysis.GCFrequency = float64(gcCount) / periodSeconds
	}

	if gcCount > 0 {
		analysis.AvgGCInterval = analysis.Period / time.Duration(gcCount)
	}
}

// analyzePauseTimes analyzes GC pause time statistics.
// Uses events if available, otherwise falls back to metrics data.
func (a *Analyzer) analyzePauseTimes(analysis *types.GCAnalysis) {
	if len(a.events) == 0 {
		// Fallback to analyzing pause data from metrics
		a.analyzePauseTimesFromMetrics(analysis)
		return
	}

	n := len(a.events)

	// Use pooled slice to reduce allocations
	durationsPtr := getDurationSlice()
	defer putDurationSlice(durationsPtr)
	durations := *durationsPtr

	// Ensure capacity
	if cap(durations) < n {
		durations = make([]time.Duration, n)
	} else {
		durations = durations[:n]
	}

	var total time.Duration
	for i, event := range a.events {
		durations[i] = event.Duration
		total += event.Duration
	}

	// Use cmp.Compare for cleaner sorting (Go 1.21+)
	slices.SortFunc(durations, cmp.Compare)

	analysis.AvgPauseTime = total / time.Duration(n)
	analysis.MinPauseTime = durations[0]
	analysis.MaxPauseTime = durations[n-1]

	// Calculate percentiles with bounds checking
	analysis.P95PauseTime = durations[percentileIndex(n, 0.95)]
	analysis.P99PauseTime = durations[percentileIndex(n, 0.99)]
}

// percentileIndex calculates the index for a given percentile
func percentileIndex(n int, percentile float64) int {
	idx := int(float64(n-1) * percentile)
	if idx >= n {
		idx = n - 1
	}
	if idx < 0 {
		idx = 0
	}
	return idx
}

// analyzePauseTimesFromMetrics analyzes pause times from metrics when events are not available.
// This is a fallback method that extracts pause data from the PauseNs ring buffer.
func (a *Analyzer) analyzePauseTimesFromMetrics(analysis *types.GCAnalysis) {
	if len(a.metrics) < 2 {
		return
	}

	first := a.metrics[0]
	last := a.metrics[len(a.metrics)-1]

	// Calculate average pause time from total pause time
	totalGCs := last.NumGC - first.NumGC
	totalPauseTime := time.Duration(last.PauseTotalNs - first.PauseTotalNs)

	if totalGCs > 0 {
		analysis.AvgPauseTime = totalPauseTime / time.Duration(totalGCs)
	}

	// Count total non-zero pauses first for accurate capacity
	pauseCount := 0
	for _, metrics := range a.metrics {
		for _, pauseNs := range metrics.PauseNs {
			if pauseNs > 0 {
				pauseCount++
			}
		}
	}

	if pauseCount == 0 {
		return
	}

	// Use pooled slice
	pausesPtr := getDurationSlice()
	defer putDurationSlice(pausesPtr)
	pauses := *pausesPtr

	// Ensure capacity
	if cap(pauses) < pauseCount {
		pauses = make([]time.Duration, 0, pauseCount)
	}

	for _, metrics := range a.metrics {
		for _, pauseNs := range metrics.PauseNs {
			if pauseNs > 0 {
				pauses = append(pauses, time.Duration(pauseNs))
			}
		}
	}

	if len(pauses) > 0 {
		// Use cmp.Compare for cleaner sorting
		slices.SortFunc(pauses, cmp.Compare)

		n := len(pauses)
		analysis.MinPauseTime = pauses[0]
		analysis.MaxPauseTime = pauses[n-1]
		analysis.P95PauseTime = pauses[percentileIndex(n, 0.95)]
		analysis.P99PauseTime = pauses[percentileIndex(n, 0.99)]
	}
}

// analyzeMemoryUsage analyzes memory usage patterns
func (a *Analyzer) analyzeMemoryUsage(analysis *types.GCAnalysis) {
	n := len(a.metrics)
	if n == 0 {
		return
	}

	var totalHeap uint64
	minHeap := a.metrics[0].HeapAlloc
	maxHeap := a.metrics[0].HeapAlloc

	for _, metrics := range a.metrics {
		heapSize := metrics.HeapAlloc
		totalHeap += heapSize

		if heapSize < minHeap {
			minHeap = heapSize
		}
		if heapSize > maxHeap {
			maxHeap = heapSize
		}
	}

	analysis.AvgHeapSize = totalHeap / uint64(n)
	analysis.MinHeapSize = minHeap
	analysis.MaxHeapSize = maxHeap

	// Calculate heap growth rate
	if n >= 2 {
		periodSeconds := analysis.Period.Seconds()
		if periodSeconds > 0 {
			first := a.metrics[0]
			last := a.metrics[n-1]
			heapGrowth := int64(last.HeapAlloc) - int64(first.HeapAlloc)
			analysis.HeapGrowthRate = float64(heapGrowth) / periodSeconds
		}
	}
}

// analyzeAllocations analyzes allocation patterns
func (a *Analyzer) analyzeAllocations(analysis *types.GCAnalysis) {
	if len(a.metrics) < 2 {
		return
	}

	first := a.metrics[0]
	last := a.metrics[len(a.metrics)-1]

	totalAllocs := last.TotalAlloc - first.TotalAlloc
	analysis.AllocCount = last.Mallocs - first.Mallocs
	analysis.FreeCount = last.Frees - first.Frees

	periodSeconds := analysis.Period.Seconds()
	if periodSeconds > 0 {
		analysis.AllocRate = float64(totalAllocs) / periodSeconds
	}
}

// calculateEfficiencyMetrics calculates GC efficiency metrics
func (a *Analyzer) calculateEfficiencyMetrics(analysis *types.GCAnalysis) {
	n := len(a.metrics)
	if n == 0 {
		return
	}

	// Calculate average GC CPU fraction
	var totalGCCPUFraction float64
	validSamples := 0

	for _, metrics := range a.metrics {
		if metrics.GCCPUFraction >= 0 {
			totalGCCPUFraction += metrics.GCCPUFraction
			validSamples++
		}
	}

	if validSamples > 0 {
		analysis.GCOverhead = (totalGCCPUFraction / float64(validSamples)) * 100
	}

	// Calculate memory efficiency (heap in use vs heap allocated)
	if analysis.AvgHeapSize > 0 {
		var totalHeapSys uint64
		for _, metrics := range a.metrics {
			totalHeapSys += metrics.HeapSys
		}
		avgHeapSys := totalHeapSys / uint64(n)

		if avgHeapSys > 0 {
			analysis.MemoryEfficiency = (float64(analysis.AvgHeapSize) / float64(avgHeapSys)) * 100
		}
	}
}

// generateRecommendations generates performance improvement recommendations
func (a *Analyzer) generateRecommendations(analysis *types.GCAnalysis) {
	// Pre-allocate with estimated capacity
	recommendations := make([]string, 0, 8)

	// High GC frequency recommendations
	if analysis.GCFrequency > types.ThresholdGCFrequencyHigh {
		recommendations = append(recommendations,
			"High GC frequency detected. Consider reducing allocation rate or increasing GOGC value.")
	}

	// Long pause time recommendations
	if analysis.AvgPauseTime > types.ThresholdAvgPauseLong {
		recommendations = append(recommendations,
			"Long GC pause times detected. Consider reducing heap size or optimizing allocation patterns.")
	}

	if analysis.P99PauseTime > types.ThresholdP99PauseVeryLong {
		recommendations = append(recommendations,
			"Very long P99 pause times detected. This may impact application responsiveness.")
	}

	// Memory growth recommendations
	if analysis.HeapGrowthRate > types.ThresholdHeapGrowthRateHigh {
		recommendations = append(recommendations,
			"High heap growth rate detected. Check for memory leaks or excessive allocations.")
	}

	// High GC overhead recommendations
	if analysis.GCOverhead > types.ThresholdGCOverheadHigh {
		recommendations = append(recommendations,
			"High GC overhead detected. Consider optimizing allocation patterns or tuning GC parameters.")
	}

	// Low memory efficiency recommendations
	if analysis.MemoryEfficiency > 0 && analysis.MemoryEfficiency < types.ThresholdMemoryEfficiencyLow {
		recommendations = append(recommendations,
			"Low memory efficiency detected. Consider reducing heap fragmentation or optimizing data structures.")
	}

	// Allocation rate recommendations
	if analysis.AllocRate > types.ThresholdAllocationRateHigh {
		recommendations = append(recommendations,
			"High allocation rate detected. Consider object pooling or reducing temporary object creation.")
	}

	// Memory leak detection
	if len(a.metrics) >= types.MinSamplesForTrendAnalysis {
		recentGrowth := a.calculateRecentGrowthTrend()
		if recentGrowth > types.ThresholdConsistentGrowth {
			recommendations = append(recommendations,
				"Consistent memory growth detected. Investigate potential memory leaks.")
		}
	}

	analysis.Recommendations = recommendations
}

// calculateRecentGrowthTrend calculates the recent memory growth trend
func (a *Analyzer) calculateRecentGrowthTrend() float64 {
	n := len(a.metrics)
	if n < types.MinSamplesForTrendAnalysis {
		return 0
	}

	// Look at the last MinSamplesForTrendAnalysis samples to detect trend
	startIdx := n - types.MinSamplesForTrendAnalysis
	recent := a.metrics[startIdx:]

	var totalGrowth float64
	growthPoints := 0

	for i := 1; i < len(recent); i++ {
		prev := recent[i-1].HeapAlloc
		curr := recent[i].HeapAlloc

		if prev > 0 {
			growth := (float64(curr) - float64(prev)) / float64(prev)
			totalGrowth += growth
			growthPoints++
		}
	}

	if growthPoints > 0 {
		return totalGrowth / float64(growthPoints)
	}

	return 0
}

// GetPauseTimeDistribution returns pause time distribution data
func (a *Analyzer) GetPauseTimeDistribution() map[string]int {
	// Use pre-defined buckets for zero allocation
	distribution := map[string]int{
		"0-1ms":    0,
		"1-5ms":    0,
		"5-10ms":   0,
		"10-50ms":  0,
		"50-100ms": 0,
		"100ms+":   0,
	}

	for _, event := range a.events {
		bucket := getPauseTimeBucket(event.Duration)
		distribution[bucket]++
	}

	return distribution
}

// getPauseTimeBucket returns the bucket name for a given duration
func getPauseTimeBucket(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return "0-1ms"
	case d < 5*time.Millisecond:
		return "1-5ms"
	case d < 10*time.Millisecond:
		return "5-10ms"
	case d < 50*time.Millisecond:
		return "10-50ms"
	case d < 100*time.Millisecond:
		return "50-100ms"
	default:
		return "100ms+"
	}
}

// GetMemoryTrend returns memory usage trend over time
func (a *Analyzer) GetMemoryTrend() []types.MemoryPoint {
	n := len(a.metrics)
	if n == 0 {
		return nil
	}

	points := make([]types.MemoryPoint, n)
	for i, metrics := range a.metrics {
		points[i] = types.MemoryPoint{
			Timestamp: metrics.Timestamp,
			HeapAlloc: metrics.HeapAlloc,
			HeapSys:   metrics.HeapSys,
			HeapInuse: metrics.HeapInuse,
		}
	}

	return points
}

// Stats provides statistics about the analysis operation itself
type Stats struct {
	MetricCount   int
	EventCount    int
	PeriodSeconds float64
	GCCount       uint32
}

// GetStats returns statistics about the data being analyzed
func (a *Analyzer) GetStats() Stats {
	stats := Stats{
		MetricCount: len(a.metrics),
		EventCount:  len(a.events),
	}

	if len(a.metrics) >= 2 {
		first := a.metrics[0]
		last := a.metrics[len(a.metrics)-1]
		stats.PeriodSeconds = last.Timestamp.Sub(first.Timestamp).Seconds()
		stats.GCCount = last.NumGC - first.NumGC
	}

	return stats
}
