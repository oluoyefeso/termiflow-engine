package engine

import "errors"

var (
	// ErrProviderUnavailable indicates the LLM or search provider could not be reached.
	ErrProviderUnavailable = errors.New("engine: provider unavailable")

	// ErrAllScoringFailed indicates that every scoring call in a Curate batch failed.
	ErrAllScoringFailed = errors.New("engine: all scoring calls failed")
)
