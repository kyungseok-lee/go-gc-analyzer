package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/analyzer"
)

// MonitoringService provides GC monitoring with HTTP endpoints
type MonitoringService struct {
	collector *analyzer.Collector
	analyzer  *analyzer.Analyzer
	mu        sync.RWMutex
	metrics   []*analyzer.GCMetrics
	events    []*analyzer.GCEvent
	analysis  *analyzer.GCAnalysis

	// Configuration
	alertThresholds AlertThresholds
}

// AlertThresholds defines when to trigger alerts
type AlertThresholds struct {
	MaxGCFrequency float64       // GCs per second
	MaxPauseTime   time.Duration // Maximum pause time
	MaxGCOverhead  float64       // Maximum GC CPU percentage
	MinHealthScore int           // Minimum health score
}

// DefaultAlertThresholds returns sensible default alert thresholds
func DefaultAlertThresholds() AlertThresholds {
	return AlertThresholds{
		MaxGCFrequency: 5.0,                   // 5 GCs per second
		MaxPauseTime:   50 * time.Millisecond, // 50ms pause
		MaxGCOverhead:  20.0,                  // 20% CPU in GC
		MinHealthScore: 70,                    // Health score below 70
	}
}

func main() {
	fmt.Println("=== GC Monitoring Service Example ===")

	// Create monitoring service
	service := NewMonitoringService(DefaultAlertThresholds())

	// Start monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := service.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start monitoring service: %v", err)
	}

	// Start HTTP server for metrics exposition
	go service.StartHTTPServer(":8080")

	// Start background workload for demonstration
	go generateApplicationWorkload(ctx)

	// Start alerting system
	go service.StartAlerting(ctx)

	fmt.Println("GC Monitoring Service started:")
	fmt.Println("  - HTTP server: http://localhost:8080")
	fmt.Println("  - Metrics endpoint: http://localhost:8080/metrics")
	fmt.Println("  - Health endpoint: http://localhost:8080/health")
	fmt.Println("  - Analysis endpoint: http://localhost:8080/analysis")
	fmt.Println("  - Prometheus endpoint: http://localhost:8080/prometheus")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop...")

	// Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nShutting down monitoring service...")
	service.Stop()
	cancel()
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(thresholds AlertThresholds) *MonitoringService {
	service := &MonitoringService{
		alertThresholds: thresholds,
		metrics:         make([]*analyzer.GCMetrics, 0),
		events:          make([]*analyzer.GCEvent, 0),
	}

	config := &analyzer.CollectorConfig{
		Interval:          1 * time.Second,
		MaxSamples:        300, // Keep 5 minutes of data
		OnMetricCollected: service.onMetricCollected,
		OnGCEvent:         service.onGCEvent,
	}

	service.collector = analyzer.NewCollector(config)

	return service
}

// Start begins the monitoring service
func (s *MonitoringService) Start(ctx context.Context) error {
	return s.collector.Start(ctx)
}

// Stop stops the monitoring service
func (s *MonitoringService) Stop() {
	s.collector.Stop()
}

// onMetricCollected handles new metric collection
func (s *MonitoringService) onMetricCollected(m *analyzer.GCMetrics) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics = append(s.metrics, m)

	// Keep last 300 samples (5 minutes at 1Hz)
	if len(s.metrics) > 300 {
		s.metrics = s.metrics[len(s.metrics)-300:]
	}

	// Reanalyze with new data
	if len(s.metrics) >= 2 {
		s.analyzer = analyzer.NewAnalyzerWithEvents(s.metrics, s.events)
		analysis, err := s.analyzer.Analyze()
		if err == nil {
			s.analysis = analysis
		}
	}
}

// onGCEvent handles new GC events
func (s *MonitoringService) onGCEvent(e *analyzer.GCEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, e)

	// Keep last 300 events
	if len(s.events) > 300 {
		s.events = s.events[len(s.events)-300:]
	}

	// Immediate alert for long pauses
	if e.Duration > s.alertThresholds.MaxPauseTime {
		log.Printf("🚨 ALERT: Long GC pause detected: %v (threshold: %v)",
			e.Duration.Round(time.Microsecond),
			s.alertThresholds.MaxPauseTime.Round(time.Microsecond))
	}
}

