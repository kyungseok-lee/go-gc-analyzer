package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/analyzer"
)

func main() {
	fmt.Println("=== Basic GC Analyzer Example ===")

	// Example 1: Single point-in-time collection
	fmt.Println("1. Single Metrics Collection:")
	singleMetrics := analyzer.CollectOnce()
	fmt.Printf("   Current GC count: %d\n", singleMetrics.NumGC)
	fmt.Printf("   Current heap size: %s\n", singleMetrics.ToBytes(singleMetrics.HeapAlloc))
	fmt.Printf("   GC CPU fraction: %.2f%%\n\n", singleMetrics.GCCPUFraction*100)

	// Example 2: Collect metrics for a duration
	fmt.Println("2. Duration-based Collection:")
	fmt.Println("   Collecting metrics for 5 seconds while generating some load...")

	// Start background work to generate GC activity
	ctx, cancel := context.WithCancel(context.Background())
	go generateWorkload(ctx)

	// Collect metrics for 5 seconds with 500ms intervals
	metrics, err := analyzer.CollectForDuration(
		context.Background(),
		5*time.Second,
		500*time.Millisecond,
	)
	cancel() // Stop the workload

	if err != nil {
		log.Fatalf("Failed to collect metrics: %v", err)
	}

	fmt.Printf("   Collected %d metric samples\n", len(metrics))

	if len(metrics) >= 2 {
		first := metrics[0]
		last := metrics[len(metrics)-1]
		gcCount := last.NumGC - first.NumGC
		fmt.Printf("   GC count during collection: %d\n", gcCount)
		fmt.Printf("   Heap growth: %s -> %s\n",
			first.ToBytes(first.HeapAlloc),
			last.ToBytes(last.HeapAlloc))
	}

	// Example 3: Analyze the collected metrics
	fmt.Println("\n3. GC Performance Analysis:")

	if len(metrics) < 2 {
		fmt.Println("   Not enough data for analysis (need at least 2 samples)")
		return
	}

	gcAnalyzer := analyzer.NewAnalyzer(metrics)
	analysis, err := gcAnalyzer.Analyze()
	if err != nil {
		log.Fatalf("Failed to analyze metrics: %v", err)
	}

	// Display key analysis results
	fmt.Printf("   Analysis Period: %v\n", analysis.Period.Round(time.Second))
	fmt.Printf("   GC Frequency: %.2f GCs/second\n", analysis.GCFrequency)
	fmt.Printf("   Average GC Interval: %v\n", analysis.AvgGCInterval.Round(time.Millisecond))
	fmt.Printf("   Average Pause Time: %v\n", analysis.AvgPauseTime.Round(time.Microsecond))
	fmt.Printf("   Average Heap Size: %s\n", formatBytes(analysis.AvgHeapSize))
	fmt.Printf("   Allocation Rate: %s/second\n", formatBytes(uint64(analysis.AllocRate)))
	fmt.Printf("   GC Overhead: %.2f%%\n", analysis.GCOverhead)
	fmt.Printf("   Memory Efficiency: %.2f%%\n", analysis.MemoryEfficiency)

	// Example 4: Generate reports
	fmt.Println("\n4. Generate Reports:")

	reporter := analyzer.NewReporter(analysis, metrics, nil)

	// Generate summary report
	fmt.Println("   Summary Report:")
	fmt.Println("   " + strings.Repeat("-", 50))
	err = reporter.GenerateSummaryReport(os.Stdout)
	if err != nil {
		log.Printf("Failed to generate summary report: %v", err)
	}

	// Generate health check
	healthCheck := reporter.GenerateHealthCheck()
	fmt.Printf("\n   Health Check: %s (Score: %d/100)\n",
		healthCheck.Status, healthCheck.Score)

	if len(healthCheck.Issues) > 0 {
		fmt.Println("   Issues detected:")
		for _, issue := range healthCheck.Issues {
			fmt.Printf("   - %s\n", issue)
		}
	}

	// Show recommendations if any
	if len(analysis.Recommendations) > 0 {
		fmt.Println("\n   Recommendations:")
		for i, rec := range analysis.Recommendations {
			fmt.Printf("   %d. %s\n", i+1, rec)
		}
	}

	fmt.Println("\n=== Example Complete ===")
}

// generateWorkload creates some memory allocation patterns to trigger GC activity
func generateWorkload(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var data [][]byte

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Allocate some memory
			for i := 0; i < 100; i++ {
				chunk := make([]byte, 1024) // 1KB chunks
				data = append(data, chunk)
			}

			// Occasionally clean up some data to create GC pressure
			if len(data) > 1000 {
				data = data[500:] // Keep last 500 chunks
			}

			// Force GC occasionally
			if len(data)%200 == 0 {
				runtime.GC()
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
