// Package gcanalyzer provides a comprehensive Go garbage collection analyzer.
//
// This package offers tools for monitoring, analyzing, and reporting on Go's
// garbage collection performance. It provides both real-time monitoring
// capabilities and detailed analysis of GC behavior over time.
//
// Basic usage:
//
//	// Collect metrics for 10 seconds
//	metrics, err := gcanalyzer.CollectForDuration(ctx, 10*time.Second, time.Second)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Analyze the collected metrics
//	analysis, err := gcanalyzer.Analyze(metrics)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Generate a summary report
//	report := gcanalyzer.GenerateSummaryReport(analysis)
//	fmt.Println(report)
//
// For continuous monitoring:
//
//	monitor := gcanalyzer.NewMonitor(&gcanalyzer.MonitorConfig{
//		Interval: time.Second,
//		OnAlert: func(alert *gcanalyzer.Alert) {
//			log.Printf("GC Alert: %s", alert.Message)
//		},
//	})
//
//	monitor.Start(ctx)
//	defer monitor.Stop()
package gcanalyzer

import (
	"context"
	"io"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/internal/analysis"
	"github.com/kyungseok-lee/go-gc-analyzer/internal/collector"
	"github.com/kyungseok-lee/go-gc-analyzer/internal/reporting"
	"github.com/kyungseok-lee/go-gc-analyzer/pkg/types"
)

// Alert threshold constants
const (
	// GC CPU fraction thresholds
	AlertGCCPUFractionThreshold = 0.25 // 25%

	// Pause time thresholds
	AlertWarningPauseThreshold  = 100 * time.Millisecond
	AlertCriticalPauseThreshold = 500 * time.Millisecond
)

// Re-export commonly used types for convenience
type (
	GCMetrics         = types.GCMetrics
	GCAnalysis        = types.GCAnalysis
	GCEvent           = types.GCEvent
	MemoryPoint       = types.MemoryPoint
	HealthCheckStatus = types.HealthCheckStatus
)

// Re-export commonly used errors
var (
	ErrInsufficientData = types.ErrInsufficientData
)

// CollectOnce collects a single GC metrics snapshot
func CollectOnce() *GCMetrics {
	return collector.CollectOnce()
}

// CollectForDuration collects GC metrics for a specified duration
func CollectForDuration(ctx context.Context, duration, interval time.Duration) ([]*GCMetrics, error) {
	return collector.CollectForDuration(ctx, duration, interval)
}

// Analyze performs comprehensive analysis on the provided metrics
func Analyze(metrics []*GCMetrics) (*GCAnalysis, error) {
	analyzer := analysis.New(metrics)
	return analyzer.Analyze()
}

// AnalyzeWithEvents performs analysis with both metrics and events
func AnalyzeWithEvents(metrics []*GCMetrics, events []*GCEvent) (*GCAnalysis, error) {
	analyzer := analysis.NewWithEvents(metrics, events)
	return analyzer.Analyze()
}

// GenerateTextReport generates a detailed text report
func GenerateTextReport(analysis *GCAnalysis, metrics []*GCMetrics, events []*GCEvent, w io.Writer) error {
	reporter := reporting.New(analysis, metrics, events)
	return reporter.GenerateTextReport(w)
}

// GenerateJSONReport generates a JSON report
func GenerateJSONReport(analysis *GCAnalysis, metrics []*GCMetrics, events []*GCEvent, w io.Writer, indent bool) error {
	reporter := reporting.New(analysis, metrics, events)
	return reporter.GenerateJSONReport(w, indent)
}

// GenerateSummaryReport generates a concise summary report
func GenerateSummaryReport(analysis *GCAnalysis, w io.Writer) error {
	reporter := reporting.New(analysis, nil, nil)
	return reporter.GenerateSummaryReport(w)
}

// GenerateHealthCheck generates a health check status
func GenerateHealthCheck(analysis *GCAnalysis) *HealthCheckStatus {
	reporter := reporting.New(analysis, nil, nil)
	return reporter.GenerateHealthCheck()
}

// Monitor provides continuous GC monitoring capabilities
type Monitor struct {
	collector *collector.Collector
	config    *MonitorConfig
}

// MonitorConfig holds configuration for continuous monitoring
type MonitorConfig struct {
	// Collection interval (default: 1 second)
	Interval time.Duration

	// Maximum samples to keep in memory (default: 1000)
	MaxSamples int

	// Alert callback function
	OnAlert func(*Alert)

	// Metric collection callback
	OnMetric func(*GCMetrics)

	// GC event callback
	OnGCEvent func(*GCEvent)
}