// StartAlerting starts the alerting system
func (s *MonitoringService) StartAlerting(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAlerts()
		}
	}
}

// checkAlerts checks current metrics against alert thresholds
func (s *MonitoringService) checkAlerts() {
	s.mu.RLock()
	analysis := s.analysis
	s.mu.RUnlock()

	if analysis == nil {
		return
	}

	// Check GC frequency
	if analysis.GCFrequency > s.alertThresholds.MaxGCFrequency {
		log.Printf("🚨 ALERT: High GC frequency: %.2f GCs/s (threshold: %.2f)",
			analysis.GCFrequency, s.alertThresholds.MaxGCFrequency)
	}

	// Check GC overhead
	if analysis.GCOverhead > s.alertThresholds.MaxGCOverhead {
		log.Printf("🚨 ALERT: High GC overhead: %.2f%% (threshold: %.2f%%)",
			analysis.GCOverhead, s.alertThresholds.MaxGCOverhead)
	}

	// Check health score
	reporter := analyzer.NewReporter(analysis, nil, nil)
	healthCheck := reporter.GenerateHealthCheck()

	if healthCheck.Score < s.alertThresholds.MinHealthScore {
		log.Printf("🚨 ALERT: Low GC health score: %d (threshold: %d) - %s",
			healthCheck.Score, s.alertThresholds.MinHealthScore, healthCheck.Summary)
	}
}

// StartHTTPServer starts the HTTP server for metrics exposition
func (s *MonitoringService) StartHTTPServer(addr string) {
	mux := http.NewServeMux()

	// Root endpoint - service info
	mux.HandleFunc("/", s.handleRoot)

	// Current metrics endpoint
	mux.HandleFunc("/metrics", s.handleMetrics)

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// Analysis endpoint
	mux.HandleFunc("/analysis", s.handleAnalysis)

	// Prometheus metrics endpoint
	mux.HandleFunc("/prometheus", s.handlePrometheus)

	// Memory trend endpoint
	mux.HandleFunc("/trend", s.handleTrend)

	// Pause distribution endpoint
	mux.HandleFunc("/distribution", s.handleDistribution)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("Starting HTTP server on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP server error: %v", err)
	}
}

// HTTP handlers

func (s *MonitoringService) handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `GC Monitoring Service

Available endpoints:
- /metrics       - Current GC metrics (JSON)
- /health        - Health check status (JSON)
- /analysis      - Full GC analysis (JSON)
- /prometheus    - Prometheus format metrics
- /trend         - Memory usage trend (JSON)
- /distribution  - Pause time distribution (JSON)
`)
}

func (s *MonitoringService) handleMetrics(w http.ResponseWriter, r *http.Request) {
	latest := s.collector.GetLatestMetrics()
	if latest == nil {
		http.Error(w, "No metrics available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(latest)
}

func (s *MonitoringService) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	analysis := s.analysis
	s.mu.RUnlock()

	if analysis == nil {
		http.Error(w, "Analysis not available", http.StatusServiceUnavailable)
		return
	}

	reporter := analyzer.NewReporter(analysis, nil, nil)
	healthCheck := reporter.GenerateHealthCheck()

	w.Header().Set("Content-Type", "application/json")

	// Set HTTP status based on health
	switch healthCheck.Status {
	case "healthy":
		w.WriteHeader(http.StatusOK)
	case "warning":
		w.WriteHeader(http.StatusOK) // 200 but with warnings
	case "critical":
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(healthCheck)
}

func (s *MonitoringService) handleAnalysis(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	analysis := s.analysis
	metrics := make([]*analyzer.GCMetrics, len(s.metrics))
	copy(metrics, s.metrics)
	events := make([]*analyzer.GCEvent, len(s.events))
	copy(events, s.events)
	s.mu.RUnlock()

	if analysis == nil {
		http.Error(w, "Analysis not available", http.StatusServiceUnavailable)
		return
	}

	response := struct {
		Analysis *analyzer.GCAnalysis  `json:"analysis"`
		Metrics  []*analyzer.GCMetrics `json:"metrics"`
		Events   []*analyzer.GCEvent   `json:"events"`
	}{
		Analysis: analysis,
		Metrics:  metrics,
		Events:   events,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *MonitoringService) handlePrometheus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	analysis := s.analysis
	metrics := make([]*analyzer.GCMetrics, len(s.metrics))
	copy(metrics, s.metrics)
	s.mu.RUnlock()

	if analysis == nil {
		http.Error(w, "Analysis not available", http.StatusServiceUnavailable)
		return
	}

	reporter := analyzer.NewReporter(analysis, metrics, nil)

	w.Header().Set("Content-Type", "text/plain")
	err := reporter.GenerateGrafanaMetrics(w)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate metrics: %v", err),
			http.StatusInternalServerError)
	}
}

