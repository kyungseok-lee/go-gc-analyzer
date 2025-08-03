package types

import (
	"fmt"
	"runtime"
	"time"
)

// GCMetrics represents comprehensive garbage collection metrics
type GCMetrics struct {
	// Basic GC stats
	NumGC        uint32    `json:"num_gc"`
	PauseTotalNs uint64    `json:"pause_total_ns"`
	PauseNs      []uint64  `json:"pause_ns"`
	PauseEnd     []uint64  `json:"pause_end"`
	LastGC       time.Time `json:"last_gc"`

	// Memory stats
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	Lookups    uint64 `json:"lookups"`
	Mallocs    uint64 `json:"mallocs"`
	Frees      uint64 `json:"frees"`

	// Heap stats
	HeapAlloc    uint64 `json:"heap_alloc"`
	HeapSys      uint64 `json:"heap_sys"`
	HeapIdle     uint64 `json:"heap_idle"`
	HeapInuse    uint64 `json:"heap_inuse"`
	HeapReleased uint64 `json:"heap_released"`
	HeapObjects  uint64 `json:"heap_objects"`

	// Stack stats
	StackInuse uint64 `json:"stack_inuse"`
	StackSys   uint64 `json:"stack_sys"`

	// GC performance metrics
	NextGC        uint64  `json:"next_gc"`
	GCCPUFraction float64 `json:"gc_cpu_fraction"`

	// Collection timestamp
	Timestamp time.Time `json:"timestamp"`
}

// GCAnalysis represents analyzed GC performance data
type GCAnalysis struct {
	// Collection period
	Period    time.Duration `json:"period"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`

	// GC frequency analysis
	GCFrequency   float64       `json:"gc_frequency"` // GCs per second
	AvgGCInterval time.Duration `json:"avg_gc_interval"`

	// Pause time analysis
	AvgPauseTime time.Duration `json:"avg_pause_time"`
	MaxPauseTime time.Duration `json:"max_pause_time"`
	MinPauseTime time.Duration `json:"min_pause_time"`
	P95PauseTime time.Duration `json:"p95_pause_time"`
	P99PauseTime time.Duration `json:"p99_pause_time"`

	// Memory analysis
	AvgHeapSize    uint64  `json:"avg_heap_size"`
	MaxHeapSize    uint64  `json:"max_heap_size"`
	MinHeapSize    uint64  `json:"min_heap_size"`
	HeapGrowthRate float64 `json:"heap_growth_rate"` // bytes per second

	// Allocation analysis
	AllocRate  float64 `json:"alloc_rate"`  // bytes per second
	AllocCount uint64  `json:"alloc_count"` // total allocations
	FreeCount  uint64  `json:"free_count"`  // total frees

	// Efficiency metrics
	GCOverhead       float64 `json:"gc_overhead"`       // percentage of CPU time spent in GC
	MemoryEfficiency float64 `json:"memory_efficiency"` // ratio of heap in use to heap allocated

	// Recommendations
	Recommendations []string `json:"recommendations"`
}

// GCEvent represents a single garbage collection event
type GCEvent struct {
	Sequence      uint32        `json:"sequence"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Duration      time.Duration `json:"duration"`
	HeapBefore    uint64        `json:"heap_before"`
	HeapAfter     uint64        `json:"heap_after"`
	HeapReleased  uint64        `json:"heap_released"`
	TriggerReason string        `json:"trigger_reason"`
}

// MemoryPoint represents a point in memory usage trend
type MemoryPoint struct {
	Timestamp time.Time `json:"timestamp"`
	HeapAlloc uint64    `json:"heap_alloc"`
	HeapSys   uint64    `json:"heap_sys"`
	HeapInuse uint64    `json:"heap_inuse"`
}

// HealthCheckStatus represents the health status based on GC analysis
type HealthCheckStatus struct {
	Status      string    `json:"status"` // healthy, warning, critical
	Score       int       `json:"score"`  // 0-100
	Issues      []string  `json:"issues"`
	Summary     string    `json:"summary"`
	LastUpdated time.Time `json:"last_updated"`
}

// NewGCMetrics creates a new GCMetrics from runtime.MemStats
func NewGCMetrics() *GCMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &GCMetrics{
		NumGC:         m.NumGC,
		PauseTotalNs:  m.PauseTotalNs,
		PauseNs:       make([]uint64, len(m.PauseNs)),
		PauseEnd:      make([]uint64, len(m.PauseEnd)),
		LastGC:        time.Unix(0, int64(m.LastGC)),
		Alloc:         m.Alloc,
		TotalAlloc:    m.TotalAlloc,
		Sys:           m.Sys,
		Lookups:       m.Lookups,
		Mallocs:       m.Mallocs,
		Frees:         m.Frees,
		HeapAlloc:     m.HeapAlloc,
		HeapSys:       m.HeapSys,
		HeapIdle:      m.HeapIdle,
		HeapInuse:     m.HeapInuse,
		HeapReleased:  m.HeapReleased,
		HeapObjects:   m.HeapObjects,
		StackInuse:    m.StackInuse,
		StackSys:      m.StackSys,
		NextGC:        m.NextGC,
		GCCPUFraction: m.GCCPUFraction,
		Timestamp:     time.Now(),
	}
}

// ToBytes converts size values to human-readable byte format
func (m *GCMetrics) ToBytes(size uint64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// ToDuration converts nanoseconds to human-readable duration
func (m *GCMetrics) ToDuration(ns uint64) time.Duration {
	return time.Duration(ns) * time.Nanosecond
}
