// Example: personalized curation pipeline using taste profiles.
//
// Demonstrates how to:
//   1. Feed behavioral signals (reads, likes, skips)
//   2. Build a taste profile via exponential moving average
//   3. Wrap a mock LLM with PersonalizeScorer for taste-aware scoring
//   4. Run the standard Curator pipeline with personalized results
//   5. Generate an explanation for why content scored the way it did
//
// Run: go run ./examples/personalized-curation
package main

import (
	"context"
	"fmt"
	"time"

	engine "github.com/oluoyefeso/termiflow-engine"
	"github.com/oluoyefeso/termiflow-engine/mock"
	"github.com/oluoyefeso/termiflow-engine/taste"
)

func main() {
	ctx := context.Background()

	// Step 1: Simulate user behavioral signals.
	signals := []taste.Signal{
		{ContentID: "a1", Kind: taste.SignalRead, Value: 1.0, Topics: []string{"go", "concurrency"}, Timestamp: time.Now()},
		{ContentID: "a2", Kind: taste.SignalRead, Value: 0.9, Topics: []string{"go", "performance"}, Timestamp: time.Now()},
		{ContentID: "a3", Kind: taste.SignalLike, Value: 1.0, Topics: []string{"distributed-systems"}, Timestamp: time.Now()},
		{ContentID: "a4", Kind: taste.SignalSkip, Value: 1.0, Topics: []string{"javascript", "react"}, Timestamp: time.Now()},
		{ContentID: "a5", Kind: taste.SignalDislike, Value: 1.0, Topics: []string{"javascript"}, Timestamp: time.Now()},
		{ContentID: "a6", Kind: taste.SignalRead, Value: 0.7, Topics: []string{"rust", "performance"}, Timestamp: time.Now()},
		{ContentID: "a7", Kind: taste.SignalDwell, Value: 0.85, Topics: []string{"go", "testing"}, Timestamp: time.Now()},
		{ContentID: "a8", Kind: taste.SignalRead, Value: 1.0, Topics: []string{"databases", "sqlite"}, Timestamp: time.Now()},
	}

	// Step 2: Build a taste profile from signals.
	profile := taste.UpdateProfile(nil, signals)
	profile.UserID = "demo-user"

	fmt.Println("=== Taste Profile ===")
	fmt.Printf("User: %s | Signals: %d\n", profile.UserID, profile.SignalCount)
	fmt.Println("Affinities:")
	for topic, weight := range profile.Affinities {
		fmt.Printf("  %-25s %.3f\n", topic, weight)
	}
	fmt.Println()

	// Step 3: Wrap the LLM provider with PersonalizeScorer.
	baseLLM := mock.LLM()
	personalLLM := taste.PersonalizeScorer(baseLLM, profile)

	// Step 4: Run the standard Curator pipeline.
	curator := engine.NewCurator(personalLLM)
	now := time.Now()
	results := []engine.SearchResult{
		{Title: "Go 1.23 Concurrency Improvements", URL: "https://go.dev/blog/go1.23", Snippet: "Major concurrency improvements in Go 1.23.", PublishedAt: now.Add(-2 * time.Hour), Source: "go.dev"},
		{Title: "React Server Components Deep Dive", URL: "https://react.dev/blog/rsc", Snippet: "Understanding React Server Components.", PublishedAt: now.Add(-3 * time.Hour), Source: "react.dev"},
		{Title: "SQLite WAL Mode Performance", URL: "https://sqlite.org/wal.html", Snippet: "How WAL mode improves SQLite performance.", PublishedAt: now.Add(-6 * time.Hour), Source: "sqlite.org"},
		{Title: "Distributed Systems Reading List", URL: "https://dsrl.dev", Snippet: "Essential papers for distributed systems.", PublishedAt: now.Add(-12 * time.Hour), Source: "dsrl.dev"},
		{Title: "JavaScript Framework Fatigue", URL: "https://jsfatigue.dev", Snippet: "Another day, another JavaScript framework.", PublishedAt: now.Add(-24 * time.Hour), Source: "jsfatigue.dev"},
	}

	items, err := curator.Curate(ctx, "technology news", results)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("=== Curated Results ===")
	for i, item := range items {
		fmt.Printf("%d. %s (score: %.2f)\n", i+1, item.Title, item.RelevanceScore)
		if item.Summary != "" {
			fmt.Printf("   Summary: %s\n", item.Summary)
		}
	}
	fmt.Println()

	// Step 5: Explain the top result's score.
	if len(items) > 0 {
		explanation, err := taste.Explain(ctx, baseLLM, profile, items[0].Title, items[0].RelevanceScore)
		if err != nil {
			fmt.Printf("Explain error: %v\n", err)
		} else {
			fmt.Println("=== Why This Scored High ===")
			fmt.Println(explanation)
		}
	}
}
