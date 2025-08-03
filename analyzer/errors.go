package analyzer

import "errors"

// Common errors used by the analyzer package
var (
	ErrCollectorAlreadyRunning = errors.New("collector is already running")
	ErrCollectorNotRunning     = errors.New("collector is not running")
	ErrInsufficientData        = errors.New("insufficient data for analysis")
	ErrInvalidDuration         = errors.New("invalid duration specified")
	ErrInvalidInterval         = errors.New("invalid interval specified")
)
