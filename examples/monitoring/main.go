package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kyungseok-lee/go-gc-analyzer/pkg/gcanalyzer"
)

func main() {
	fmt.Println("=== GC Monitoring Service Example ===")

	// Create monitor with alert thresholds
	monitor := gcanalyzer.NewMonitor(&gcanalyzer.MonitorConfig{
		Interval:   time.Second,
		MaxSamples: 300, // Keep 5 minutes of data
		OnAlert: func(alert *gcanalyzer.Alert) {
			log.Printf("🚨 ALERT [%s]: %s (%.2f, threshold: %.2f)",
				alert.Severity, alert.Message, alert.Value, alert.Threshold)
		},
		OnMetric: func(m *gcanalyzer.GCMetrics) {
			// Log significant metrics changes
			if m.GCCPUFraction > 0.05 { // More than 5% CPU in GC
				log.Printf("📊 GC CPU Usage: %.2f%%", m.GCCPUFraction*100)
			}
		},
		OnGCEvent: func(e *gcanalyzer.GCEvent) {
			// Log long pause times
			if e.Duration > 10*time.Millisecond {
				log.Printf("⏱️  GC Pause: %v", e.Duration.Round(time.Microsecond))
			}
		},
	})

	// Start monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := monitor.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start monitoring: %v", err)
	}

	log.Println("🔍 GC Monitoring started...")
	log.Println("💡 Generating some workload to trigger GC activity...")

	// Start background workload for demonstration
	go generateApplicationWorkload(ctx)

	// Periodic analysis reporting
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				analysis, err := monitor.GetCurrentAnalysis()
				if err != nil {
					log.Printf("📊 Analysis not available: %v", err)
					continue
				}

				healthCheck := gcanalyzer.GenerateHealthCheck(analysis)
				log.Printf("📊 GC Health: %s (Score: %d/100)",
					healthCheck.Status, healthCheck.Score)

				if len(analysis.Recommendations) > 0 {
					log.Printf("💡 Recommendations:")
					for _, rec := range analysis.Recommendations {
						log.Printf("   - %s", rec)
					}
				}
			}
		}
	}()

	log.Println("Press Ctrl+C to stop...")

	// Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nShutting down monitoring service...")
	monitor.Stop()
	cancel()

	// Final analysis
	if analysis, err := monitor.GetCurrentAnalysis(); err == nil {
		fmt.Println("\n=== Final GC Analysis ===")
		gcanalyzer.GenerateSummaryReport(analysis, os.Stdout)
	}
}

// generateApplicationWorkload simulates a typical application workload
func generateApplicationWorkload(ctx context.Context) {
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

			log.Printf("🔄 Switching to workload pattern %d", patternIndex+1)
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
