package types

import (
	"strconv"
)

// Size units for formatting
const (
	_        = iota
	KB int64 = 1 << (10 * iota)
	MB
	GB
	TB
	PB
)

// byteUnits maps size thresholds to their unit suffixes for efficient lookup
var byteUnits = []struct {
	threshold int64
	suffix    string
	divisor   float64
}{
	{PB, " PB", float64(PB)},
	{TB, " TB", float64(TB)},
	{GB, " GB", float64(GB)},
	{MB, " MB", float64(MB)},
	{KB, " KB", float64(KB)},
}

// FormatBytes formats bytes into human-readable format (KB, MB, GB, etc.)
// Optimized to reduce allocations by avoiding fmt.Sprintf where possible.
func FormatBytes(bytes uint64) string {
	if bytes < 1024 {
		return strconv.FormatUint(bytes, 10) + " B"
	}

	b := int64(bytes)
	for _, unit := range byteUnits {
		if b >= unit.threshold {
			value := float64(bytes) / unit.divisor
			return formatFloat(value, 1) + unit.suffix
		}
	}

	return strconv.FormatUint(bytes, 10) + " B"
}

// FormatBytesRate formats bytes per second into human-readable format
func FormatBytesRate(bytesPerSecond float64) string {
	if bytesPerSecond < 0 {
		return "0 B/s"
	}
	return FormatBytes(uint64(bytesPerSecond)) + "/s"
}

// formatFloat formats a float with specified decimal places
// Optimized to reduce allocations compared to fmt.Sprintf
func formatFloat(value float64, decimals int) string {
	return strconv.FormatFloat(value, 'f', decimals, 64)
}
