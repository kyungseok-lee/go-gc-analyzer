package reporting

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/pkg/types"
)

// Report generation errors
var (
	ErrNoAnalysisData = errors.New("no analysis data available")
	ErrNoMetricsData  = errors.New("no metrics data available")
	ErrNoEventsData   = errors.New("no events data available")
)

// builderPool provides reusable strings.Builder to reduce allocations
var builderPool = sync.Pool{
	New: func() any {
		return &strings.Builder{}
	},
}

func getBuilder() *strings.Builder {
	b := builderPool.Get().(*strings.Builder)
	b.Reset()
	return b
}

func putBuilder(b *strings.Builder) {
	if b.Cap() > 64*1024 { // Don't pool very large builders
		return
	}
	builderPool.Put(b)
}

// Reporter provides various reporting formats for GC analysis.
// It generates human-readable and machine-readable reports from GC analysis data.
type Reporter struct {
	analysis *types.GCAnalysis
	metrics  []*types.GCMetrics
	events   []*types.GCEvent
}

// New creates a new reporter with the provided analysis data.
// Metrics and events are optional and can be nil.
func New(analysis *types.GCAnalysis, metrics []*types.GCMetrics, events []*types.GCEvent) *Reporter {
	return &Reporter{
		analysis: analysis,
		metrics:  metrics,
		events:   events,
	}
}

