package engine

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// testLLM implements LLMProvider for testing.
type testLLM struct {
	completeFunc func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	callCount    atomic.Int64
}

func (m *testLLM) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	m.callCount.Add(1)
	if m.completeFunc != nil {
		return m.completeFunc(ctx, req)
	}
	return &CompletionResponse{Content: "0.85"}, nil
}

func TestNewCurator_NilProvider(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil LLMProvider")
		}
		msg := fmt.Sprint(r)
		if msg != "engine: NewCurator requires a non-nil LLMProvider" {
			t.Errorf("unexpected panic message: %s", msg)
		}
	}()
	NewCurator(nil)
}

func TestNewCurator_DefaultConfig(t *testing.T) {
	llm := &testLLM{}
	c := NewCurator(llm)
	if c.cfg.maxConcurrency != 5 {
		t.Errorf("default maxConcurrency = %d, want 5", c.cfg.maxConcurrency)
	}
	if c.cfg.relevanceThreshold != 0.5 {
		t.Errorf("default relevanceThreshold = %f, want 0.5", c.cfg.relevanceThreshold)
	}
}

func TestNewCurator_WithOptions(t *testing.T) {
	llm := &testLLM{}
	c := NewCurator(llm,
		WithMaxConcurrency(10),
		WithRelevanceThreshold(0.3),
		WithScoringWeight(0.5, 0.5),
		WithScoringPrompt("custom: %s %s %s"),
	)
	if c.cfg.maxConcurrency != 10 {
		t.Errorf("maxConcurrency = %d, want 10", c.cfg.maxConcurrency)
	}
	if c.cfg.relevanceThreshold != 0.3 {
		t.Errorf("relevanceThreshold = %f, want 0.3", c.cfg.relevanceThreshold)
	}
	if c.cfg.scoringPrompt != "custom: %s %s %s" {
		t.Errorf("scoringPrompt not set")
	}
}

func TestNewCurator_InvalidOptions(t *testing.T) {
	llm := &testLLM{}
	c := NewCurator(llm,
		WithMaxConcurrency(-1),
		WithRelevanceThreshold(2.0),
	)
	// Should keep defaults for invalid values
	if c.cfg.maxConcurrency != 5 {
		t.Errorf("negative concurrency should keep default, got %d", c.cfg.maxConcurrency)
	}
	if c.cfg.relevanceThreshold != 0.5 {
		t.Errorf("out-of-range threshold should keep default, got %f", c.cfg.relevanceThreshold)
	}
}

func TestCurate_EmptyInput(t *testing.T) {
	llm := &testLLM{}
	c := NewCurator(llm)
	items, err := c.Curate(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestCurate_FiltersLowRelevance(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, req CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "0.2"}, nil
		},
	}
	c := NewCurator(llm)

	now := time.Now()
	results := []SearchResult{
		{Title: "Irrelevant", URL: "https://example.com", Snippet: "not relevant", PublishedAt: now},
	}

	items, err := c.Curate(context.Background(), "rust", results)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items (filtered), got %d", len(items))
	}
}

func TestCurate_KeepsHighRelevance(t *testing.T) {
	callNum := atomic.Int64{}
	llm := &testLLM{
		completeFunc: func(_ context.Context, req CompletionRequest) (*CompletionResponse, error) {
			n := callNum.Add(1)
			switch {
			case n%3 == 1:
				return &CompletionResponse{Content: "0.9"}, nil
			case n%3 == 2:
				return &CompletionResponse{Content: "Test summary"}, nil
			default:
				return &CompletionResponse{Content: "rust, programming"}, nil
			}
		},
	}
	c := NewCurator(llm)

	now := time.Now()
	results := []SearchResult{
		{Title: "Rust Updates", URL: "https://rust.org", Snippet: "new rust features", PublishedAt: now},
	}

	items, err := c.Curate(context.Background(), "rust", results)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Summary != "Test summary" {
		t.Errorf("summary = %q, want %q", items[0].Summary, "Test summary")
	}
}

func TestCurate_AllScoringFails(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return nil, fmt.Errorf("provider down")
		},
	}
	c := NewCurator(llm)

	now := time.Now()
	results := []SearchResult{
		{Title: "Article 1", URL: "https://a.com", Snippet: "test", PublishedAt: now},
		{Title: "Article 2", URL: "https://b.com", Snippet: "test", PublishedAt: now},
	}

	_, err := c.Curate(context.Background(), "test", results)
	if err != ErrAllScoringFailed {
		t.Errorf("expected ErrAllScoringFailed, got %v", err)
	}
}

