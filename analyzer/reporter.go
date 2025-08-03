package analyzer

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"
)

// Reporter provides various reporting formats for GC analysis
type Reporter struct {
	analysis *GCAnalysis
	metrics  []*GCMetrics
	events   []*GCEvent
}

// NewReporter creates a new reporter
func NewReporter(analysis *GCAnalysis, metrics []*GCMetrics, events []*GCEvent) *Reporter {
	return &Reporter{
		analysis: analysis,
		metrics:  metrics,
		events:   events,
	}
}

// GenerateTextReport generates a human-readable text report
func (r *Reporter) GenerateTextReport(w io.Writer) error {
	if r.analysis == nil {
		return fmt.Errorf("no analysis data available")
	}

	fmt.Fprintf(w, "=== Go GC Analysis Report ===\n\n")

	// Analysis period
	fmt.Fprintf(w, "Analysis Period: %v (from %v to %v)\n\n",
		r.analysis.Period.Round(time.Second),
		r.analysis.StartTime.Format("2006-01-02 15:04:05"),
		r.analysis.EndTime.Format("2006-01-02 15:04:05"))

	// GC Frequency
	fmt.Fprintf(w, "=== GC Frequency ===\n")
	fmt.Fprintf(w, "GC Frequency: %.2f GCs/second\n", r.analysis.GCFrequency)
	fmt.Fprintf(w, "Average GC Interval: %v\n\n", r.analysis.AvgGCInterval.Round(time.Millisecond))

	// Pause Times
	fmt.Fprintf(w, "=== GC Pause Times ===\n")
	fmt.Fprintf(w, "Average Pause: %v\n", r.analysis.AvgPauseTime.Round(time.Microsecond))
	fmt.Fprintf(w, "Min Pause: %v\n", r.analysis.MinPauseTime.Round(time.Microsecond))
	fmt.Fprintf(w, "Max Pause: %v\n", r.analysis.MaxPauseTime.Round(time.Microsecond))
	fmt.Fprintf(w, "P95 Pause: %v\n", r.analysis.P95PauseTime.Round(time.Microsecond))
	fmt.Fprintf(w, "P99 Pause: %v\n\n", r.analysis.P99PauseTime.Round(time.Microsecond))

	// Memory Usage
	fmt.Fprintf(w, "=== Memory Usage ===\n")
	fmt.Fprintf(w, "Average Heap Size: %s\n", formatBytes(r.analysis.AvgHeapSize))
	fmt.Fprintf(w, "Min Heap Size: %s\n", formatBytes(r.analysis.MinHeapSize))
	fmt.Fprintf(w, "Max Heap Size: %s\n", formatBytes(r.analysis.MaxHeapSize))
	fmt.Fprintf(w, "Heap Growth Rate: %s/second\n\n", formatBytes(uint64(r.analysis.HeapGrowthRate)))

	// Allocation Stats
	fmt.Fprintf(w, "=== Allocation Statistics ===\n")
	fmt.Fprintf(w, "Allocation Rate: %s/second\n", formatBytes(uint64(r.analysis.AllocRate)))
	fmt.Fprintf(w, "Total Allocations: %d\n", r.analysis.AllocCount)
	fmt.Fprintf(w, "Total Frees: %d\n\n", r.analysis.FreeCount)

	// Efficiency Metrics
	fmt.Fprintf(w, "=== Efficiency Metrics ===\n")
	fmt.Fprintf(w, "GC Overhead: %.2f%%\n", r.analysis.GCOverhead)
	fmt.Fprintf(w, "Memory Efficiency: %.2f%%\n\n", r.analysis.MemoryEfficiency)

	// Recommendations
	if len(r.analysis.Recommendations) > 0 {
		fmt.Fprintf(w, "=== Recommendations ===\n")
		for i, rec := range r.analysis.Recommendations {
			fmt.Fprintf(w, "%d. %s\n", i+1, rec)
		}
		fmt.Fprintf(w, "\n")
	}

	return nil
}

// GenerateJSONReport generates a JSON report
func (r *Reporter) GenerateJSONReport(w io.Writer, indent bool) error {
	report := struct {
		Analysis *GCAnalysis  `json:"analysis"`
		Metrics  []*GCMetrics `json:"metrics,omitempty"`
		Events   []*GCEvent   `json:"events,omitempty"`
	}{
		Analysis: r.analysis,
		Metrics:  r.metrics,
		Events:   r.events,
	}

	encoder := json.NewEncoder(w)
	if indent {
		encoder.SetIndent("", "  ")
	}

	return encoder.Encode(report)
}

