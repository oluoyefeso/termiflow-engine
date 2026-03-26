// Package mock provides deterministic test providers for termiflow-engine.
//
// Usage:
//
//	curator := engine.NewCurator(mock.LLM())
//	items, _ := curator.Curate(ctx, "rust", results)
package mock

import (
	"context"
	"fmt"
	"time"

	engine "github.com/oluoyefeso/termiflow-engine"
)

// LLMOption configures a mock LLM provider.
type LLMOption func(*mockLLM)

// WithScore sets the fixed relevance score returned by the mock LLM.
func WithScore(score float64) LLMOption {
	return func(m *mockLLM) { m.score = score }
}

// WithSummary sets the fixed summary returned by the mock LLM.
func WithSummary(summary string) LLMOption {
	return func(m *mockLLM) { m.summary = summary }
}

// WithTags sets the fixed tags returned by the mock LLM.
func WithTags(tags string) LLMOption {
	return func(m *mockLLM) { m.tags = tags }
}

// WithAnswer sets the fixed answer returned by the mock LLM for Ask calls.
func WithAnswer(answer string) LLMOption {
	return func(m *mockLLM) { m.answer = answer }
}

type mockLLM struct {
	score   float64
	summary string
	tags    string
	answer  string
	calls   int
}

// LLM returns a deterministic LLMProvider for testing.
// Default behavior: score=0.8, summary="This is a test summary of the article.",
// tags="go, testing", answer="Based on the sources, here is the answer."
func LLM(opts ...LLMOption) engine.LLMProvider {
	m := &mockLLM{
		score:   0.8,
		summary: "This is a test summary of the article.",
		tags:    "go, testing",
		answer:  "Based on the sources, here is the answer.",
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *mockLLM) Complete(_ context.Context, req engine.CompletionRequest) (*engine.CompletionResponse, error) {
	m.calls++

	// Detect what kind of call this is by inspecting the prompt
	if len(req.Messages) > 0 {
		content := req.Messages[len(req.Messages)-1].Content

		// Scoring call: short MaxTokens
		if req.MaxTokens <= 10 {
			return &engine.CompletionResponse{
				Content: fmt.Sprintf("%.1f", m.score),
			}, nil
		}

		// Tag extraction: medium MaxTokens
		if req.MaxTokens <= 50 {
			return &engine.CompletionResponse{
				Content: m.tags,
			}, nil
		}

		// Summary: MaxTokens <= 200
		if req.MaxTokens <= 200 {
			return &engine.CompletionResponse{
				Content: m.summary,
			}, nil
		}

		// Ask/general: everything else
		_ = content
		return &engine.CompletionResponse{
			Content: m.answer,
		}, nil
	}

	return &engine.CompletionResponse{Content: ""}, nil
}

// SearchOption configures a mock Search provider.
type SearchOption func(*mockSearch)

// WithResults sets custom search results.
func WithResults(results []engine.SearchResult) SearchOption {
	return func(m *mockSearch) { m.results = results }
}

type mockSearch struct {
	results []engine.SearchResult
}

// Search returns a deterministic SearchProvider for testing.
// Default behavior: returns 5 hardcoded tech news results.
func Search(opts ...SearchOption) engine.SearchProvider {
	now := time.Now()
	m := &mockSearch{
		results: []engine.SearchResult{
			{Title: "Go 1.23 Released with New Features", URL: "https://go.dev/blog/go1.23", Snippet: "Go 1.23 introduces range-over-func and other improvements.", PublishedAt: now.Add(-2 * time.Hour), Source: "go.dev"},
			{Title: "Rust vs Go Performance Comparison 2026", URL: "https://benchmark.dev/rust-vs-go", Snippet: "A comprehensive performance comparison of Rust and Go for server workloads.", PublishedAt: now.Add(-6 * time.Hour), Source: "benchmark.dev"},
			{Title: "Building AI Pipelines in Go", URL: "https://blog.example.com/ai-pipelines-go", Snippet: "How to build efficient AI content processing pipelines using Go.", PublishedAt: now.Add(-12 * time.Hour), Source: "blog.example.com"},
			{Title: "SQLite Best Practices for CLI Tools", URL: "https://sqlite.org/cli-best-practices", Snippet: "Optimizing SQLite for command-line applications with WAL mode.", PublishedAt: now.Add(-24 * time.Hour), Source: "sqlite.org"},
			{Title: "Terminal UI Frameworks Compared", URL: "https://tui.dev/frameworks", Snippet: "Comparing Bubble Tea, tview, and tcell for terminal UIs.", PublishedAt: now.Add(-48 * time.Hour), Source: "tui.dev"},
		},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *mockSearch) Search(_ context.Context, req engine.SearchRequest) ([]engine.SearchResult, error) {
	max := req.MaxResults
	if max <= 0 || max > len(m.results) {
		max = len(m.results)
	}
	return m.results[:max], nil
}
