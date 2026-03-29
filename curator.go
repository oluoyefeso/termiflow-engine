package engine

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"
)

// Curator orchestrates the search→score→summarize→tag→sort pipeline.
type Curator struct {
	llm LLMProvider
	cfg curatorConfig
}

// NewCurator creates a new Curator with the given LLM provider and options.
// Panics if llm is nil — this is a programmer error.
func NewCurator(llm LLMProvider, opts ...CuratorOption) *Curator {
	if llm == nil {
		panic("engine: NewCurator requires a non-nil LLMProvider")
	}
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Curator{llm: llm, cfg: cfg}
}

// Curate processes search results through the full pipeline: score → filter → summarize → tag → sort.
// Returns partial results on individual item failures (check FeedItem.Error for per-item issues).
// Returns error only on total pipeline failure (e.g., all scoring calls fail).
func (c *Curator) Curate(ctx context.Context, topic string, results []SearchResult) ([]*FeedItem, error) {
	if len(results) == 0 {
		return nil, nil
	}

	var (
		mu    sync.Mutex
		items = make([]*FeedItem, len(results))
		wg    sync.WaitGroup
		sem   = make(chan struct{}, c.cfg.maxConcurrency)
	)

	allFailed := true

	for i, result := range results {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, r SearchResult) {
			defer wg.Done()
			defer func() { <-sem }()

			item := &FeedItem{
				Title:       r.Title,
				SourceName:  r.Source,
				SourceURL:   r.URL,
				Content:     truncateContent(r.Content, 2000),
				PublishedAt: &r.PublishedAt,
			}

			// Score relevance
			score, err := scoreRelevanceWithPrompt(ctx, c.llm, c.cfg.scoringPrompt, topic, r.Title, r.Snippet)
			if err != nil {
				item.Error = err
				score = 0.5
			} else {
				mu.Lock()
				allFailed = false
				mu.Unlock()
			}
			item.RelevanceScore = score

			// Only summarize and tag items above threshold
			// Summarize and ExtractTags are independent — run in parallel.
			if score > c.cfg.relevanceThreshold {
				var innerWg sync.WaitGroup
				innerWg.Add(2)

				var summaryResult string
				var summaryErr error
				var tagsResult []string
				var tagsErr error

				go func() {
					defer innerWg.Done()
					summaryResult, summaryErr = Summarize(ctx, c.llm, topic, r.Title, r.Content)
				}()

				go func() {
					defer innerWg.Done()
					tagsResult, tagsErr = ExtractTags(ctx, c.llm, r.Title, r.Content)
				}()

				innerWg.Wait()

				if summaryErr != nil {
					item.Error = summaryErr
				} else {
					item.Summary = summaryResult
				}

				if tagsErr != nil {
					if item.Error == nil {
						item.Error = tagsErr
					}
				} else {
					item.Tags = tagsResult
				}
			}

			mu.Lock()
			items[idx] = item
			mu.Unlock()
		}(i, result)
	}

	wg.Wait()

	if allFailed && len(results) > 0 {
		return nil, ErrAllScoringFailed
	}

	// Collect non-nil items
	var collected []*FeedItem
	for _, item := range items {
		if item != nil {
			collected = append(collected, item)
		}
	}

	collected = filterByRelevance(collected, c.cfg.relevanceThreshold)
	sortByRelevanceAndRecency(collected, c.cfg.relevanceWeight, c.cfg.recencyWeight)

	return collected, nil
}

// truncateContent safely truncates content to maxLen runes (UTF-8 safe).
func truncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen])
}

func filterByRelevance(items []*FeedItem, threshold float64) []*FeedItem {
	var filtered []*FeedItem
	for _, item := range items {
		if item.RelevanceScore >= threshold {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// sortByRelevanceAndRecency sorts items by a weighted combination of relevance and recency.
// Recency uses exponential time-decay with a 1-week half-life instead of binary newer/older.
func sortByRelevanceAndRecency(items []*FeedItem, relevanceWeight, recencyWeight float64) {
	now := time.Now()
	weekHours := 168.0 // hours in a week

	sort.Slice(items, func(i, j int) bool {
		scoreI := items[i].RelevanceScore * relevanceWeight
		scoreJ := items[j].RelevanceScore * relevanceWeight

		if items[i].PublishedAt != nil {
			hours := now.Sub(*items[i].PublishedAt).Hours()
			scoreI += math.Exp(-hours/weekHours) * recencyWeight
		}
		if items[j].PublishedAt != nil {
			hours := now.Sub(*items[j].PublishedAt).Hours()
			scoreJ += math.Exp(-hours/weekHours) * recencyWeight
		}

		return scoreI > scoreJ
	})
}
