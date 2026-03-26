// Example: custom scoring prompt and configuration.
// Run: go run ./examples/custom-scoring
package main

import (
	"context"
	"fmt"
	"time"

	engine "github.com/oluoyefeso/termiflow-engine"
	"github.com/oluoyefeso/termiflow-engine/mock"
)

func main() {
	// Custom scoring prompt that evaluates novelty instead of relevance
	noveltyPrompt := `Rate how novel and surprising this content is on a scale of 0.0 to 1.0.

Topic: %s
Title: %s
Snippet: %s

0.0-0.3: Well-known, nothing new
0.4-0.6: Some new information
0.7-1.0: Highly novel, breaking news

Respond with only a number.`

	curator := engine.NewCurator(
		mock.LLM(mock.WithScore(0.9)), // Mock always returns high novelty
		engine.WithScoringPrompt(noveltyPrompt),
		engine.WithRelevanceThreshold(0.3),  // Lower threshold for novelty scoring
		engine.WithMaxConcurrency(3),         // Fewer concurrent calls
	)

	results := []engine.SearchResult{
		{Title: "Breakthrough in Quantum Computing", URL: "https://quantum.dev", Snippet: "First fault-tolerant quantum computer", PublishedAt: time.Now().Add(-1 * time.Hour)},
		{Title: "Go Turns 17", URL: "https://go.dev/17", Snippet: "Celebrating 17 years of Go", PublishedAt: time.Now().Add(-3 * time.Hour)},
	}

	items, err := curator.Curate(context.Background(), "technology", results)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Novelty-scored %d items:\n\n", len(items))
	for _, item := range items {
		fmt.Printf("  %s — novelty: %.2f\n", item.Title, item.RelevanceScore)
	}
}