// GenerateTableReport generates a tabular report
func (r *Reporter) GenerateTableReport(w io.Writer) error {
	if len(r.metrics) == 0 {
		return fmt.Errorf("no metrics data available")
	}

	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer tw.Flush()

	// Header
	fmt.Fprintf(tw, "Timestamp\tGC#\tHeap\tSys\tPause\tObjects\tAlloc/s\n")
	fmt.Fprintf(tw, "---------\t---\t----\t---\t-----\t-------\t-------\n")

	// Data rows
	for i, metrics := range r.metrics {
		var allocRate string
		if i > 0 {
			prev := r.metrics[i-1]
			duration := metrics.Timestamp.Sub(prev.Timestamp)
			if duration > 0 {
				allocDiff := metrics.TotalAlloc - prev.TotalAlloc
				rate := float64(allocDiff) / duration.Seconds()
				allocRate = formatBytes(uint64(rate)) + "/s"
			}
		}

		var avgPause time.Duration
		if metrics.NumGC > 0 && metrics.PauseTotalNs > 0 {
			avgPause = time.Duration(metrics.PauseTotalNs/uint64(metrics.NumGC)) * time.Nanosecond
		}

		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%v\t%d\t%s\n",
			metrics.Timestamp.Format("15:04:05"),
			metrics.NumGC,
			formatBytes(metrics.HeapAlloc),
			formatBytes(metrics.Sys),
			avgPause.Round(time.Microsecond),
			metrics.HeapObjects,
			allocRate,
		)
	}

	return nil
}

// GenerateSummaryReport generates a concise summary report
func (r *Reporter) GenerateSummaryReport(w io.Writer) error {
	if r.analysis == nil {
		return fmt.Errorf("no analysis data available")
	}

	fmt.Fprintf(w, "GC Summary Report\n")
	fmt.Fprintf(w, "=================\n\n")

	fmt.Fprintf(w, "Period: %v | GC Frequency: %.1f/s | Avg Pause: %v\n",
		r.analysis.Period.Round(time.Second),
		r.analysis.GCFrequency,
		r.analysis.AvgPauseTime.Round(time.Microsecond))

	fmt.Fprintf(w, "Memory: %s avg, %s max | Alloc Rate: %s/s\n",
		formatBytes(r.analysis.AvgHeapSize),
		formatBytes(r.analysis.MaxHeapSize),
		formatBytes(uint64(r.analysis.AllocRate)))

	fmt.Fprintf(w, "Efficiency: %.1f%% GC overhead, %.1f%% memory efficiency\n\n",
		r.analysis.GCOverhead,
		r.analysis.MemoryEfficiency)

	if len(r.analysis.Recommendations) > 0 {
		fmt.Fprintf(w, "⚠️  Issues found: %d recommendations\n", len(r.analysis.Recommendations))
	} else {
		fmt.Fprintf(w, "✅ No performance issues detected\n")
	}

	return nil
}

// GenerateEventsReport generates a report focused on GC events
func (r *Reporter) GenerateEventsReport(w io.Writer) error {
	if len(r.events) == 0 {
		return fmt.Errorf("no events data available")
	}

	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer tw.Flush()

	fmt.Fprintf(tw, "=== GC Events Report ===\n\n")
	fmt.Fprintf(tw, "Seq#\tStart Time\tDuration\tTrigger\tHeap Before\tHeap After\tReleased\n")
	fmt.Fprintf(tw, "----\t----------\t--------\t-------\t-----------\t----------\t--------\n")

	for _, event := range r.events {
		fmt.Fprintf(tw, "%d\t%s\t%v\t%s\t%s\t%s\t%s\n",
			event.Sequence,
			event.StartTime.Format("15:04:05.000"),
			event.Duration.Round(time.Microsecond),
			event.TriggerReason,
			formatBytes(event.HeapBefore),
			formatBytes(event.HeapAfter),
			formatBytes(event.HeapReleased),
		)
	}

	return nil
}

