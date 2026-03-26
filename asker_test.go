package engine

import (
	"context"
	"testing"
)

func TestAsk_WithSources(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "Go 1.23 has iterators."}, nil
		},
	}
	search := &testSearch{
		results: []SearchResult{
			{Title: "Go Blog", URL: "https://go.dev", Snippet: "Go 1.23 released"},
		},
	}

	result, err := Ask(context.Background(), "What's new in Go?", llm, search, 3)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.Answer != "Go 1.23 has iterators." {
		t.Errorf("answer = %q", result.Answer)
	}
	if len(result.Sources) != 1 {
		t.Errorf("sources = %d, want 1", len(result.Sources))
	}
}

func TestAsk_NilSearchProvider(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "I don't have sources."}, nil
		},
	}

	result, err := Ask(context.Background(), "question", llm, nil, 3)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Sources) != 0 {
		t.Errorf("expected 0 sources with nil provider, got %d", len(result.Sources))
	}
}

func TestAsk_MaxSourcesZero(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "answer"}, nil
		},
	}
	search := &testSearch{
		results: []SearchResult{{Title: "Result", URL: "https://a.com"}},
	}

	result, err := Ask(context.Background(), "question", llm, search, 0)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(result.Sources) != 0 {
		t.Errorf("maxSources=0 should skip search, got %d sources", len(result.Sources))
	}
}

func TestBuildPromptWithSources_ManySourcesNumbering(t *testing.T) {
	sources := make([]SearchResult, 12)
	for i := range sources {
		sources[i] = SearchResult{Title: "Source", URL: "https://example.com"}
	}

	prompt := buildPromptWithSources("question", sources)

	// Verify source 10, 11, 12 are numbered correctly (not rune arithmetic)
	if !contains(prompt, "Source 10:") {
		t.Error("source 10 should be numbered correctly")
	}
	if !contains(prompt, "Source 12:") {
		t.Error("source 12 should be numbered correctly")
	}
}

func TestAskStream_NonStreamProvider(t *testing.T) {
	llm := &testLLM{
		completeFunc: func(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Content: "fallback answer"}, nil
		},
	}

	ch, sources, err := AskStream(context.Background(), "question", llm, nil, 0)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if sources != nil && len(sources) != 0 {
		t.Errorf("expected empty sources")
	}

	chunk := <-ch
	if chunk.Content != "fallback answer" {
		t.Errorf("chunk = %q, want fallback answer", chunk.Content)
	}
	if !chunk.Done {
		t.Error("expected Done=true for non-stream fallback")
	}
}

func TestAskStream_WithStreamProvider(t *testing.T) {
	llm := &testStreamLLM{
		chunks: []StreamChunk{
			{Content: "Hello ", Done: false},
			{Content: "world!", Done: true},
		},
	}

	ch, _, err := AskStream(context.Background(), "question", llm, nil, 0)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	var result string
	for chunk := range ch {
		result += chunk.Content
	}
	if result != "Hello world!" {
		t.Errorf("streamed result = %q, want %q", result, "Hello world!")
	}
}

// Test helpers

type testSearch struct {
	results []SearchResult
}

func (s *testSearch) Search(_ context.Context, req SearchRequest) ([]SearchResult, error) {
	max := req.MaxResults
	if max <= 0 || max > len(s.results) {
		max = len(s.results)
	}
	return s.results[:max], nil
}

type testStreamLLM struct {
	chunks []StreamChunk
}

func (s *testStreamLLM) Complete(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
	var content string
	for _, c := range s.chunks {
		content += c.Content
	}
	return &CompletionResponse{Content: content}, nil
}

func (s *testStreamLLM) Stream(_ context.Context, _ CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, len(s.chunks))
	for _, c := range s.chunks {
		ch <- c
	}
	close(ch)
	return ch, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
