package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/analyzer"
)

func main() {
	fmt.Println("=== Advanced GC Analyzer Example ===")

	// Example 1: Continuous monitoring with callbacks
	fmt.Println("1. Continuous Monitoring with Callbacks")
	runContinuousMonitoring()

	// Example 2: Analyzing different workload patterns
	fmt.Println("\n2. Workload Pattern Analysis")
	analyzeWorkloadPatterns()

	// Example 3: Memory trend analysis
	fmt.Println("\n3. Memory Trend Analysis")
	analyzeMemoryTrends()

	// Example 4: Advanced reporting
	fmt.Println("\n4. Advanced Reporting")
	generateAdvancedReports()

	// Example 5: Performance comparison
	fmt.Println("\n5. Performance Comparison")
	compareGCPerformance()

	fmt.Println("\n=== Advanced Example Complete ===")
}

// runContinuousMonitoring demonstrates continuous GC monitoring with real-time callbacks
func runContinuousMonitoring() {
	var mu sync.Mutex
	var metrics []*analyzer.GCMetrics
	var events []*analyzer.GCEvent

	config := &analyzer.CollectorConfig{
		Interval:   200 * time.Millisecond,
		MaxSamples: 100,
		OnMetricCollected: func(m *analyzer.GCMetrics) {
			mu.Lock()
			defer mu.Unlock()
			metrics = append(metrics, m)

			// Real-time alerting example
			if m.GCCPUFraction > 0.1 { // More than 10% CPU in GC
				fmt.Printf("   ⚠️  High GC CPU usage detected: %.2f%%\n", m.GCCPUFraction*100)
			}
		},
		OnGCEvent: func(e *analyzer.GCEvent) {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, e)

			// Alert on long pauses
			if e.Duration > 10*time.Millisecond {
				fmt.Printf("   ⚠️  Long GC pause detected: %v\n", e.Duration.Round(time.Microsecond))
			}
		},
	}

	collector := analyzer.NewCollector(config)

	// Start monitoring
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := collector.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}

	// Generate workload while monitoring
	go func() {
		generateVariableWorkload(ctx)
	}()

	fmt.Println("   Monitoring for 3 seconds...")
	<-ctx.Done()
	collector.Stop()

	mu.Lock()
	defer mu.Unlock()
	fmt.Printf("   Collected %d metrics and %d GC events\n", len(metrics), len(events))
}

// analyzeWorkloadPatterns compares GC behavior under different workload patterns
func analyzeWorkloadPatterns() {
	patterns := []struct {
		name string
		fn   func(context.Context)
	}{
		{"High allocation rate", generateHighAllocationWorkload},
		{"Large object allocation", generateLargeObjectWorkload},
		{"Memory leak simulation", generateMemoryLeakWorkload},
	}

	for _, pattern := range patterns {
		fmt.Printf("   Analyzing pattern: %s\n", pattern.name)

		// Collect metrics for this pattern
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		// Start workload
		go pattern.fn(ctx)

		metrics, err := analyzer.CollectForDuration(ctx, 2*time.Second, 200*time.Millisecond)
		cancel()

		if err != nil {
			log.Printf("Failed to collect metrics for %s: %v", pattern.name, err)
			continue
		}

		if len(metrics) < 2 {
			fmt.Printf("     Insufficient data for analysis\n")
			continue
		}

		// Analyze the pattern
		gcAnalyzer := analyzer.NewAnalyzer(metrics)
		analysis, err := gcAnalyzer.Analyze()
		if err != nil {
			log.Printf("Failed to analyze %s: %v", pattern.name, err)
			continue
		}

		fmt.Printf("     GC Frequency: %.2f/s, Avg Pause: %v, Alloc Rate: %s/s\n",
			analysis.GCFrequency,
			analysis.AvgPauseTime.Round(time.Microsecond),
			formatBytes(uint64(analysis.AllocRate)))

		// Print top recommendation
		if len(analysis.Recommendations) > 0 {
			fmt.Printf("     💡 %s\n", analysis.Recommendations[0])
		}
	}
}

