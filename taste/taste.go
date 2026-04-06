// Package taste provides LLM-native personalization for termiflow-engine.
//
// Feed behavioral signals (reads, likes, skips), build taste profiles,
// and wrap any LLMProvider to inject personalization context into scoring prompts.
//
// Usage:
//
//	profile := taste.UpdateProfile(nil, signals)
//	personalLLM := taste.PersonalizeScorer(llm, profile)
//	curator := engine.NewCurator(personalLLM)
package taste

import (
	"math"
	"time"
)

// Signal kind constants.
const (
	SignalRead    = "read"
	SignalLike    = "like"
	SignalDislike = "dislike"
	SignalSkip    = "skip"
	SignalDwell   = "dwell"
)

// Internal thresholds.
const (
	coldStartThreshold = 20  // signals before switching from fast to normal alpha
	coldStartAlpha     = 0.3 // faster EMA adaptation for new users
)

// Signal represents a single user behavioral event.
// Value is always normalized to 0.0-1.0 by the caller (clamped defensively):
//   - "read": completion percentage (0.0 = opened, 1.0 = finished)
//   - "like": always 1.0
//   - "dislike": always 1.0 (negative signal handled by Kind, not Value)
//   - "skip": always 1.0 (negative signal)
//   - "dwell": normalized by caller (e.g., seconds / max_expected_seconds)
//
// Signals with empty Topics are ignored by UpdateProfile.
type Signal struct {
	ContentID string
	Kind      string
	Value     float64
	Topics    []string
	Timestamp time.Time
}

// TasteProfile represents a user's learned content preferences.
// Serializable to/from JSON. Designed to be user-owned and inspectable.
type TasteProfile struct {
	Version     int                `json:"version"`
	UserID      string             `json:"user_id"`
	Affinities  map[string]float64 `json:"affinities"`
	SignalCount int                `json:"signal_count"`
	LastUpdated time.Time          `json:"last_updated"`
}

// UpdateOption configures UpdateProfile behavior.
type UpdateOption func(*updateConfig)

type updateConfig struct {
	alpha float64
}

func defaultUpdateConfig() updateConfig {
	return updateConfig{alpha: 0.1}
}

// WithAlpha sets the EMA smoothing factor (default: 0.1).
// Higher alpha = faster adaptation to new signals, less memory of old ones.
// Values outside (0.0, 1.0] are ignored.
func WithAlpha(alpha float64) UpdateOption {
	return func(c *updateConfig) {
		if alpha > 0 && alpha <= 1.0 {
			c.alpha = alpha
		}
	}
}

// UpdateProfile ingests a batch of signals and updates the taste profile
// using exponential moving average.
//
// If profile is nil, creates a new empty profile. Signals with empty Topics
// are silently ignored. Unknown signal kinds are silently ignored.
//
// EMA formula per topic per signal:
//
//	delta = +clampedValue for read/like/dwell, -clampedValue for dislike/skip
//	affinity[topic] = (1 - alpha) * affinity[topic] + alpha * delta
//	affinity clamped to [-1.0, 1.0]
//
// Cold-start boost: if profile.SignalCount < 20, alpha is boosted to 0.3
// for faster early adaptation (unless overridden by WithAlpha).
func UpdateProfile(profile *TasteProfile, signals []Signal, opts ...UpdateOption) *TasteProfile {
	if profile == nil {
		profile = &TasteProfile{
			Version:    1,
			Affinities: make(map[string]float64),
		}
	}
	if profile.Affinities == nil {
		profile.Affinities = make(map[string]float64)
	}

	cfg := defaultUpdateConfig()

	// Cold-start boost: use higher alpha for early signals.
	if profile.SignalCount < coldStartThreshold {
		cfg.alpha = coldStartAlpha
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	for _, sig := range signals {
		if len(sig.Topics) == 0 {
			continue
		}

		delta, ok := signalDelta(sig.Kind, sig.Value)
		if !ok {
			continue
		}

		for _, topic := range sig.Topics {
			old := profile.Affinities[topic]
			updated := (1-cfg.alpha)*old + cfg.alpha*delta
			profile.Affinities[topic] = clamp(updated, -1.0, 1.0)
		}

		profile.SignalCount++
	}

	profile.LastUpdated = time.Now()
	return profile
}

// signalDelta computes the delta for a signal kind and value.
// Returns false for unknown kinds or invalid values (NaN, Inf).
func signalDelta(kind string, value float64) (float64, bool) {
	// Reject NaN and Inf, which would poison the profile permanently.
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}

	// Clamp value to [0.0, 1.0] defensively.
	if value < 0 {
		value = 0
	} else if value > 1 {
		value = 1
	}

	switch kind {
	case SignalRead, SignalLike, SignalDwell:
		return value, true
	case SignalDislike, SignalSkip:
		return -value, true
	default:
		return 0, false
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
