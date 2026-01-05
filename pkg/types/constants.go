package types

import "time"

// Threshold constants for GC performance analysis and health checks
// These constants are used across the analyzer, reporter, and monitoring packages.
const (
	// GC frequency thresholds (GCs per second)
	ThresholdGCFrequencyHigh = 10.0

	// Pause time thresholds
	ThresholdAvgPauseLong     = 100 * time.Millisecond
	ThresholdP99PauseVeryLong = 500 * time.Millisecond
	ThresholdPauseWarning     = 100 * time.Millisecond
	ThresholdPauseCritical    = 500 * time.Millisecond

	// Memory thresholds (bytes per second)
	ThresholdHeapGrowthRateHigh = 10 * 1024 * 1024  // 10 MB/s
	ThresholdAllocationRateHigh = 100 * 1024 * 1024 // 100 MB/s

	// Efficiency thresholds (percentage)
	ThresholdGCOverheadHigh      = 25.0 // 25%
	ThresholdMemoryEfficiencyLow = 50.0 // 50%
	ThresholdGCCPUFractionAlert  = 0.25 // 25%

	// Growth trend thresholds
	ThresholdConsistentGrowth  = 0.1 // 10% consistent growth
	MinSamplesForTrendAnalysis = 10

	// Health score thresholds
	HealthScoreHealthy = 80
	HealthScoreWarning = 60

	// Health check penalties
	PenaltyGCFrequency      = 15
	PenaltyAvgPause         = 20
	PenaltyP99Pause         = 10
	PenaltyGCOverhead       = 25
	PenaltyMemoryEfficiency = 15
	PenaltyAllocationRate   = 10

	// Default configuration values
	DefaultCollectionInterval = time.Second
	DefaultMaxSamples         = 1000
)