// analyzeMemoryTrends demonstrates memory trend analysis and visualization
func analyzeMemoryTrends() {
	fmt.Println("   Collecting data for memory trend analysis...")

	// Collect data with growing memory usage
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go generateGrowingMemoryWorkload(ctx)

	metrics, err := analyzer.CollectForDuration(ctx, 3*time.Second, 300*time.Millisecond)
	if err != nil {
		log.Printf("Failed to collect metrics: %v", err)
		return
	}

	if len(metrics) < 2 {
		fmt.Println("     Insufficient data for trend analysis")
		return
	}

	gcAnalyzer := analyzer.NewAnalyzer(metrics)
	memoryTrend := gcAnalyzer.GetMemoryTrend()

	fmt.Printf("   Memory Trend Analysis (%d data points):\n", len(memoryTrend))

	// Print memory usage over time
	for i, point := range memoryTrend {
		if i%3 == 0 { // Print every 3rd point to avoid clutter
			fmt.Printf("     %s: Heap=%s, Sys=%s, InUse=%s\n",
				point.Timestamp.Format("15:04:05.000"),
				formatBytes(point.HeapAlloc),
				formatBytes(point.HeapSys),
				formatBytes(point.HeapInuse))
		}
	}

	// Calculate growth rate
	if len(memoryTrend) >= 2 {
		first := memoryTrend[0]
		last := memoryTrend[len(memoryTrend)-1]
		duration := last.Timestamp.Sub(first.Timestamp)
		growth := int64(last.HeapAlloc) - int64(first.HeapAlloc)
		growthRate := float64(growth) / duration.Seconds()

		fmt.Printf("     Memory growth rate: %s/second\n", formatBytes(uint64(growthRate)))
	}
}

// generateAdvancedReports demonstrates various report formats
func generateAdvancedReports() {
	// Collect some sample data
	ctx := context.Background()
	metrics, err := analyzer.CollectForDuration(ctx, 1*time.Second, 200*time.Millisecond)
	if err != nil {
		log.Printf("Failed to collect metrics: %v", err)
		return
	}

	if len(metrics) < 2 {
		fmt.Println("   Insufficient data for reporting")
		return
	}

	gcAnalyzer := analyzer.NewAnalyzer(metrics)
	analysis, err := gcAnalyzer.Analyze()
	if err != nil {
		log.Printf("Failed to analyze metrics: %v", err)
		return
	}

	reporter := analyzer.NewReporter(analysis, metrics, nil)

	// 1. JSON Report
	fmt.Println("   Generating JSON report...")
	jsonFile, err := os.Create("gc_analysis.json")
	if err != nil {
		log.Printf("Failed to create JSON file: %v", err)
	} else {
		err = reporter.GenerateJSONReport(jsonFile, true)
		jsonFile.Close()
		if err != nil {
			log.Printf("Failed to generate JSON report: %v", err)
		} else {
			fmt.Println("     ✅ JSON report saved to gc_analysis.json")
		}
	}

	// 2. Prometheus/Grafana metrics
	fmt.Println("   Generating Prometheus metrics...")
	promFile, err := os.Create("gc_metrics.prom")
	if err != nil {
		log.Printf("Failed to create Prometheus file: %v", err)
	} else {
		err = reporter.GenerateGrafanaMetrics(promFile)
		promFile.Close()
		if err != nil {
			log.Printf("Failed to generate Prometheus metrics: %v", err)
		} else {
			fmt.Println("     ✅ Prometheus metrics saved to gc_metrics.prom")
		}
	}

	// 3. Detailed text report
	fmt.Println("   Generating detailed text report...")
	textFile, err := os.Create("gc_analysis.txt")
	if err != nil {
		log.Printf("Failed to create text file: %v", err)
	} else {
		err = reporter.GenerateTextReport(textFile)
		textFile.Close()
		if err != nil {
			log.Printf("Failed to generate text report: %v", err)
		} else {
			fmt.Println("     ✅ Text report saved to gc_analysis.txt")
		}
	}

	// 4. Health check as JSON
	fmt.Println("   Generating health check...")
	healthCheck := reporter.GenerateHealthCheck()
	healthFile, err := os.Create("gc_health.json")
	if err != nil {
		log.Printf("Failed to create health file: %v", err)
	} else {
		encoder := json.NewEncoder(healthFile)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(healthCheck)
		healthFile.Close()
		if err != nil {
			log.Printf("Failed to generate health check: %v", err)
		} else {
			fmt.Println("     ✅ Health check saved to gc_health.json")
		}
	}
}