// Alert represents a GC performance alert
type Alert struct {
	Type      string     `json:"type"`     // frequency, pause, overhead, memory
	Severity  string     `json:"severity"` // info, warning, critical
	Message   string     `json:"message"`
	Value     float64    `json:"value"`
	Threshold float64    `json:"threshold"`
	Metric    *GCMetrics `json:"metric,omitempty"`
	Event     *GCEvent   `json:"event,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}

// NewMonitor creates a new continuous GC monitor
func NewMonitor(config *MonitorConfig) *Monitor {
	if config == nil {
		config = &MonitorConfig{
			Interval:   time.Second,
			MaxSamples: 1000,
		}
	}

	if config.Interval == 0 {
		config.Interval = time.Second
	}

	if config.MaxSamples == 0 {
		config.MaxSamples = 1000
	}

	monitor := &Monitor{
		config: config,
	}

	// Create collector with alert-enabled callbacks
	collectorConfig := &collector.Config{
		Interval:   config.Interval,
		MaxSamples: config.MaxSamples,
		OnMetricCollected: func(m *types.GCMetrics) {
			if config.OnMetric != nil {
				config.OnMetric(m)
			}
			monitor.checkAlerts(m, nil)
		},
		OnGCEvent: func(e *types.GCEvent) {
			if config.OnGCEvent != nil {
				config.OnGCEvent(e)
			}
			monitor.checkAlerts(nil, e)
		},
	}

	monitor.collector = collector.New(collectorConfig)

	return monitor
}

// Start begins continuous monitoring
func (m *Monitor) Start(ctx context.Context) error {
	return m.collector.Start(ctx)
}

// Stop ends continuous monitoring
func (m *Monitor) Stop() {
	m.collector.Stop()
}

// IsRunning returns whether the monitor is currently running
func (m *Monitor) IsRunning() bool {
	return m.collector.IsRunning()
}

// GetMetrics returns all collected metrics
func (m *Monitor) GetMetrics() []*GCMetrics {
	return m.collector.GetMetrics()
}

// GetEvents returns all collected GC events
func (m *Monitor) GetEvents() []*GCEvent {
	return m.collector.GetEvents()
}

// GetLatestMetrics returns the most recent metrics
func (m *Monitor) GetLatestMetrics() *GCMetrics {
	return m.collector.GetLatestMetrics()
}

// GetCurrentAnalysis performs analysis on currently collected data
func (m *Monitor) GetCurrentAnalysis() (*GCAnalysis, error) {
	metrics := m.collector.GetMetrics()
	events := m.collector.GetEvents()

	if len(metrics) < 2 {
		return nil, ErrInsufficientData
	}

	analyzer := analysis.NewWithEvents(metrics, events)
	return analyzer.Analyze()
}

// checkAlerts checks for alert conditions
func (m *Monitor) checkAlerts(metric *GCMetrics, event *GCEvent) {
	if m.config.OnAlert == nil {
		return
	}

	// Check metric-based alerts
	if metric != nil {
		// High GC CPU fraction alert
		if metric.GCCPUFraction > AlertGCCPUFractionThreshold {
			alert := &Alert{
				Type:      "overhead",
				Severity:  "warning",
				Message:   "High GC CPU overhead detected",
				Value:     metric.GCCPUFraction * 100,
				Threshold: AlertGCCPUFractionThreshold * 100,
				Metric:    metric,
				Timestamp: time.Now(),
			}
			m.config.OnAlert(alert)
		}

		// Rapid heap growth alert
		// This would require historical data comparison
		// For simplicity, we'll skip this in the basic implementation
	}

	// Check event-based alerts
	if event != nil {
		// Long pause time alert
		if event.Duration > AlertWarningPauseThreshold {
			severity := "warning"
			if event.Duration > AlertCriticalPauseThreshold {
				severity = "critical"
			}

			alert := &Alert{
				Type:      "pause",
				Severity:  severity,
				Message:   "Long GC pause time detected",
				Value:     float64(event.Duration.Nanoseconds()) / 1e6, // ms
				Threshold: float64(AlertWarningPauseThreshold.Milliseconds()),
				Event:     event,
				Timestamp: time.Now(),
			}
			m.config.OnAlert(alert)
		}
	}
}

// Utility functions for easy access to analysis features

// GetMemoryTrend returns memory trend analysis for the given metrics
func GetMemoryTrend(metrics []*GCMetrics) []MemoryPoint {
	analyzer := analysis.New(metrics)
	return analyzer.GetMemoryTrend()
}

// GetPauseTimeDistribution returns pause time distribution for the given events
func GetPauseTimeDistribution(events []*GCEvent) map[string]int {
	analyzer := analysis.NewWithEvents(nil, events)
	return analyzer.GetPauseTimeDistribution()
}
