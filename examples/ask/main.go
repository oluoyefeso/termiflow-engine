// Example: question-answering with search sources.
// Run: go run ./examples/ask
package main

import (
	"context"
	"fmt"

	engine "github.com/oluoyefeso/termiflow-engine"
	"github.com/oluoyefeso/termiflow-engine/mock"
)

func main() {
	llm := mock.LLM(mock.WithAnswer("Go 1.23 introduced range-over-func iterators, which allow you to use for-range loops with custom iterator functions. This is a significant language change that simplifies many common patterns."))
	search := mock.Search()

	result, err := engine.Ask(
		context.Background(),
		"What are the new features in Go 1.23?",
		llm,
		search,
		3, // max 3 sources
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Answer:")
	fmt.Println(result.Answer)
	fmt.Printf("\nSources (%d):\n", len(result.Sources))
	for i, src := range result.Sources {
		fmt.Printf("  [%d] %s — %s\n", i+1, src.Title, src.URL)
	}
}
