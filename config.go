package engine

// CuratorOption configures a Curator instance.
type CuratorOption func(*curatorConfig)

type curatorConfig struct {
	maxConcurrency     int
	relevanceThreshold float64
	relevanceWeight    float64
	recencyWeight      float64
	scoringPrompt      string
}

func defaultConfig() curatorConfig {
	return curatorConfig{
		maxConcurrency:     5,
		relevanceThreshold: 0.5,
		relevanceWeight:    0.7,
		recencyWeight:      0.3,
		scoringPrompt:      DefaultScoringPrompt,
	}
}

// WithMaxConcurrency sets the max concurrent LLM calls during scoring/summarization.
// Default: 5.
func WithMaxConcurrency(n int) CuratorOption {
	return func(c *curatorConfig) {
		if n > 0 {
			c.maxConcurrency = n
		}
	}
}

// WithRelevanceThreshold sets the minimum relevance score (0.0-1.0) to include an item.
// Default: 0.5.
func WithRelevanceThreshold(t float64) CuratorOption {
	return func(c *curatorConfig) {
		if t >= 0 && t <= 1 {
			c.relevanceThreshold = t
		}
	}
}

// WithScoringWeight sets the relative weights for relevance and recency in final sorting.
// Both values should sum to ~1.0. Default: relevance=0.7, recency=0.3.
func WithScoringWeight(relevance, recency float64) CuratorOption {
	return func(c *curatorConfig) {
		if relevance >= 0 && recency >= 0 {
			c.relevanceWeight = relevance
			c.recencyWeight = recency
		}
	}
}

// WithScoringPrompt overrides the default scoring prompt template.
// The template must contain three %s placeholders: topic, title, snippet.
func WithScoringPrompt(prompt string) CuratorOption {
	return func(c *curatorConfig) {
		if prompt != "" {
			c.scoringPrompt = prompt
		}
	}
}