func (s *MonitoringService) handleTrend(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.analyzer == nil {
		http.Error(w, "Analyzer not available", http.StatusServiceUnavailable)
		return
	}

	trend := s.analyzer.GetMemoryTrend()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trend)
}

func (s *MonitoringService) handleDistribution(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.analyzer == nil {
		http.Error(w, "Analyzer not available", http.StatusServiceUnavailable)
		return
	}

	distribution := s.analyzer.GetPauseTimeDistribution()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(distribution)
}

// generateApplicationWorkload simulates a typical application workload
func generateApplicationWorkload(ctx context.Context) {
	// Simulate different types of workloads throughout the day
	patterns := []func(context.Context){
		generateWebServerWorkload,
		generateBatchProcessingWorkload,
		generateCacheWorkload,
	}

	patternIndex := 0
	switchTimer := time.NewTicker(1 * time.Minute)
	defer switchTimer.Stop()

	currentCtx, currentCancel := context.WithCancel(ctx)
	go patterns[patternIndex](currentCtx)

	for {
		select {
		case <-ctx.Done():
			currentCancel()
			return
		case <-switchTimer.C:
			// Switch to next pattern
			currentCancel()
			patternIndex = (patternIndex + 1) % len(patterns)

			currentCtx, currentCancel = context.WithCancel(ctx)
			go patterns[patternIndex](currentCtx)

			log.Printf("Switching to workload pattern %d", patternIndex+1)
		}
	}
}

// Web server workload - frequent small allocations
func generateWebServerWorkload(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	var connections [][]byte

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate HTTP request processing
			for i := 0; i < 10; i++ {
				// Request data
				requestData := make([]byte, 1024+i%512)
				connections = append(connections, requestData)

				// Response data
				responseData := make([]byte, 2048+i%1024)
				_ = responseData // Simulate sending response
			}

			// Cleanup old connections
			if len(connections) > 1000 {
				connections = connections[500:]
			}
		}
	}
}

// Batch processing workload - large allocations with processing
func generateBatchProcessingWorkload(ctx context.Context) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var batches [][]byte

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Process a batch of data
			batchSize := 1024 * 1024 // 1MB batch
			batch := make([]byte, batchSize)

			// Simulate processing
			for i := 0; i < len(batch); i += 1024 {
				batch[i] = byte(i % 256)
			}

			batches = append(batches, batch)

			// Keep only last 5 batches
			if len(batches) > 5 {
				batches = batches[1:]
			}
		}
	}
}

// Cache workload - mixed read/write pattern
func generateCacheWorkload(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	cache := make(map[string][]byte)
	keyCounter := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 70% reads, 30% writes
			if keyCounter%10 < 7 && len(cache) > 0 {
				// Read from cache
				for key := range cache {
					_ = cache[key]
					break // Just read one item
				}
			} else {
				// Write to cache
				key := fmt.Sprintf("key_%d", keyCounter)
				value := make([]byte, 512+keyCounter%512)
				cache[key] = value
				keyCounter++

				// Periodic cache cleanup
				if len(cache) > 1000 {
					// Remove half of the cache
					count := 0
					for key := range cache {
						delete(cache, key)
						count++
						if count >= 500 {
							break
						}
					}
				}
			}
		}
	}
}
