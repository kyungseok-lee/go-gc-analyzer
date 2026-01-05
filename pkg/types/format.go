package types

import "fmt"

// FormatBytes formats bytes into human-readable format (KB, MB, GB, etc.)
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatBytesRate formats bytes per second into human-readable format
func FormatBytesRate(bytesPerSecond float64) string {
	if bytesPerSecond < 0 {
		return "0 B/s"
	}
	return FormatBytes(uint64(bytesPerSecond)) + "/s"
}