// GenerateGrafanaMetrics generates metrics in Prometheus/Grafana format
func (r *Reporter) GenerateGrafanaMetrics(w io.Writer) error {
	if r.analysis == nil {
		return fmt.Errorf("no analysis data available")
	}

	timestamp := time.Now().Unix()

	fmt.Fprintf(w, "# HELP gc_frequency_total Number of garbage collections per second\n")
	fmt.Fprintf(w, "# TYPE gc_frequency_total gauge\n")
	fmt.Fprintf(w, "gc_frequency_total %.6f %d\n\n", r.analysis.GCFrequency, timestamp)

	fmt.Fprintf(w, "# HELP gc_pause_time_avg_seconds Average GC pause time in seconds\n")
	fmt.Fprintf(w, "# TYPE gc_pause_time_avg_seconds gauge\n")
	fmt.Fprintf(w, "gc_pause_time_avg_seconds %.6f %d\n\n", r.analysis.AvgPauseTime.Seconds(), timestamp)

	fmt.Fprintf(w, "# HELP gc_pause_time_p99_seconds P99 GC pause time in seconds\n")
	fmt.Fprintf(w, "# TYPE gc_pause_time_p99_seconds gauge\n")
	fmt.Fprintf(w, "gc_pause_time_p99_seconds %.6f %d\n\n", r.analysis.P99PauseTime.Seconds(), timestamp)

	fmt.Fprintf(w, "# HELP heap_size_avg_bytes Average heap size in bytes\n")
	fmt.Fprintf(w, "# TYPE heap_size_avg_bytes gauge\n")
	fmt.Fprintf(w, "heap_size_avg_bytes %d %d\n\n", r.analysis.AvgHeapSize, timestamp)

	fmt.Fprintf(w, "# HELP allocation_rate_bytes_per_second Allocation rate in bytes per second\n")
	fmt.Fprintf(w, "# TYPE allocation_rate_bytes_per_second gauge\n")
	fmt.Fprintf(w, "allocation_rate_bytes_per_second %.2f %d\n\n", r.analysis.AllocRate, timestamp)

	fmt.Fprintf(w, "# HELP gc_overhead_percent GC overhead as percentage of CPU time\n")
	fmt.Fprintf(w, "# TYPE gc_overhead_percent gauge\n")
	fmt.Fprintf(w, "gc_overhead_percent %.2f %d\n\n", r.analysis.GCOverhead, timestamp)

	return nil
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// HealthCheckStatus represents the health status based on GC analysis
type HealthCheckStatus struct {
	Status      string    `json:"status"` // healthy, warning, critical
	Score       int       `json:"score"`  // 0-100
	Issues      []string  `json:"issues"`
	Summary     string    `json:"summary"`
	LastUpdated time.Time `json:"last_updated"`
}

// GenerateHealthCheck generates a health check status based on GC metrics
func (r *Reporter) GenerateHealthCheck() *HealthCheckStatus {
	if r.analysis == nil {
		return &HealthCheckStatus{
			Status:      "unknown",
			Score:       0,
			Issues:      []string{"No analysis data available"},
			Summary:     "Unable to determine GC health status",
			LastUpdated: time.Now(),
		}
	}

	status := &HealthCheckStatus{
		Status:      "healthy",
		Score:       100,
		Issues:      make([]string, 0),
		LastUpdated: time.Now(),
	}

	// Check GC frequency (penalty: -15 points)
	if r.analysis.GCFrequency > 10 {
		status.Score -= 15
		status.Issues = append(status.Issues, "High GC frequency")
	}

	// Check pause times (penalty: -20 points for avg, -10 for P99)
	if r.analysis.AvgPauseTime > 100*time.Millisecond {
		status.Score -= 20
		status.Issues = append(status.Issues, "Long average pause times")
	}
	if r.analysis.P99PauseTime > 500*time.Millisecond {
		status.Score -= 10
		status.Issues = append(status.Issues, "Very long P99 pause times")
	}

	// Check GC overhead (penalty: -25 points)
	if r.analysis.GCOverhead > 25 {
		status.Score -= 25
		status.Issues = append(status.Issues, "High GC overhead")
	}

	// Check memory efficiency (penalty: -15 points)
	if r.analysis.MemoryEfficiency < 50 {
		status.Score -= 15
		status.Issues = append(status.Issues, "Low memory efficiency")
	}

	// Check allocation rate (penalty: -10 points)
	if r.analysis.AllocRate > 1024*1024*100 { // 100MB/s
		status.Score -= 10
		status.Issues = append(status.Issues, "High allocation rate")
	}

	// Determine status based on score
	switch {
	case status.Score >= 80:
		status.Status = "healthy"
		status.Summary = "GC performance is good"
	case status.Score >= 60:
		status.Status = "warning"
		status.Summary = "GC performance needs attention"
	default:
		status.Status = "critical"
		status.Summary = "GC performance issues detected"
	}

	return status
}