// compareGCPerformance demonstrates before/after performance comparison
func compareGCPerformance() {
	fmt.Println("   Comparing GC performance with different GOGC values...")

	originalGOGC := os.Getenv("GOGC")
	defer func() {
		if originalGOGC != "" {
			os.Setenv("GOGC", originalGOGC)
		}
	}()

	gogcValues := []string{"100", "200", "50"}
	results := make(map[string]*analyzer.GCAnalysis)

	for _, gogc := range gogcValues {
		fmt.Printf("     Testing with GOGC=%s\n", gogc)

		// Set GOGC value
		os.Setenv("GOGC", gogc)
		runtime.GC() // Force GC to apply new settings

		// Collect metrics
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		go generateConsistentWorkload(ctx)

		metrics, err := analyzer.CollectForDuration(ctx, 2*time.Second, 250*time.Millisecond)
		cancel()

		if err != nil {
			log.Printf("Failed to collect metrics for GOGC=%s: %v", gogc, err)
			continue
		}

		if len(metrics) < 2 {
			fmt.Printf("       Insufficient data\n")
			continue
		}

		gcAnalyzer := analyzer.NewAnalyzer(metrics)
		analysis, err := gcAnalyzer.Analyze()
		if err != nil {
			log.Printf("Failed to analyze GOGC=%s: %v", gogc, err)
			continue
		}

		results[gogc] = analysis

		fmt.Printf("       GC Freq: %.2f/s, Avg Pause: %v, GC Overhead: %.2f%%\n",
			analysis.GCFrequency,
			analysis.AvgPauseTime.Round(time.Microsecond),
			analysis.GCOverhead)
	}

	// Find best performing configuration
	var bestGOGC string
	var bestScore float64

	for gogc, analysis := range results {
		// Simple scoring: lower frequency + lower overhead + lower pause times = better
		score := 100.0 - (analysis.GCFrequency*10 + analysis.GCOverhead + float64(analysis.AvgPauseTime.Nanoseconds())/1000000)

		if bestGOGC == "" || score > bestScore {
			bestGOGC = gogc
			bestScore = score
		}
	}

	if bestGOGC != "" {
		fmt.Printf("     🏆 Best performing GOGC value: %s (score: %.2f)\n", bestGOGC, bestScore)
	}
}

// Workload generators for different patterns

func generateVariableWorkload(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var data [][]byte
	iteration := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			iteration++

			// Variable allocation pattern
			allocCount := 50 + (iteration%100)*2
			for i := 0; i < allocCount; i++ {
				chunk := make([]byte, 512+iteration%1024)
				data = append(data, chunk)
			}

			// Periodic cleanup
			if iteration%20 == 0 && len(data) > 500 {
				data = data[len(data)/2:]
			}
		}
	}
}

func generateHighAllocationWorkload(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// High frequency, small allocations
			for i := 0; i < 1000; i++ {
				_ = make([]byte, 64) // 64 bytes, short-lived
			}
		}
	}
}

func generateLargeObjectWorkload(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var data [][]byte

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Large, infrequent allocations
			chunk := make([]byte, 1024*1024) // 1MB
			data = append(data, chunk)

			// Keep only last 10 chunks
			if len(data) > 10 {
				data = data[1:]
			}
		}
	}
}

func generateMemoryLeakWorkload(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var leak [][]byte

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate memory leak - never release old data
			chunk := make([]byte, 10*1024) // 10KB
			leak = append(leak, chunk)
		}
	}
}

func generateGrowingMemoryWorkload(ctx context.Context) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var data [][]byte
	iteration := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			iteration++

			// Gradually increasing memory usage
			chunkSize := 1024 * iteration // Growing chunks
			count := 10 + iteration       // More chunks over time

			for i := 0; i < count; i++ {
				chunk := make([]byte, chunkSize)
				data = append(data, chunk)
			}

			// Slow cleanup - memory keeps growing
			if len(data) > 1000 {
				data = data[100:] // Remove only 100 items
			}
		}
	}
}

func generateConsistentWorkload(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var data [][]byte

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Consistent allocation pattern
			for i := 0; i < 100; i++ {
				chunk := make([]byte, 1024) // 1KB
				data = append(data, chunk)
			}

			// Consistent cleanup
			if len(data) > 500 {
				data = data[100:] // Keep 400 items
			}
		}
	}
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
