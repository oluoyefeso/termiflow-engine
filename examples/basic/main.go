// Example: basic curation pipeline using mock providers.
// Run: go run ./examples/basic
package main

import (
	"context"
	"fmt"
	"time"

	engine "github.com/oluoyefeso/termiflow-engine"
	"github.com/oluoyefeso/termiflow-engine/mock"
)

func main() {
	// Create a curator with the mock LLM provider
	curator := engine.NewCurator(mock.LLM())

	// Simulate search results
	results := []engine.SearchResult{
		{Title: "Go 1.23 Released", URL: "https://go.dev/blog/go1.23", Snippet: "New features in Go 1.23", Content: "Go 1.23 introduces range-over-func iterators...", PublishedAt: time.Now().Add(-2 * time.Hour), Source: "go.dev"},
		{Title: "Cooking Recipes 2026", URL: "https://recipes.com/best", Snippet: "Best recipes of 2026", Content: "The top cooking trends this year...", PublishedAt: time.Now().Add(-5 * time.Hour), Source: "recipes.com"},
		{Title: "Rust Memory Safety", URL: "https://rust-lang.org/safety", Snippet: "How Rust prevents memory bugs", Content: "Rust's ownership model ensures...", PublishedAt: time.Now().Add(-8 * time.Hour), Source: "rust-lang.org"},
	}

	// Run the curation pipeline
	items, err := curator.Curate(context.Background(), "go programming", results)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Curated %d items from %d search results:\n\n", len(items), len(results))
	for i, item := range items {
		fmt.Printf("[%d] %s (score: %.2f)\n", i+1, item.Title, item.RelevanceScore)
		if item.Summary != "" {
			fmt.Printf("    Summary: %s\n", item.Summary)
		}
		if len(item.Tags) > 0 {
			fmt.Printf("    Tags: %v\n", item.Tags)
		}
		fmt.Println()
	}
}
