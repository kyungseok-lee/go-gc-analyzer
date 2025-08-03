package analyzer

import (
	"sort"
	"time"
)

// Analyzer provides GC performance analysis capabilities
type Analyzer struct {
	metrics []*GCMetrics
	events  []*GCEvent
}

// NewAnalyzer creates a new analyzer with the provided metrics
func NewAnalyzer(metrics []*GCMetrics) *Analyzer {
	return &Analyzer{
		metrics: metrics,
	}
}

// NewAnalyzerWithEvents creates a new analyzer with metrics and events
func NewAnalyzerWithEvents(metrics []*GCMetrics, events []*GCEvent) *Analyzer {
	return &Analyzer{
		metrics: metrics,
		events:  events,
	}
}

// Analyze performs comprehensive GC analysis
func (a *Analyzer) Analyze() (*GCAnalysis, error) {
	if len(a.metrics) < 2 {
		return nil, ErrInsufficientData
	}

	first := a.metrics[0]
	last := a.metrics[len(a.metrics)-1]

	analysis := &GCAnalysis{
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
func (a *Analyzer) analyzeGCFrequency(analysis *GCAnalysis) {
	if len(a.metrics) < 2 {
		return
	}

	first := a.metrics[0]
	last := a.metrics[len(a.metrics)-1]

	gcCount := last.NumGC - first.NumGC

	if analysis.Period.Seconds() > 0 {
		analysis.GCFrequency = float64(gcCount) / analysis.Period.Seconds()
	}

	if gcCount > 0 {
		analysis.AvgGCInterval = analysis.Period / time.Duration(gcCount)
	}
}

// analyzePauseTimes analyzes GC pause time statistics
func (a *Analyzer) analyzePauseTimes(analysis *GCAnalysis) {
	if len(a.events) == 0 {
		// Fallback to analyzing pause data from metrics
		a.analyzePauseTimesFromMetrics(analysis)
		return
	}

	if len(a.events) == 0 {
		return
	}

	durations := make([]time.Duration, len(a.events))
	var total time.Duration

	for i, event := range a.events {
		durations[i] = event.Duration
		total += event.Duration
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	analysis.AvgPauseTime = total / time.Duration(len(durations))
	analysis.MinPauseTime = durations[0]
	analysis.MaxPauseTime = durations[len(durations)-1]

	// Calculate percentiles
	p95Index := int(float64(len(durations)) * 0.95)
	p99Index := int(float64(len(durations)) * 0.99)

	if p95Index < len(durations) {
		analysis.P95PauseTime = durations[p95Index]
	}
	if p99Index < len(durations) {
		analysis.P99PauseTime = durations[p99Index]
	}
}

// analyzePauseTimesFromMetrics analyzes pause times from metrics when events are not available
func (a *Analyzer) analyzePauseTimesFromMetrics(analysis *GCAnalysis) {
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

	// Find min/max from recent pause data
	var pauses []time.Duration
	for _, metrics := range a.metrics {
		for _, pauseNs := range metrics.PauseNs {
			if pauseNs > 0 {
				pauses = append(pauses, time.Duration(pauseNs))
			}
		}
	}

	if len(pauses) > 0 {
		sort.Slice(pauses, func(i, j int) bool {
			return pauses[i] < pauses[j]
		})

		analysis.MinPauseTime = pauses[0]
		analysis.MaxPauseTime = pauses[len(pauses)-1]

		// Calculate percentiles
		p95Index := int(float64(len(pauses)) * 0.95)
		p99Index := int(float64(len(pauses)) * 0.99)

		if p95Index < len(pauses) {
			analysis.P95PauseTime = pauses[p95Index]
		}
		if p99Index < len(pauses) {
			analysis.P99PauseTime = pauses[p99Index]
		}
	}
}

// analyzeMemoryUsage analyzes memory usage patterns
func (a *Analyzer) analyzeMemoryUsage(analysis *GCAnalysis) {
	if len(a.metrics) == 0 {
		return
	}

	var totalHeap uint64
	var minHeap, maxHeap uint64

	for i, metrics := range a.metrics {
		heapSize := metrics.HeapAlloc
		totalHeap += heapSize

		if i == 0 {
			minHeap = heapSize
			maxHeap = heapSize
		} else {
			if heapSize < minHeap {
				minHeap = heapSize
			}
			if heapSize > maxHeap {
				maxHeap = heapSize
			}
		}
	}

	analysis.AvgHeapSize = totalHeap / uint64(len(a.metrics))
	analysis.MinHeapSize = minHeap
	analysis.MaxHeapSize = maxHeap

	// Calculate heap growth rate
	if len(a.metrics) >= 2 && analysis.Period.Seconds() > 0 {
		first := a.metrics[0]
		last := a.metrics[len(a.metrics)-1]
		heapGrowth := int64(last.HeapAlloc) - int64(first.HeapAlloc)
		analysis.HeapGrowthRate = float64(heapGrowth) / analysis.Period.Seconds()
	}
}

// analyzeAllocations analyzes allocation patterns
func (a *Analyzer) analyzeAllocations(analysis *GCAnalysis) {
	if len(a.metrics) < 2 {
		return
	}

	first := a.metrics[0]
	last := a.metrics[len(a.metrics)-1]

	totalAllocs := last.TotalAlloc - first.TotalAlloc
	allocCount := last.Mallocs - first.Mallocs
	freeCount := last.Frees - first.Frees

	analysis.AllocCount = allocCount
	analysis.FreeCount = freeCount

	if analysis.Period.Seconds() > 0 {
		analysis.AllocRate = float64(totalAllocs) / analysis.Period.Seconds()
	}
}

// calculateEfficiencyMetrics calculates GC efficiency metrics
func (a *Analyzer) calculateEfficiencyMetrics(analysis *GCAnalysis) {
	if len(a.metrics) == 0 {
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
		avgHeapSys := totalHeapSys / uint64(len(a.metrics))

		if avgHeapSys > 0 {
			analysis.MemoryEfficiency = (float64(analysis.AvgHeapSize) / float64(avgHeapSys)) * 100
		}
	}
}

// generateRecommendations generates performance improvement recommendations
func (a *Analyzer) generateRecommendations(analysis *GCAnalysis) {
	recommendations := make([]string, 0)

	// High GC frequency recommendations
	if analysis.GCFrequency > 10 { // More than 10 GCs per second
		recommendations = append(recommendations,
			"High GC frequency detected. Consider reducing allocation rate or increasing GOGC value.")
	}

	// Long pause time recommendations
	if analysis.AvgPauseTime > 100*time.Millisecond {
		recommendations = append(recommendations,
			"Long GC pause times detected. Consider reducing heap size or optimizing allocation patterns.")
	}

	if analysis.P99PauseTime > 500*time.Millisecond {
		recommendations = append(recommendations,
			"Very long P99 pause times detected. This may impact application responsiveness.")
	}

	// Memory growth recommendations
	if analysis.HeapGrowthRate > 1024*1024*10 { // 10MB/s growth
		recommendations = append(recommendations,
			"High heap growth rate detected. Check for memory leaks or excessive allocations.")
	}

	// High GC overhead recommendations
	if analysis.GCOverhead > 25 { // More than 25% CPU time in GC
		recommendations = append(recommendations,
			"High GC overhead detected. Consider optimizing allocation patterns or tuning GC parameters.")
	}

	// Low memory efficiency recommendations
	if analysis.MemoryEfficiency < 50 { // Less than 50% memory efficiency
		recommendations = append(recommendations,
			"Low memory efficiency detected. Consider reducing heap fragmentation or optimizing data structures.")
	}

	// Allocation rate recommendations
	if analysis.AllocRate > 1024*1024*100 { // 100MB/s allocation rate
		recommendations = append(recommendations,
			"High allocation rate detected. Consider object pooling or reducing temporary object creation.")
	}

	// Memory leak detection
	if len(a.metrics) >= 10 {
		recentGrowth := a.calculateRecentGrowthTrend()
		if recentGrowth > 0.1 { // 10% consistent growth
			recommendations = append(recommendations,
				"Consistent memory growth detected. Investigate potential memory leaks.")
		}
	}

	analysis.Recommendations = recommendations
}

// calculateRecentGrowthTrend calculates the recent memory growth trend
func (a *Analyzer) calculateRecentGrowthTrend() float64 {
	if len(a.metrics) < 10 {
		return 0
	}

	// Look at the last 10 samples to detect trend
	recent := a.metrics[len(a.metrics)-10:]

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
	distribution := map[string]int{
		"0-1ms":    0,
		"1-5ms":    0,
		"5-10ms":   0,
		"10-50ms":  0,
		"50-100ms": 0,
		"100ms+":   0,
	}

	for _, event := range a.events {
		duration := event.Duration

		switch {
		case duration < time.Millisecond:
			distribution["0-1ms"]++
		case duration < 5*time.Millisecond:
			distribution["1-5ms"]++
		case duration < 10*time.Millisecond:
			distribution["5-10ms"]++
		case duration < 50*time.Millisecond:
			distribution["10-50ms"]++
		case duration < 100*time.Millisecond:
			distribution["50-100ms"]++
		default:
			distribution["100ms+"]++
		}
	}

	return distribution
}

// GetMemoryTrend returns memory usage trend over time
func (a *Analyzer) GetMemoryTrend() []MemoryPoint {
	points := make([]MemoryPoint, len(a.metrics))

	for i, metrics := range a.metrics {
		points[i] = MemoryPoint{
			Timestamp: metrics.Timestamp,
			HeapAlloc: metrics.HeapAlloc,
			HeapSys:   metrics.HeapSys,
			HeapInuse: metrics.HeapInuse,
		}
	}

	return points
}

// MemoryPoint represents a point in memory usage trend
type MemoryPoint struct {
	Timestamp time.Time `json:"timestamp"`
	HeapAlloc uint64    `json:"heap_alloc"`
	HeapSys   uint64    `json:"heap_sys"`
	HeapInuse uint64    `json:"heap_inuse"`
}