// GenerateTextReport generates a human-readable text report.
// It includes all analysis metrics, statistics, and recommendations.
// Optimized to reduce allocations by using strings.Builder.
func (r *Reporter) GenerateTextReport(w io.Writer) error {
	if r.analysis == nil {
		return ErrNoAnalysisData
	}

	b := getBuilder()
	defer putBuilder(b)

	// Pre-allocate reasonable capacity
	b.Grow(2048)

	// Title
	b.WriteString("=== Go GC Analysis Report ===\n\n")

	// Analysis period
	b.WriteString("Analysis Period: ")
	b.WriteString(r.analysis.Period.Round(time.Second).String())
	b.WriteString(" (from ")
	b.WriteString(r.analysis.StartTime.Format("2006-01-02 15:04:05"))
	b.WriteString(" to ")
	b.WriteString(r.analysis.EndTime.Format("2006-01-02 15:04:05"))
	b.WriteString(")\n\n")

	// GC Frequency
	b.WriteString("=== GC Frequency ===\n")
	b.WriteString("GC Frequency: ")
	b.WriteString(formatFloat(r.analysis.GCFrequency, 2))
	b.WriteString(" GCs/second\n")
	b.WriteString("Average GC Interval: ")
	b.WriteString(r.analysis.AvgGCInterval.Round(time.Millisecond).String())
	b.WriteString("\n\n")

	// Pause Times
	b.WriteString("=== GC Pause Times ===\n")
	b.WriteString("Average Pause: ")
	b.WriteString(r.analysis.AvgPauseTime.Round(time.Microsecond).String())
	b.WriteString("\n")
	b.WriteString("Min Pause: ")
	b.WriteString(r.analysis.MinPauseTime.Round(time.Microsecond).String())
	b.WriteString("\n")
	b.WriteString("Max Pause: ")
	b.WriteString(r.analysis.MaxPauseTime.Round(time.Microsecond).String())
	b.WriteString("\n")
	b.WriteString("P95 Pause: ")
	b.WriteString(r.analysis.P95PauseTime.Round(time.Microsecond).String())
	b.WriteString("\n")
	b.WriteString("P99 Pause: ")
	b.WriteString(r.analysis.P99PauseTime.Round(time.Microsecond).String())
	b.WriteString("\n\n")

	// Memory Usage
	b.WriteString("=== Memory Usage ===\n")
	b.WriteString("Average Heap Size: ")
	b.WriteString(types.FormatBytes(r.analysis.AvgHeapSize))
	b.WriteString("\n")
	b.WriteString("Min Heap Size: ")
	b.WriteString(types.FormatBytes(r.analysis.MinHeapSize))
	b.WriteString("\n")
	b.WriteString("Max Heap Size: ")
	b.WriteString(types.FormatBytes(r.analysis.MaxHeapSize))
	b.WriteString("\n")
	b.WriteString("Heap Growth Rate: ")
	b.WriteString(types.FormatBytesRate(r.analysis.HeapGrowthRate))
	b.WriteString("\n\n")

	// Allocation Stats
	b.WriteString("=== Allocation Statistics ===\n")
	b.WriteString("Allocation Rate: ")
	b.WriteString(types.FormatBytesRate(r.analysis.AllocRate))
	b.WriteString("\n")
	b.WriteString("Total Allocations: ")
	b.WriteString(strconv.FormatUint(r.analysis.AllocCount, 10))
	b.WriteString("\n")
	b.WriteString("Total Frees: ")
	b.WriteString(strconv.FormatUint(r.analysis.FreeCount, 10))
	b.WriteString("\n\n")

	// Efficiency Metrics
	b.WriteString("=== Efficiency Metrics ===\n")
	b.WriteString("GC Overhead: ")
	b.WriteString(formatFloat(r.analysis.GCOverhead, 2))
	b.WriteString("%\n")
	b.WriteString("Memory Efficiency: ")
	b.WriteString(formatFloat(r.analysis.MemoryEfficiency, 2))
	b.WriteString("%\n\n")

	// Recommendations
	if len(r.analysis.Recommendations) > 0 {
		b.WriteString("=== Recommendations ===\n")
		for i, rec := range r.analysis.Recommendations {
			b.WriteString(strconv.Itoa(i + 1))
			b.WriteString(". ")
			b.WriteString(rec)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// formatFloat formats a float with the specified number of decimal places
func formatFloat(f float64, decimals int) string {
	return strconv.FormatFloat(f, 'f', decimals, 64)
}

// JSONReportOptions configures JSON report generation
type JSONReportOptions struct {
	// Indent enables pretty printing with indentation
	Indent bool
	// IncludeMetrics includes raw metrics data (can be large)
	IncludeMetrics bool
	// IncludeEvents includes raw events data
	IncludeEvents bool
	// CompactPauseData omits pause slice data from metrics to reduce size
	CompactPauseData bool
}

// GenerateJSONReport generates a JSON report
func (r *Reporter) GenerateJSONReport(w io.Writer, indent bool) error {
	return r.GenerateJSONReportWithOptions(w, JSONReportOptions{
		Indent:         indent,
		IncludeMetrics: true,
		IncludeEvents:  true,
	})
}

// GenerateJSONReportWithOptions generates a JSON report with configurable options
func (r *Reporter) GenerateJSONReportWithOptions(w io.Writer, opts JSONReportOptions) error {
	// Build report structure based on options
	type compactMetrics struct {
		NumGC         uint32    `json:"num_gc"`
		PauseTotalNs  uint64    `json:"pause_total_ns"`
		LastGC        time.Time `json:"last_gc"`
		HeapAlloc     uint64    `json:"heap_alloc"`
		HeapSys       uint64    `json:"heap_sys"`
		HeapInuse     uint64    `json:"heap_inuse"`
		HeapObjects   uint64    `json:"heap_objects"`
		GCCPUFraction float64   `json:"gc_cpu_fraction"`
		Timestamp     time.Time `json:"timestamp"`
	}

	var report any

	if opts.CompactPauseData && opts.IncludeMetrics && len(r.metrics) > 0 {
		// Use compact metrics without pause data
		compact := make([]compactMetrics, len(r.metrics))
		for i, m := range r.metrics {
			compact[i] = compactMetrics{
				NumGC:         m.NumGC,
				PauseTotalNs:  m.PauseTotalNs,
				LastGC:        m.LastGC,
				HeapAlloc:     m.HeapAlloc,
				HeapSys:       m.HeapSys,
				HeapInuse:     m.HeapInuse,
				HeapObjects:   m.HeapObjects,
				GCCPUFraction: m.GCCPUFraction,
				Timestamp:     m.Timestamp,
			}
		}

		var events []*types.GCEvent
		if opts.IncludeEvents {
			events = r.events
		}

		report = struct {
			Analysis *types.GCAnalysis `json:"analysis"`
			Metrics  []compactMetrics  `json:"metrics,omitempty"`
			Events   []*types.GCEvent  `json:"events,omitempty"`
		}{
			Analysis: r.analysis,
			Metrics:  compact,
			Events:   events,
		}
	} else {
		var metrics []*types.GCMetrics
		var events []*types.GCEvent

		if opts.IncludeMetrics {
			metrics = r.metrics
		}
		if opts.IncludeEvents {
			events = r.events
		}

		report = struct {
			Analysis *types.GCAnalysis  `json:"analysis"`
			Metrics  []*types.GCMetrics `json:"metrics,omitempty"`
			Events   []*types.GCEvent   `json:"events,omitempty"`
		}{
			Analysis: r.analysis,
			Metrics:  metrics,
			Events:   events,
		}
	}

	encoder := json.NewEncoder(w)
	if opts.Indent {
		encoder.SetIndent("", "  ")
	}

	return encoder.Encode(report)
}

// GenerateCompactJSONReport generates a compact JSON report without raw data
func (r *Reporter) GenerateCompactJSONReport(w io.Writer) error {
	return r.GenerateJSONReportWithOptions(w, JSONReportOptions{
		Indent:         false,
		IncludeMetrics: false,
		IncludeEvents:  false,
	})
}

// GenerateTableReport generates a tabular report.
// It displays metrics in a tabulated format for easy reading.
func (r *Reporter) GenerateTableReport(w io.Writer) error {
	if len(r.metrics) == 0 {
		return ErrNoMetricsData
	}

	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer tw.Flush()

	b := getBuilder()
	defer putBuilder(b)

	// Header
	b.WriteString("Timestamp\tGC#\tHeap\tSys\tPause\tObjects\tAlloc/s\n")
	b.WriteString("---------\t---\t----\t---\t-----\t-------\t-------\n")
	io.WriteString(tw, b.String())
	b.Reset()

	// Data rows
	for i, metrics := range r.metrics {
		var allocRate string
		if i > 0 {
			prev := r.metrics[i-1]
			duration := metrics.Timestamp.Sub(prev.Timestamp)
			if duration > 0 {
				allocDiff := metrics.TotalAlloc - prev.TotalAlloc
				rate := float64(allocDiff) / duration.Seconds()
				allocRate = types.FormatBytesRate(rate)
			}
		}

		var avgPause time.Duration
		if metrics.NumGC > 0 && metrics.PauseTotalNs > 0 {
			avgPause = time.Duration(metrics.PauseTotalNs/uint64(metrics.NumGC)) * time.Nanosecond
		}

		b.WriteString(metrics.Timestamp.Format("15:04:05"))
		b.WriteByte('\t')
		b.WriteString(strconv.FormatUint(uint64(metrics.NumGC), 10))
		b.WriteByte('\t')
		b.WriteString(types.FormatBytes(metrics.HeapAlloc))
		b.WriteByte('\t')
		b.WriteString(types.FormatBytes(metrics.Sys))
		b.WriteByte('\t')
		b.WriteString(avgPause.Round(time.Microsecond).String())
		b.WriteByte('\t')
		b.WriteString(strconv.FormatUint(metrics.HeapObjects, 10))
		b.WriteByte('\t')
		b.WriteString(allocRate)
		b.WriteByte('\n')

		io.WriteString(tw, b.String())
		b.Reset()
	}

	return nil
}

// GenerateSummaryReport generates a concise summary report.
// It provides a quick overview of GC performance metrics.
func (r *Reporter) GenerateSummaryReport(w io.Writer) error {
	if r.analysis == nil {
		return ErrNoAnalysisData
	}

	b := getBuilder()
	defer putBuilder(b)
	b.Grow(512)

	b.WriteString("GC Summary Report\n")
	b.WriteString("=================\n\n")

	b.WriteString("Period: ")
	b.WriteString(r.analysis.Period.Round(time.Second).String())
	b.WriteString(" | GC Frequency: ")
	b.WriteString(formatFloat(r.analysis.GCFrequency, 1))
	b.WriteString("/s | Avg Pause: ")
	b.WriteString(r.analysis.AvgPauseTime.Round(time.Microsecond).String())
	b.WriteString("\n")

	b.WriteString("Memory: ")
	b.WriteString(types.FormatBytes(r.analysis.AvgHeapSize))
	b.WriteString(" avg, ")
	b.WriteString(types.FormatBytes(r.analysis.MaxHeapSize))
	b.WriteString(" max | Alloc Rate: ")
	b.WriteString(types.FormatBytesRate(r.analysis.AllocRate))
	b.WriteString("\n")

	b.WriteString("Efficiency: ")
	b.WriteString(formatFloat(r.analysis.GCOverhead, 1))
	b.WriteString("% GC overhead, ")
	b.WriteString(formatFloat(r.analysis.MemoryEfficiency, 1))
	b.WriteString("% memory efficiency\n\n")

	if len(r.analysis.Recommendations) > 0 {
		b.WriteString("⚠️  Issues found: ")
		b.WriteString(strconv.Itoa(len(r.analysis.Recommendations)))
		b.WriteString(" recommendations\n")
	} else {
		b.WriteString("✅ No performance issues detected\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// GenerateEventsReport generates a report focused on GC events.
// It displays detailed information about each GC event.
func (r *Reporter) GenerateEventsReport(w io.Writer) error {
	if len(r.events) == 0 {
		return ErrNoEventsData
	}

	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer tw.Flush()

	b := getBuilder()
	defer putBuilder(b)

	b.WriteString("=== GC Events Report ===\n\n")
	b.WriteString("Seq#\tStart Time\tDuration\tTrigger\tHeap Before\tHeap After\tReleased\n")
	b.WriteString("----\t----------\t--------\t-------\t-----------\t----------\t--------\n")
	io.WriteString(tw, b.String())
	b.Reset()

	for _, event := range r.events {
		b.WriteString(strconv.FormatUint(uint64(event.Sequence), 10))
		b.WriteByte('\t')
		b.WriteString(event.StartTime.Format("15:04:05.000"))
		b.WriteByte('\t')
		b.WriteString(event.Duration.Round(time.Microsecond).String())
		b.WriteByte('\t')
		b.WriteString(event.TriggerReason)
		b.WriteByte('\t')
		b.WriteString(types.FormatBytes(event.HeapBefore))
		b.WriteByte('\t')
		b.WriteString(types.FormatBytes(event.HeapAfter))
		b.WriteByte('\t')
		b.WriteString(types.FormatBytes(event.HeapReleased))
		b.WriteByte('\n')

		io.WriteString(tw, b.String())
		b.Reset()
	}

	return nil
}

// GenerateGrafanaMetrics generates metrics in Prometheus/Grafana format.
// It outputs metrics in the Prometheus exposition format for integration with monitoring systems.
func (r *Reporter) GenerateGrafanaMetrics(w io.Writer) error {
	if r.analysis == nil {
		return ErrNoAnalysisData
	}

	b := getBuilder()
	defer putBuilder(b)
	b.Grow(1024)

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	b.WriteString("# HELP gc_frequency_total Number of garbage collections per second\n")
	b.WriteString("# TYPE gc_frequency_total gauge\n")
	b.WriteString("gc_frequency_total ")
	b.WriteString(formatFloat(r.analysis.GCFrequency, 6))
	b.WriteByte(' ')
	b.WriteString(timestamp)
	b.WriteString("\n\n")

	b.WriteString("# HELP gc_pause_time_avg_seconds Average GC pause time in seconds\n")
	b.WriteString("# TYPE gc_pause_time_avg_seconds gauge\n")
	b.WriteString("gc_pause_time_avg_seconds ")
	b.WriteString(formatFloat(r.analysis.AvgPauseTime.Seconds(), 6))
	b.WriteByte(' ')
	b.WriteString(timestamp)
	b.WriteString("\n\n")

	b.WriteString("# HELP gc_pause_time_p99_seconds P99 GC pause time in seconds\n")
	b.WriteString("# TYPE gc_pause_time_p99_seconds gauge\n")
	b.WriteString("gc_pause_time_p99_seconds ")
	b.WriteString(formatFloat(r.analysis.P99PauseTime.Seconds(), 6))
	b.WriteByte(' ')
	b.WriteString(timestamp)
	b.WriteString("\n\n")

	b.WriteString("# HELP heap_size_avg_bytes Average heap size in bytes\n")
	b.WriteString("# TYPE heap_size_avg_bytes gauge\n")
	b.WriteString("heap_size_avg_bytes ")
	b.WriteString(strconv.FormatUint(r.analysis.AvgHeapSize, 10))
	b.WriteByte(' ')
	b.WriteString(timestamp)
	b.WriteString("\n\n")

	b.WriteString("# HELP allocation_rate_bytes_per_second Allocation rate in bytes per second\n")
	b.WriteString("# TYPE allocation_rate_bytes_per_second gauge\n")
	b.WriteString("allocation_rate_bytes_per_second ")
	b.WriteString(formatFloat(r.analysis.AllocRate, 2))
	b.WriteByte(' ')
	b.WriteString(timestamp)
	b.WriteString("\n\n")

	b.WriteString("# HELP gc_overhead_percent GC overhead as percentage of CPU time\n")
	b.WriteString("# TYPE gc_overhead_percent gauge\n")
	b.WriteString("gc_overhead_percent ")
	b.WriteString(formatFloat(r.analysis.GCOverhead, 2))
	b.WriteByte(' ')
	b.WriteString(timestamp)
	b.WriteString("\n\n")

	_, err := io.WriteString(w, b.String())
	return err
}

// GenerateHealthCheck generates a health check status based on GC metrics
func (r *Reporter) GenerateHealthCheck() *types.HealthCheckStatus {
	if r.analysis == nil {
		return &types.HealthCheckStatus{
			Status:      "unknown",
			Score:       0,
			Issues:      []string{"No analysis data available"},
			Summary:     "Unable to determine GC health status",
			LastUpdated: time.Now(),
		}
	}

	status := &types.HealthCheckStatus{
		Status:      "healthy",
		Score:       100,
		Issues:      make([]string, 0, 6), // Pre-allocate with estimated capacity
		LastUpdated: time.Now(),
	}

	// Check GC frequency
	if r.analysis.GCFrequency > types.ThresholdGCFrequencyHigh {
		status.Score -= types.PenaltyGCFrequency
		status.Issues = append(status.Issues, "High GC frequency")
	}

	// Check pause times
	if r.analysis.AvgPauseTime > types.ThresholdAvgPauseLong {
		status.Score -= types.PenaltyAvgPause
		status.Issues = append(status.Issues, "Long average pause times")
	}
	if r.analysis.P99PauseTime > types.ThresholdP99PauseVeryLong {
		status.Score -= types.PenaltyP99Pause
		status.Issues = append(status.Issues, "Very long P99 pause times")
	}

	// Check GC overhead
	if r.analysis.GCOverhead > types.ThresholdGCOverheadHigh {
		status.Score -= types.PenaltyGCOverhead
		status.Issues = append(status.Issues, "High GC overhead")
	}

	// Check memory efficiency
	if r.analysis.MemoryEfficiency > 0 && r.analysis.MemoryEfficiency < types.ThresholdMemoryEfficiencyLow {
		status.Score -= types.PenaltyMemoryEfficiency
		status.Issues = append(status.Issues, "Low memory efficiency")
	}

	// Check allocation rate
	if r.analysis.AllocRate > types.ThresholdAllocationRateHigh {
		status.Score -= types.PenaltyAllocationRate
		status.Issues = append(status.Issues, "High allocation rate")
	}

	// Ensure score doesn't go below 0
	if status.Score < 0 {
		status.Score = 0
	}

	// Determine status based on score
	switch {
	case status.Score >= types.HealthScoreHealthy:
		status.Status = "healthy"
		status.Summary = "GC performance is good"
	case status.Score >= types.HealthScoreWarning:
		status.Status = "warning"
		status.Summary = "GC performance needs attention"
	default:
		status.Status = "critical"
		status.Summary = "GC performance issues detected"
	}

	return status
}
