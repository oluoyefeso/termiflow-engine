package engine

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkScoreRelevance(b *testing.B) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "0.85"}, nil
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ScoreRelevance(context.Background(), llm, "go", "Go 1.23", "new features")
	}
}

func BenchmarkCurate100(b *testing.B) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, req CompletionRequest) (*CompletionResponse, error) {
			if req.MaxTokens <= 10 {
				return &CompletionResponse{Content: "0.8"}, nil
			}
			return &CompletionResponse{Content: "summary text"}, nil
		},
	}
	curator := NewCurator(llm)

	now := time.Now()
	results := make([]SearchResult, 100)
	for i := range results {
		results[i] = SearchResult{
			Title:       fmt.Sprintf("Article %d", i),
			URL:         fmt.Sprintf("https://example.com/%d", i),
			Snippet:     "test content for benchmarking",
			Content:     "This is a longer content string for benchmarking purposes.",
			PublishedAt: now.Add(-time.Duration(i) * time.Hour),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = curator.Curate(context.Background(), "technology", results)
	}
}

func BenchmarkCurateConcurrency(b *testing.B) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "0.3"}, nil // Below threshold, minimal work
		},
	}

	for _, concurrency := range []int{1, 3, 5, 10} {
		b.Run(fmt.Sprintf("concurrency=%d", concurrency), func(b *testing.B) {
			curator := NewCurator(llm, WithMaxConcurrency(concurrency))

			now := time.Now()
			results := make([]SearchResult, 20)
			for i := range results {
				results[i] = SearchResult{
					Title: fmt.Sprintf("Article %d", i), URL: fmt.Sprintf("https://example.com/%d", i),
					Snippet: "test", PublishedAt: now,
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = curator.Curate(context.Background(), "test", results)
			}
		})
	}
}