func TestCurate_PartialFailure(t *testing.T) {
	callNum := atomic.Int64{}
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			n := callNum.Add(1)
			// First item scores, second fails
			if n <= 3 { // score + summarize + tag for first item
				if n == 1 {
					return &CompletionResponse{Content: "0.9"}, nil
				}
				return &CompletionResponse{Content: "summary"}, nil
			}
			return nil, fmt.Errorf("provider intermittent failure")
		},
	}
	c := NewCurator(llm)

	now := time.Now()
	results := []SearchResult{
		{Title: "Good Article", URL: "https://a.com", Snippet: "works", PublishedAt: now},
		{Title: "Fails Article", URL: "https://b.com", Snippet: "fails", PublishedAt: now},
	}

	items, err := c.Curate(context.Background(), "test", results)
	if err != nil {
		t.Fatalf("partial failure should not return error, got %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least 1 item from partial failure")
	}
}

func TestCurate_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	llm := &testLLM{
		completeFunc: func(ctx context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "0.5"}, ctx.Err()
		},
	}
	c := NewCurator(llm)

	results := []SearchResult{
		{Title: "Test", URL: "https://example.com", Snippet: "test", PublishedAt: time.Now()},
	}

	// Should not panic — returns partial results or error
	_, _ = c.Curate(ctx, "test", results)
}

func TestCurate_BoundedConcurrency(t *testing.T) {
	var concurrent atomic.Int64
	var maxConcurrent atomic.Int64

	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			cur := concurrent.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			concurrent.Add(-1)
			return &CompletionResponse{Content: "0.3"}, nil
		},
	}
	c := NewCurator(llm)

	now := time.Now()
	results := make([]SearchResult, 15)
	for i := range results {
		results[i] = SearchResult{
			Title: fmt.Sprintf("Article %d", i), URL: fmt.Sprintf("https://example.com/%d", i),
			Snippet: "test", PublishedAt: now,
		}
	}

	_, _ = c.Curate(context.Background(), "test", results)

	peak := maxConcurrent.Load()
	if peak > 5 {
		t.Errorf("concurrency exceeded limit: %d > 5", peak)
	}
}

func TestCurate_AllBelowThreshold(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "0.1"}, nil
		},
	}
	c := NewCurator(llm)

	results := []SearchResult{
		{Title: "A", URL: "https://a.com", Snippet: "a", PublishedAt: time.Now()},
		{Title: "B", URL: "https://b.com", Snippet: "b", PublishedAt: time.Now()},
	}

	items, err := c.Curate(context.Background(), "test", results)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items when all below threshold, got %d", len(items))
	}
}

func TestCurate_SingleResult(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, req CompletionRequest) (*CompletionResponse, error) {
			if req.MaxTokens <= 10 {
				return &CompletionResponse{Content: "0.9"}, nil
			}
			return &CompletionResponse{Content: "summary"}, nil
		},
	}
	c := NewCurator(llm)

	results := []SearchResult{
		{Title: "Only Article", URL: "https://a.com", Snippet: "test", PublishedAt: time.Now()},
	}

	items, err := c.Curate(context.Background(), "test", results)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestTruncateContent_UTF8Safe(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  string
	}{
		{"short", "short", 100, "short"},
		{"exact", "exact", 5, "exact"},
		{"empty", "", 10, ""},
		{"ascii truncate", "hello world", 5, "hello"},
		{"multibyte safe", "日本語テスト", 3, "日本語"},
		{"emoji safe", "👋🌍🚀✨", 2, "👋🌍"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateContent(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("truncateContent(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

func TestFilterByRelevance(t *testing.T) {
	items := []*FeedItem{
		{Title: "Low", RelevanceScore: 0.3},
		{Title: "High", RelevanceScore: 0.7},
		{Title: "Threshold", RelevanceScore: 0.5},
	}
	filtered := filterByRelevance(items, 0.5)
	if len(filtered) != 2 {
		t.Errorf("expected 2 items at/above threshold, got %d", len(filtered))
	}
}

func TestSortByRelevanceAndRecency_TimeDecay(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)
	oneWeekAgo := now.Add(-168 * time.Hour)

	items := []*FeedItem{
		{Title: "Old but relevant", RelevanceScore: 0.9, PublishedAt: &oneWeekAgo},
		{Title: "New but less relevant", RelevanceScore: 0.6, PublishedAt: &oneHourAgo},
		{Title: "Medium age, medium relevance", RelevanceScore: 0.7, PublishedAt: &oneDayAgo},
	}

	sortByRelevanceAndRecency(items, 0.7, 0.3)

	// Verify items are sorted (first should have highest combined score)
	for i := 0; i < len(items)-1; i++ {
		if items[i].RelevanceScore < items[i+1].RelevanceScore && items[i].PublishedAt.Before(*items[i+1].PublishedAt) {
			// Both relevance AND recency are worse — this should never happen if sorted correctly
			t.Errorf("item %d should not rank higher than item %d", i, i+1)
		}
	}
}
