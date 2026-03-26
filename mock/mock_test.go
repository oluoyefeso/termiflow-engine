package mock

import (
	"context"
	"sync"
	"testing"

	engine "github.com/oluoyefeso/termiflow-engine"
)

func TestLLM_DefaultBehavior(t *testing.T) {
	llm := LLM()

	// Scoring call (MaxTokens <= 10)
	resp, err := llm.Complete(context.Background(), engine.CompletionRequest{
		Messages:  []engine.Message{{Role: "user", Content: "score this"}},
		MaxTokens: 10,
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if resp.Content != "0.8" {
		t.Errorf("default score = %q, want 0.8", resp.Content)
	}
}

func TestLLM_WithCustomScore(t *testing.T) {
	llm := LLM(WithScore(0.3))

	resp, err := llm.Complete(context.Background(), engine.CompletionRequest{
		Messages:  []engine.Message{{Role: "user", Content: "score"}},
		MaxTokens: 10,
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if resp.Content != "0.3" {
		t.Errorf("custom score = %q, want 0.3", resp.Content)
	}
}

func TestLLM_ConcurrentSafety(t *testing.T) {
	llm := LLM()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := llm.Complete(context.Background(), engine.CompletionRequest{
				Messages:  []engine.Message{{Role: "user", Content: "test"}},
				MaxTokens: 10,
			})
			if err != nil {
				t.Errorf("concurrent call failed: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestSearch_DefaultResults(t *testing.T) {
	s := Search()

	results, err := s.Search(context.Background(), engine.SearchRequest{
		Query:      "go programming",
		MaxResults: 3,
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(results) != 3 {
		t.Errorf("got %d results, want 3", len(results))
	}
}

func TestSearch_CustomResults(t *testing.T) {
	custom := []engine.SearchResult{
		{Title: "Custom", URL: "https://custom.com"},
	}
	s := Search(WithResults(custom))

	results, err := s.Search(context.Background(), engine.SearchRequest{
		Query:      "test",
		MaxResults: 5,
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
	if results[0].Title != "Custom" {
		t.Errorf("title = %q, want Custom", results[0].Title)
	}
}
